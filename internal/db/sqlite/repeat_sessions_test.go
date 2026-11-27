package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/stretchr/testify/require"
)

func repeatFixture(t testing.TB) *Sqlite {
	t.Helper()
	s := hallFixture(t)
	require.NoError(t, s.Exec("INSERT INTO users(id,username,password) VALUES(2,'other','test')"))
	require.NoError(t, s.Exec("INSERT INTO tracks(id,release_id,duration) VALUES(1,1,240),(2,1,120),(3,1,0)"))
	require.NoError(t, s.Exec("INSERT INTO track_aliases VALUES(1,'Long track','test',1),(2,'Interruption','test',1),(3,'Unknown duration','test',1)"))
	require.NoError(t, s.Exec("INSERT INTO artist_tracks VALUES(1,1,1),(2,1,0),(2,2,1),(3,3,1)"))
	return s
}

func recordRepeats(t testing.TB, s *Sqlite, track, user int32, at ...int64) {
	t.Helper()
	for _, ts := range at {
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{
			TrackID: track, UserID: user, Time: time.Unix(ts, 0), Client: "import",
		}))
	}
}

func TestRepeatSessionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		seed func(testing.TB, *Sqlite)
		want int64
	}{
		{"interruption survives search", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1240, 1720, 1960)
			recordRepeats(t, s, 2, 1, 1480)
		}, 0},
		{"exact known gap and minimum", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1540, 2080)
		}, 1},
		{"long known gap", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1540, 2081)
		}, 0},
		{"same-second different tracks break sequence", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1240, 1480, 1720, 1960)
			recordRepeats(t, s, 2, 1, 1480)
		}, 0},
		{"interleaved users have independent timelines", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1240, 1480)
			recordRepeats(t, s, 1, 2, 1120, 1360)
			recordRepeats(t, s, 2, 2, 1200)
		}, 1},
		{"users cannot contribute to the same run", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1240)
			recordRepeats(t, s, 1, 2, 1480)
		}, 0},
		{"same-second plays by different users do not collide", func(t testing.TB, s *Sqlite) {
			recordRepeats(t, s, 1, 1, 1000, 1240, 1480)
			recordRepeats(t, s, 2, 2, 1240)
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := repeatFixture(t)
			tc.seed(t, s)
			page, err := s.GetRepeatSessions(context.Background(), db.RepeatSessionOpts{Query: "long"})
			require.NoError(t, err)
			require.Equal(t, tc.want, page.TotalCount)
			if tc.want > 0 {
				require.EqualValues(t, 3, page.Items[0].PlayCount)
				require.EqualValues(t, 720, *page.Items[0].EstimatedSeconds)
				require.Equal(t, "Long track", page.Items[0].Track.Title)
				require.Len(t, page.Items[0].Track.Artists, 2)
			}
		})
	}
}

func TestRepeatSessionsHistoricalCalendarAndUnknownDuration(t *testing.T) {
	s := repeatFixture(t)
	ctx := context.Background()
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	// March 31, 2024 is a 23-hour local date. The first session crosses midnight.
	midnight := time.Date(2024, 3, 31, 0, 0, 0, 0, berlin).Unix()
	nextMidnight := time.Date(2024, 4, 1, 0, 0, 0, 0, berlin).Unix()
	require.EqualValues(t, 23*3600, nextMidnight-midnight)
	// Insert historical listens out of order.
	recordRepeats(t, s, 1, 1, midnight+360, midnight-120, midnight+120)
	recordRepeats(t, s, 3, 1, midnight+2000, midnight+2600, midnight+3200, midnight+3801)
	recordRepeats(t, s, 1, 1, nextMidnight, nextMidnight+240, nextMidnight+480)
	page, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{From: &midnight, To: &nextMidnight})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	item := page.Items[0]
	require.EqualValues(t, 3, item.PlayCount)
	require.Nil(t, item.EstimatedSeconds)
	require.EqualValues(t, 1200, item.ElapsedSeconds)

	previousDay := time.Date(2024, 3, 30, 0, 0, 0, 0, berlin).Unix()
	page, err = s.GetRepeatSessions(ctx, db.RepeatSessionOpts{From: &previousDay, To: &midnight, Query: "Coast"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, midnight-120, page.Items[0].StartedAt.Unix())
	require.Equal(t, midnight+360, page.Items[0].EndedAt.Unix())
	require.EqualValues(t, 480, page.Items[0].ElapsedSeconds)
}

