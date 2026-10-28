package sqlite

import (
	"context"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/gabehf/koito/internal/db"
)

func TestHourlyListening(t *testing.T) {
	ctx := context.Background()
	store, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := store.db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO users(id, username, password) VALUES(1, 'test', 'test')")
	exec("INSERT INTO releases(id) VALUES(1)")
	exec("INSERT INTO tracks(id, release_id, duration) VALUES(1, 1, 180), (2, 1, 0)")
	unix := func(value string) int64 {
		t.Helper()
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return at.Unix()
	}
	for _, value := range []string{
		"2026-03-29T00:30:00Z", // Berlin 01:30 before spring DST.
		"2026-03-29T01:30:00Z", // Berlin 03:30 after spring DST.
		"2026-10-25T00:30:00Z", // Both fall timestamps are Berlin 02:30.
		"2026-10-25T01:30:00Z",
	} {
		exec("INSERT INTO listens(track_id, listened_at, user_id) VALUES(1, ?, 1)", unix(value))
	}
	exec("INSERT INTO listens(track_id, listened_at, user_id) VALUES(2, ?, 1)", unix("2026-03-29T01:30:00Z"))
	tests := []struct {
		name, timezone, from, to string
		plays                    map[int]int64
		seconds                  map[int]int64
	}{
		{"spring DST", "Europe/Berlin", "2026-03-29T00:00:00Z", "2026-03-29T02:00:00Z", map[int]int64{1: 1, 3: 2}, map[int]int64{1: 180, 3: 180}},
		{"fall DST", "Europe/Berlin", "2026-10-25T00:00:00Z", "2026-10-25T02:00:00Z", map[int]int64{2: 2}, map[int]int64{2: 360}},
		{"quarter hour timezone", "Asia/Kathmandu", "2026-03-29T00:00:00Z", "2026-03-29T02:00:00Z", map[int]int64{6: 1, 7: 2}, map[int]int64{6: 180, 7: 180}},
		{"empty", "UTC", "2020-01-01T00:00:00Z", "2020-01-02T00:00:00Z", map[int]int64{}, map[int]int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.timezone)
			if err != nil {
				t.Fatal(err)
			}
			buckets, err := store.GetHourlyListening(ctx, db.Timeframe{From: time.Unix(unix(tt.from), 0), To: time.Unix(unix(tt.to), 0), Timezone: loc})
			if err != nil {
				t.Fatal(err)
			}
			if len(buckets) != 24 {
				t.Fatalf("got %d hours", len(buckets))
			}
			for hour, bucket := range buckets {
				if bucket.Hour != hour || bucket.Plays != tt.plays[hour] || bucket.Seconds != tt.seconds[hour] {
					t.Errorf("hour %d: got %+v, want plays=%d seconds=%d", hour, bucket, tt.plays[hour], tt.seconds[hour])
				}
			}
		})
	}
}
