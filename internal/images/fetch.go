package images

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// Check the actual address selected by the dialer, including on redirected requests.
var publicImageClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
		peer, err := netip.ParseAddrPort(address)
		if err != nil || !publicImageAddress(peer.Addr()) {
			return errors.New("image destination must be public")
		}
		return nil
	}}).DialContext},
}

func publicImageAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, block := range nonPublicImageNetworks {
		if block.Contains(ip) {
			return false
		}
	}
	return ip.Is4() || globalIPv6.Contains(ip)
}

var globalIPv6 = netip.MustParsePrefix("2000::/3")
var nonPublicImageNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2002::/16"),
}

var subsonicImageClient = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// A saved cover ID can only address the configured server's getCoverArt endpoint.
const coverPrefix = "subsonic:"

func coverID(source string) string {
	if id, ok := strings.CutPrefix(source, coverPrefix); ok {
		return id
	}
	c := imgsrc.subsonicC
	if c == nil {
		return ""
	}
	u, err := url.Parse(source)
	base, baseErr := url.Parse(c.url)
	if err != nil || baseErr != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil {
		return ""
	}
	path := strings.TrimSuffix(u.Path, ".view")
	if path != strings.TrimRight(base.Path, "/")+"/rest/getCoverArt" {
		return ""
	}
	return u.Query().Get("id")
}

func FetchImage(method, source string, allowSavedCover bool) (*http.Response, error) {
	client := publicImageClient
	if allowSavedCover {
		if id := coverID(source); id != "" {
			c := imgsrc.subsonicC
			if c == nil {
				return nil, errors.New("Subsonic image provider is disabled")
			}
			source = strings.TrimRight(c.url, "/") + "/rest/getCoverArt?" + c.authParams + "&id=" + url.QueryEscape(id) + "&v=1.13.0&c=koito"
			client = subsonicImageClient
		}
	}
	req, err := http.NewRequest(method, source, nil)
	if err != nil || req.URL.User != nil || (req.URL.Scheme != "http" && req.URL.Scheme != "https") {
		return nil, errors.New("invalid image URL")
	}
	if client == publicImageClient {
		// A legacy URL can still contain Subsonic credentials after the operator changes servers.
		q := req.URL.Query()
		if q.Has("p") || (q.Has("u") && (q.Has("t") || q.Has("s"))) {
			return nil, errors.New("credentialed cover URL requires its configured Subsonic provider")
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("could not fetch image from the requested destination")
	}
	return resp, nil
}
