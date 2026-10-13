package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func TestListeningTimeRankings(t *testing.T) {
	ctx := context.Background()
	s, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(ctx)
	// Detail lookups run while the ranked rows are open; the named memory DB is shared.
	s.db.SetMaxOpenConns(4)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO users(id,username,password) VALUES(1,'test','test')")
	// Six short, frequently played tracks outrank the long track by plays.
	// The long track must enter the top five when ranked by time, before LIMIT.
	for id := 1; id <= 9; id++ {
		duration, plays := 10, 10
		if id == 7 || id == 8 {
			duration, plays = 300, 2
		}
		if id == 9 {
			duration, plays = 0, 20
		}
		exec("INSERT INTO artists(id) VALUES(?)", id)
		exec("INSERT INTO artist_aliases VALUES(?,?,'test',1)", id, fmt.Sprint(id))
		exec("INSERT INTO releases(id) VALUES(?)", id)
		exec("INSERT INTO release_aliases VALUES(?,?,'test',1)", id, fmt.Sprint(id))
		exec("INSERT INTO tracks(id,release_id,duration) VALUES(?,?,?)", id, id, duration)
		exec("INSERT INTO track_aliases VALUES(?,?,'test',1)", id, fmt.Sprint(id))
		exec("INSERT INTO artist_tracks VALUES(?,?,1)", id, id)
		exec("INSERT INTO artist_releases VALUES(?,?,1)", id, id)
		for j := 0; j < plays; j++ {
			exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(?,?,1)", id, 1000+j)
		}
	}
	exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(7,500,1)")
	opts := db.GetItemsOpts{Page: 1, Limit: 5, Timeframe: db.Timeframe{From: time.Unix(1000, 0), To: time.Unix(2000, 0)}}
	for _, byTime := range []bool{false, true} {
		opts.RankByTime = byTime
		want := int32(9)
		if byTime {
			want = 7
		}
		artists, err := s.GetTopArtistsPaginated(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		albums, err := s.GetTopAlbumsPaginated(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		tracks, err := s.GetTopTracksPaginated(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		if artists.Items[0].Item.ID != want || albums.Items[0].Item.ID != want || tracks.Items[0].Item.ID != want {
			t.Fatalf("wrong leader for time=%v", byTime)
		}
		if byTime {
			if artists.Items[0].Item.TimeListened != 600 || albums.Items[0].Item.TimeListened != 600 || tracks.Items[0].Item.TimeListened != 600 {
				t.Fatal("ranking responses must include timeframe-specific listening seconds")
			}
			if tracks.Items[0].Item.ListenCount != 2 || tracks.Items[1].Rank != 1 || tracks.Items[2].Rank != 3 {
				t.Fatal("timeframe, ties or rank incorrect")
			}
			opts.Page = 2
			page, err := s.GetTopTracksPaginated(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != 4 || page.Items[3].Item.ID != 9 {
				t.Fatal("missing durations or pagination incorrect")
			}
			opts.Page = 1
		}
	}
	opts.ArtistID = 7
	if r, err := s.GetTopAlbumsPaginated(ctx, opts); err != nil || len(r.Items) != 1 {
		t.Fatalf("artist albums: %v", err)
	}
	if r, err := s.GetTopTracksPaginated(ctx, opts); err != nil || len(r.Items) != 1 {
		t.Fatalf("artist tracks: %v", err)
	}
	opts.ArtistID = 0
	opts.AlbumID = 7
	if r, err := s.GetTopTracksPaginated(ctx, opts); err != nil || len(r.Items) != 1 {
		t.Fatalf("album tracks: %v", err)
	}
}
