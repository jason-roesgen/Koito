package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/db/sqlite"
	"github.com/gabehf/koito/internal/memkv"
)

func statsRequest(t *testing.T) (*http.Request, string) {
	t.Helper()
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	params := url.Values{"period": {"all_time"}, "tz": {"UTC"}, "test": {t.Name()}}
	key := "stats_" + params.Encode() + "_UTC"
	memkv.Store.Delete(key)
	t.Cleanup(func() { memkv.Store.Delete(key) })
	return httptest.NewRequest(http.MethodGet, "/stats?"+params.Encode(), nil), key
}

func readStats(t *testing.T, handler http.HandlerFunc, request *http.Request) StatsResponse {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: HTTP %d %q: %s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	if !json.Valid(response.Body.Bytes()) {
		t.Fatalf("invalid JSON: %q", response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"listen_count", "track_count", "album_count", "artist_count", "minutes_listened", "days_active", "longest_streak", "avg_daily_plays", "tracks_per_artist", "albums_per_artist"} {
		if _, ok := fields[field]; !ok {
			t.Errorf("missing API field %s", field)
		}
	}
	if len(fields) != 10 {
		t.Fatalf("changed response fields: %v", fields)
	}
	var result StatsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float32{result.AvgDailyPlays, result.TracksPerArtist, result.AlbumsPerArtist} {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatalf("non-finite average: %v", result)
		}
	}
	return result
}