func TestRepeatSessionsStablePagination(t *testing.T) {
	s := repeatFixture(t)
	ctx := context.Background()
	for i := range 23 {
		start := int64(1000 + i*3600)
		recordRepeats(t, s, 1, 1, start, start+240, start+480)
	}
	first, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{})
	require.NoError(t, err)
	require.Len(t, first.Items, 20)
	require.EqualValues(t, 23, first.TotalCount)
	require.True(t, first.HasNextPage)
	last, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Page: 9})
	require.NoError(t, err)
	require.Len(t, last.Items, 3)
	require.EqualValues(t, 2, last.CurrentPage)
	require.False(t, last.HasNextPage)
	items := append(first.Items, last.Items...)
	for i, row := range items {
		require.EqualValues(t, 1000+(22-i)*3600, row.StartedAt.Unix())
	}

	unknown := int64(500)
	recordRepeats(t, s, 3, 2, unknown, unknown+100, unknown+200)
	for _, order := range []string{"asc", "desc"} {
		page, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Sort: "time", Order: order, Page: 2})
		require.NoError(t, err)
		require.Nil(t, page.Items[len(page.Items)-1].EstimatedSeconds)
	}
	empty, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Page: 9, Query: "no result"})
	require.NoError(t, err)
	require.NotNil(t, empty.Items)
	require.Empty(t, empty.Items)
	require.EqualValues(t, 1, empty.CurrentPage)
}

func TestRepeatSessionsOverviewIsAllTime(t *testing.T) {
	s := repeatFixture(t)
	ctx := context.Background()
	empty, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{})
	require.NoError(t, err)
	require.Equal(t, db.RepeatSessionOverview{}, empty.Overview)

	recordRepeats(t, s, 1, 1, 1000, 1240, 1480, 1720)
	recordRepeats(t, s, 3, 1, 4000, 4300, 4600, 4900)
	recordRepeats(t, s, 2, 1, 7000, 7120, 7240)

	from := int64(6000)
	for _, opts := range []db.RepeatSessionOpts{
		{From: &from},
		{Query: "Interruption", Sort: "time", Order: "asc"},
		{Query: "no match", Page: 2},
	} {
		page, err := s.GetRepeatSessions(ctx, opts)
		require.NoError(t, err)
		require.Equal(t, opts.Query != "no match", len(page.Items) == 1)
		require.EqualValues(t, 3, page.Overview.SessionCount)
		require.EqualValues(t, 2, page.Overview.SessionsWithEstimates)
		require.EqualValues(t, 4*240+3*120, page.Overview.EstimatedSeconds)
		longest := page.Overview.LongestSession
		require.NotNil(t, longest)
		require.EqualValues(t, 4, longest.PlayCount)
		require.EqualValues(t, 4000, longest.StartedAt.Unix())
		require.Nil(t, longest.EstimatedSeconds)
		require.Equal(t, "Unknown duration", longest.Track.Title)
		require.Len(t, longest.Track.Artists, 1)
	}
}

func repeatBenchmarkDB(t testing.TB, size int) *Sqlite {
	t.Helper()
	seed := repeatFixture(t)
	// Four-play sessions with gaps, interruptions and two per-user streams.
	require.NoError(t, seed.Exec(`
		WITH RECURSIVE seq(n) AS (VALUES(0) UNION ALL SELECT n+1 FROM seq WHERE n+1 < ?)
		INSERT INTO listens (track_id,listened_at,user_id)
		SELECT 1 + (n / 4) % 3, 946684800 + (n / 4) * 1800 + (n % 4) * 120, 1 + (n / 4) % 2
		FROM seq`, size))
	filename := filepath.Join(t.TempDir(), "repeat.db")
	require.NoError(t, seed.Exec("VACUUM INTO ?", filename))
	conn, err := sql.Open("sqlite", filename+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	s := &Sqlite{db: conn}
	t.Cleanup(func() { s.Close(context.Background()) })
	require.NoError(t, conn.Ping()) // Match production startup: enable WAL before taking concurrent requests.
	return s
}

func BenchmarkRepeatSessions(b *testing.B) {
	for _, size := range []int{100_000, 500_000, 1_000_000} {
		b.Run(fmt.Sprintf("%dk", size/1000), func(b *testing.B) {
			s := repeatBenchmarkDB(b, size)
			for _, tc := range []struct {
				name string
				opts db.RepeatSessionOpts
			}{
				{"first-page", db.RepeatSessionOpts{}},
				{"filtered-time-page-3", db.RepeatSessionOpts{Query: "Coast", Sort: "time", Order: "asc", Page: 3}},
			} {
				b.Run(tc.name, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						got, err := s.GetRepeatSessions(context.Background(), tc.opts)
						require.NoError(b, err)
						require.Len(b, got.Items, 20)
						if tc.opts.Query == "" {
							require.EqualValues(b, size/4, got.TotalCount)
						}
					}
				})
			}
		})
	}
}

