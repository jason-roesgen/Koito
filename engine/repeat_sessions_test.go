package engine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/export"
	"github.com/gabehf/koito/internal/importer"
	"github.com/gabehf/koito/internal/models"
	"github.com/stretchr/testify/require"
)

func TestRepeatSessionsAPILoginGate(t *testing.T) {
	login(t)
	getApiKey(t, session)
	cfg.SetLoginGate(true)
	t.Cleanup(func() { cfg.SetLoginGate(false) })

	for _, tc := range []struct {
		name, method, query, auth, cookie string
		status                            int
	}{
		{name: "anonymous", method: "GET", status: http.StatusUnauthorized},
		{name: "session", method: "GET", cookie: session, status: http.StatusOK},
		{name: "scrobbler token", method: "GET", auth: apikey, status: http.StatusOK},
		{name: "invalid token", method: "GET", auth: "invalid", status: http.StatusUnauthorized},
		{name: "read only", method: "POST", cookie: session, status: http.StatusMethodNotAllowed},
		{name: "invalid page", method: "GET", query: "?page=0", cookie: session, status: http.StatusBadRequest},
		{name: "invalid sort", method: "GET", query: "?sort=unknown", cookie: session, status: http.StatusBadRequest},
		{name: "invalid order", method: "GET", query: "?order=unknown", cookie: session, status: http.StatusBadRequest},
		{name: "invalid timestamp", method: "GET", query: "?from=yesterday", cookie: session, status: http.StatusBadRequest},
		{name: "reversed range", method: "GET", query: "?from=20&to=10", cookie: session, status: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, host()+"/apis/web/v1/repeat-sessions"+tc.query, nil)
			require.NoError(t, err)
			if tc.auth != "" {
				req.Header.Set("Authorization", "Token "+tc.auth)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "koito_session", Value: tc.cookie})
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.status, resp.StatusCode)
			if tc.status == http.StatusOK {
				var page db.RepeatSessionPage
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
				require.Equal(t, db.RepeatSessionMinimumPlays, page.MinimumPlays)
				require.NotNil(t, page.Items)
			}
		})
	}
	cfg.SetLoginGate(false)
	resp, err := http.Get(host() + "/apis/web/v1/repeat-sessions")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRepeatSessionsImportedHistory(t *testing.T) {
	store := newTestDB()
	ctx := context.Background()
	t.Cleanup(func() { store.Close(ctx) })
	first := time.Date(2016, 1, 1, 23, 58, 0, 0, time.UTC)
	listen := export.KoitoListen{
		Track: export.KoitoTrack{Duration: 180, Aliases: []models.Alias{{Alias: "Imported track", Primary: true}}},
		Album: export.KoitoAlbum{Aliases: []models.Alias{{Alias: "Imported album", Primary: true}}},
		Artists: []export.KoitoArtist{{
			IsPrimary: true, Aliases: []models.Alias{{Alias: "Imported artist", Primary: true}},
		}},
	}
	data := export.KoitoExport{Version: "1", User: "test", Listens: []export.KoitoListen{}}
	for _, offset := range []time.Duration{6, 0, 3} {
		listen.ListenedAt = first.Add(offset * time.Minute)
		data.Listens = append(data.Listens, listen)
	}
	body, err := json.Marshal(data)
	require.NoError(t, err)
	file := "koito_repeat_sessions_test.json"
	require.NoError(t, os.WriteFile(filepath.Join(cfg.ConfigDir(), "import", file), body, 0o600))
	require.NoError(t, importer.ImportKoitoFile(ctx, store, file))

	page, err := store.GetRepeatSessions(ctx, db.RepeatSessionOpts{Query: "imported artist"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	item := page.Items[0]
	require.Equal(t, "Imported track", item.Track.Title)
	require.Equal(t, first, item.StartedAt)
	require.Equal(t, first.Add(6*time.Minute), item.EndedAt)
	require.EqualValues(t, 3, item.PlayCount)
	require.EqualValues(t, 540, *item.EstimatedSeconds)
	require.EqualValues(t, 360, item.ElapsedSeconds)
}