func TestStatsHandlerEmptyToPopulated(t *testing.T) {
	request, key := statsRequest(t)
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close(ctx) })
	handler := StatsHandler(store)
	if empty := readStats(t, handler, request); empty != (StatsResponse{}) {
		t.Fatalf("nonzero empty stats: %+v", empty)
	}
	if _, ok := memkv.Store.Get(key); ok {
		t.Fatal("statistics response was cached")
	}

	for _, query := range []string{
		"INSERT INTO users(id,username,password) VALUES(1,'test',X'00')",
		"INSERT INTO artists(id) VALUES(1)",
		"INSERT INTO releases(id) VALUES(1)",
		"INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180)",
		"INSERT INTO artist_tracks(artist_id,track_id) VALUES(1,1)",
	} {
		if err := store.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month(), now.Day()-1, 12, 0, 0, 0, time.UTC)
	for count := int64(1); count <= 2; count++ {
		if err := store.SaveListen(ctx, db.SaveListenOpts{TrackID: 1, UserID: 1, Time: first.Add(time.Duration(count) * time.Second)}); err != nil {
			t.Fatal(err)
		}
		result := readStats(t, handler, request)
		if result.ListenCount != count || result.TrackCount != 1 || result.AlbumCount != 1 || result.ArtistCount != 1 || result.MinutesListened != count*3 || result.DaysActive != 1 || result.AvgDailyPlays != float32(count) || result.TracksPerArtist != 1 || result.AlbumsPerArtist != 1 {
			t.Fatalf("incorrect stats after %d ingested listens: %+v", count, result)
		}
	}
	if err := store.DeleteListen(ctx, 1, first.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if result := readStats(t, handler, request); result.ListenCount != 1 || result.MinutesListened != 3 {
		t.Fatalf("cached stats survived deletion: %+v", result)
	}
	if err := store.DeleteListen(ctx, 1, first.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if result := readStats(t, handler, request); result != (StatsResponse{}) {
		t.Fatalf("incorrect stats after returning to empty history: %+v", result)
	}
}

// Use real catalog methods so history and metadata mutations exercise the same
// SQLite paths as ingestion and editing, with no production database involved.
func statsCatalog(t *testing.T) (*sqlite.Sqlite, []int32, []int32, []int32) {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close(ctx) })
	if err := store.Exec("INSERT INTO users(id,username,password) VALUES(1,'test',X'00')"); err != nil {
		t.Fatal(err)
	}
	var tracks, albums, artists []int32
	for i, duration := range []int32{180, 600} {
		name := []string{"first", "second"}[i]
		artist, err := store.SaveArtist(ctx, db.SaveArtistOpts{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		album, err := store.SaveAlbum(ctx, db.SaveAlbumOpts{Title: name, ArtistIDs: []int32{artist.ID}})
		if err != nil {
			t.Fatal(err)
		}
		track, err := store.SaveTrack(ctx, db.SaveTrackOpts{Title: name, AlbumID: album.ID, ArtistIDs: []int32{artist.ID}, Duration: duration})
		if err != nil {
			t.Fatal(err)
		}
		tracks, albums, artists = append(tracks, track.ID), append(albums, album.ID), append(artists, artist.ID)
	}
	return store, tracks, albums, artists
}

func TestStatsHandlerSameCountReplacement(t *testing.T) {
	request, _ := statsRequest(t)
	store, tracks, _, _ := statsCatalog(t)
	ctx := context.Background()
	at := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	handler := StatsHandler(store)
	if err := store.SaveListen(ctx, db.SaveListenOpts{TrackID: tracks[0], UserID: 1, Time: at}); err != nil {
		t.Fatal(err)
	}
	if before := readStats(t, handler, request); before.ListenCount != 1 || before.MinutesListened != 3 {
		t.Fatalf("incorrect initial stats: %+v", before)
	}
	if err := store.DeleteListen(ctx, tracks[0], at); err != nil {
		t.Fatal(err)
	}
	// Deliberately do not request statistics between these mutations.
	if err := store.SaveListen(ctx, db.SaveListenOpts{TrackID: tracks[1], UserID: 1, Time: at}); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 1 || after.MinutesListened != 10 {
		t.Fatalf("same-count replacement returned stale stats: %+v", after)
	}
}

func TestStatsHandlerSameCountCatalogChanges(t *testing.T) {
	request, _ := statsRequest(t)
	store, tracks, albums, artists := statsCatalog(t)
	ctx := context.Background()
	at := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	handler := StatsHandler(store)
	for i := range 2 {
		if err := store.SaveListen(ctx, db.SaveListenOpts{TrackID: tracks[0], UserID: 1, Time: at.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	before := readStats(t, handler, request)
	if before.ListenCount != 2 || before.TrackCount != 1 || before.AlbumCount != 1 || before.ArtistCount != 1 {
		t.Fatalf("incorrect initial counts: %+v", before)
	}
	if err := store.DeleteListen(ctx, tracks[0], at); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveListen(ctx, db.SaveListenOpts{TrackID: tracks[1], UserID: 1, Time: at}); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 2 || after.TrackCount != 2 || after.AlbumCount != 2 || after.ArtistCount != 2 || after.MinutesListened != 13 {
		t.Fatalf("same-count replacement returned stale catalog counts: %+v", after)
	}
	// Metadata-only changes must also be visible without touching the listens.
	if err := store.UpdateTrack(ctx, db.UpdateTrackOpts{ID: tracks[0], Duration: 300}); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 2 || after.MinutesListened != 15 {
		t.Fatalf("duration update returned stale stats: %+v", after)
	}
	// Apply the same reassignment as an album merge directly: MergeAlbums
	// currently opens a second connection while holding a transaction, which
	// cannot run with NewInMemory's single-connection pool.
	if err := store.Exec("UPDATE tracks SET release_id = ? WHERE release_id = ?", albums[1], albums[0]); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 2 || after.AlbumCount != 1 {
		t.Fatalf("album merge returned stale stats: %+v", after)
	}
	if err := store.MergeArtists(ctx, artists[0], artists[1], false); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 2 || after.ArtistCount != 1 || after.TracksPerArtist != 2 {
		t.Fatalf("artist merge returned stale stats: %+v", after)
	}
	if err := store.MergeTracks(ctx, tracks[0], tracks[1]); err != nil {
		t.Fatal(err)
	}
	if after := readStats(t, handler, request); after.ListenCount != 2 || after.TrackCount != 1 || after.MinutesListened != 20 || after.TracksPerArtist != 1 {
		t.Fatalf("track merge returned stale stats: %+v", after)
	}
}

// Embed the store contract; tests implement the methods used by StatsHandler.
type statsStoreStub struct {
	statsStore
	listens, tracks, albums, artists int64
	activeDays                       int
	countCalls, detailCalls          int
	countErr                         error
}

func (s *statsStoreStub) CountListens(context.Context, db.Timeframe) (int64, error) {
	s.countCalls++
	return s.listens, s.countErr
}
func (s *statsStoreStub) CountTracks(context.Context, db.Timeframe) (int64, error) {
	s.detailCalls++
	return s.tracks, nil
}
func (s *statsStoreStub) CountAlbums(context.Context, db.Timeframe) (int64, error) {
	return s.albums, nil
}
func (s *statsStoreStub) CountArtists(context.Context, db.Timeframe) (int64, error) {
	return s.artists, nil
}
func (s *statsStoreStub) CountTimeListened(context.Context, db.Timeframe) (int64, error) {
	return 0, nil
}
func (s *statsStoreStub) GetActiveDays(context.Context, *time.Location) (int, error) {
	return s.activeDays, nil
}
func (s *statsStoreStub) GetLongestListenStreak(context.Context, db.ListenActivityOpts) (int, error) {
	return 0, nil
}

func TestStatsHandlerZeroDenominators(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		artists               int64
		days                  int
		daily, tracks, albums float32
	}{
		{"zero artists", 0, 1, 2, 0, 0},
		{"zero active days", 1, 0, 0, 2, 1},
		{"both zero", 0, 0, 0, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request, key := statsRequest(t)
			store := &statsStoreStub{listens: 2, tracks: 2, albums: 1, artists: tt.artists, activeDays: tt.days}
			result := readStats(t, StatsHandler(store), request)
			if result.AvgDailyPlays != tt.daily || result.TracksPerArtist != tt.tracks || result.AlbumsPerArtist != tt.albums {
				t.Fatalf("incorrect averages: %+v", result)
			}
			if _, ok := memkv.Store.Get(key); ok {
				t.Fatal("statistics response was cached")
			}
		})
	}
}

func TestStatsHandlerIgnoresLegacyCache(t *testing.T) {
	for _, tt := range []struct {
		name   string
		cached *StatsResponse
	}{
		{"NaN", &StatsResponse{ListenCount: 1, AvgDailyPlays: float32(math.NaN())}},
		{"positive infinity", &StatsResponse{ListenCount: 1, TracksPerArtist: float32(math.Inf(1))}},
		{"negative infinity", &StatsResponse{ListenCount: 1, AlbumsPerArtist: float32(math.Inf(-1))}},
		{"nil", nil},
		{"stale empty", &StatsResponse{}},
		{"stale finite", &StatsResponse{ListenCount: 1, MinutesListened: 999, AvgDailyPlays: 999}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request, key := statsRequest(t)
			memkv.Store.Set(key, tt.cached)
			store := &statsStoreStub{listens: 1, tracks: 1, albums: 1, artists: 1, activeDays: 1}
			handler := StatsHandler(store)
			for range 2 {
				if result := readStats(t, handler, request); result.ListenCount != 1 || result.AvgDailyPlays != 1 || result.MinutesListened != 0 {
					t.Fatalf("legacy cache was used: %+v", result)
				}
			}
			// Seeded entries may expire normally; the handler neither reads nor
			// writes this cache, even when it contains invalid legacy data.
			if cached, ok := memkv.Store.Get(key); !ok || cached != tt.cached || store.detailCalls != 2 {
				t.Fatal("handler accessed the statistics response cache")
			}
		})
	}
}

func TestStatsHandlerDoesNotCache(t *testing.T) {
	request, key := statsRequest(t)
	store := &statsStoreStub{listens: 1, tracks: 1, albums: 1, artists: 1, activeDays: 1}
	handler := StatsHandler(store)
	first := readStats(t, handler, request)
	second := readStats(t, handler, request)
	if first != second || store.countCalls != 2 || store.detailCalls != 2 {
		t.Fatalf("statistics were not recomputed: count=%d detail=%d", store.countCalls, store.detailCalls)
	}
	if _, ok := memkv.Store.Get(key); ok {
		t.Fatal("statistics response was cached")
	}
}

func TestStatsHandlerCountError(t *testing.T) {
	request, key := statsRequest(t)
	store := &statsStoreStub{countErr: errors.New("database unavailable")}
	response := httptest.NewRecorder()
	StatsHandler(store).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || !json.Valid(response.Body.Bytes()) {
		t.Fatalf("unexpected error response: %d %q", response.Code, response.Body.String())
	}
	if _, ok := memkv.Store.Get(key); ok {
		t.Fatal("failed result was cached")
	}
}