func TestRepeatSessionsConcurrentScrobbling(t *testing.T) {
	s := repeatBenchmarkDB(t, 100_000)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	type readResult struct {
		page *db.RepeatSessionPage
		err  error
	}
	read := make(chan readResult, 1)
	go func() {
		page, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Sort: "date"})
		read <- readResult{page, err}
	}()
	require.Eventually(t, func() bool { return s.db.Stats().InUse > 0 }, time.Second, time.Millisecond)
	for i := range 3 {
		require.NoError(t, s.SaveListen(ctx, db.SaveListenOpts{
			TrackID: 1, UserID: 1, Time: start.Add(time.Duration(i) * 240 * time.Second),
		}))
	}
	before := <-read
	require.NoError(t, before.err)
	if slices.ContainsFunc(before.page.Items, func(v db.RepeatSession) bool { return v.StartedAt.Equal(start) }) {
		require.EqualValues(t, 3, before.page.Items[0].PlayCount)
	}
	page, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Sort: "date"})
	require.NoError(t, err)
	require.True(t, start.Equal(page.Items[0].StartedAt))
	require.EqualValues(t, 3, page.Items[0].PlayCount)
}

func TestRepeatSessionsConcurrentReadersAndCancellation(t *testing.T) {
	s := repeatBenchmarkDB(t, 100_000)
	ctx, stop := context.WithTimeout(context.Background(), 15*time.Second)
	defer stop()

	cancelled, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.GetRepeatSessions(cancelled, db.RepeatSessionOpts{})
		done <- err
	}()
	require.Eventually(t, func() bool { return s.db.Stats().InUse > 0 }, 2*time.Second, time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-ctx.Done():
		t.Fatal("cancelled database request did not return")
	}
	require.Eventually(t, func() bool { return s.db.Stats().InUse == 0 }, time.Second, time.Millisecond)

	var readers sync.WaitGroup
	for _, query := range []string{"", "Coast", "unmatched artist", "Long track"} {
		readers.Go(func() {
			page, err := s.GetRepeatSessions(ctx, db.RepeatSessionOpts{Query: query, Page: 2})
			if err != nil {
				t.Errorf("search %q: %v", query, err)
				return
			}
			if page.Overview.SessionCount != 25_000 || page.Overview.LongestSession == nil || page.Overview.LongestSession.PlayCount != 4 {
				t.Errorf("search %q: inconsistent overview: %+v", query, page.Overview)
			}
		})
	}
	readers.Wait()
	require.Zero(t, s.db.Stats().InUse)
}

func BenchmarkRepeatSessionsWithScrobbling(b *testing.B) {
	s := repeatBenchmarkDB(b, 500_000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type writeStats struct {
		count int
		max   time.Duration
		err   error
	}
	done := make(chan writeStats, 1)
	go func() {
		var stats writeStats
		defer func() { done <- stats }()
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		first := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				stats.err = s.SaveListen(context.Background(), db.SaveListenOpts{
					TrackID: 1, UserID: 1, Time: first.Add(time.Duration(stats.count) * time.Second),
				})
				if stats.err != nil {
					return
				}
				stats.max = max(stats.max, time.Since(start))
				stats.count++
			}
		}
	}()
	b.ReportAllocs()
	for b.Loop() {
		_, err := s.GetRepeatSessions(context.Background(), db.RepeatSessionOpts{})
		require.NoError(b, err)
	}
	cancel()
	stats := <-done
	require.NoError(b, stats.err)
	require.Positive(b, stats.count)
	b.ReportMetric(float64(stats.count)/float64(b.N), "writes/op")
	b.ReportMetric(float64(stats.max)/float64(time.Millisecond), "max-write-ms")
}
