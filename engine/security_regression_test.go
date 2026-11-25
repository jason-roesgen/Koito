package engine_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSecurityWebBoundaries(t *testing.T) {
	login(t)
	getApiKey(t, session)

	upstream, err := url.Parse(host())
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	server := httptest.NewServer(proxy)
	t.Cleanup(server.Close)
	do := func(method, endpoint, body string, headers map[string]string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+endpoint, strings.NewReader(body))
		require.NoError(t, err)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	cookie := "koito_session=" + session
	sameOrigin := map[string]string{"Cookie": cookie, "Origin": strings.Replace(server.URL, "http:", "https:", 1)}
	require.Equal(t, http.StatusNoContent, do("POST", "/apis/web/v1/logout", "", map[string]string{"Origin": sameOrigin["Origin"]}).StatusCode)
	require.Equal(t, http.StatusForbidden, do("POST", "/apis/web/v1/logout", "", map[string]string{"Origin": "https://other.example"}).StatusCode)
	require.Equal(t, http.StatusBadRequest, do("PATCH", "/apis/web/v1/user", "{}", sameOrigin).StatusCode)
	for _, tc := range []struct{ size, status int }{
		{10 << 20, http.StatusNoContent},
		{10<<20 + 1, http.StatusBadRequest},
	} {
		const start = `{"username":"","padding":"`
		const end = `"}`
		body := start + strings.Repeat("x", tc.size-len(start)-len(end)) + end
		require.Equal(t, tc.status, do("PATCH", "/apis/web/v1/user", body, sameOrigin).StatusCode)
	}

	keyHeaders := map[string]string{"Authorization": "Token " + apikey}
	for _, endpoint := range []string{"/apis/web/v1/user", "/apis/web/v1/user/apikeys", "/apis/web/v1/export"} {
		require.Equal(t, http.StatusUnauthorized, do("GET", endpoint, "", keyHeaders).StatusCode, endpoint)
	}
	require.Equal(t, http.StatusOK, do("GET", "/apis/listenbrainz/1/validate-token", "", keyHeaders).StatusCode)

	// Invalid JSON is handled normally at the limit, and rejected as too large above it.
	for _, tc := range []struct{ size, status int }{
		{5 << 20, http.StatusBadRequest},
		{5<<20 + 1, http.StatusRequestEntityTooLarge},
	} {
		require.Equal(t, tc.status, do("POST", "/apis/listenbrainz/1/submit-listens", strings.Repeat(" ", tc.size), keyHeaders).StatusCode)
	}

	ctx := context.Background()
	other, err := store.SaveUser(ctx, db.SaveUserOpts{Username: "key-owner-" + uuid.NewString()[:8], Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	key, err := store.SaveApiKey(ctx, db.SaveApiKeyOpts{UserID: other.ID, Key: uuid.NewString()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.DeleteApiKey(ctx, key.ID, other.ID)) })
	resp := do("DELETE", fmt.Sprintf("/apis/web/v1/user/apikeys/%d", key.ID), "", sameOrigin)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, http.StatusOK, do("GET", "/apis/listenbrainz/1/validate-token", "", map[string]string{"Authorization": "Token " + key.Key}).StatusCode)
}
