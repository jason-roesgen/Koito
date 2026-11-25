package images

import (
	"net/netip"
	"testing"
)

func TestPublicImageAddress(t *testing.T) {
	for address, want := range map[string]bool{
		"8.8.8.8": true, "2606:4700:4700::1111": true,
		"0.0.0.1": false, "127.0.0.1": false, "10.0.0.1": false, "172.16.1.2": false,
		"169.254.1.1": false, "100.64.0.1": false, "::ffff:192.168.1.1": false,
		"::1": false, "fe80::1": false, "fc00::1": false, "64:ff9b::a00:1": false,
	} {
		if got := publicImageAddress(netip.MustParseAddr(address)); got != want {
			t.Errorf("%s: public=%t", address, got)
		}
	}
}

func TestStoredCoverSources(t *testing.T) {
	prior := imgsrc.subsonicC
	imgsrc.subsonicC = &SubsonicClient{url: "http://music.example/base"}
	t.Cleanup(func() { imgsrc.subsonicC = prior })
	for source, want := range map[string]string{
		"subsonic:cover-1": "cover-1",
		"http://music.example/base/rest/getCoverArt.view?id=2&u=test": "2",
		"http://other.example/base/rest/getCoverArt?id=2":             "",
		"http://music.example/base/rest/ping?id=2":                    "",
	} {
		if got := coverID(source); got != want {
			t.Errorf("%s: got %q", source, got)
		}
	}
}
