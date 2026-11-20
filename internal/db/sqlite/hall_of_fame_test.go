package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func hallFixture(t testing.TB) *Sqlite {
	t.Helper()
	s, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(context.Background()) })
	for _, query := range []string{
		"INSERT INTO users(id,username,password) VALUES(1,'test','test')",
		"INSERT INTO artists(id) VALUES(1),(2),(3)",
		"INSERT INTO artist_aliases VALUES(1,'Amber','test',1),(2,'Coast','test',1),(3,'Uncertified','test',1)",
		"INSERT INTO releases(id) VALUES(1)",
	} {
		if _, err := s.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func hallSong(t testing.TB, s *Sqlite, id int, plays int) {
	t.Helper()
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, q := range []string{fmt.Sprintf("INSERT INTO tracks(id,release_id) VALUES(%d,1)", id), fmt.Sprintf("INSERT INTO track_aliases VALUES(%d,'Song %03d','test',1)", id, id), fmt.Sprintf("INSERT INTO artist_tracks VALUES(1,%d,1)", id)} {
		if _, err := tx.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	stmt, err := tx.Prepare("INSERT INTO listens(track_id,listened_at,user_id) VALUES(?,?,1)")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	for i := 1; i <= plays; i++ {
		if _, err := stmt.Exec(id, 1000+i); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestHallOfFame(t *testing.T) {
	ctx := context.Background()
	s := hallFixture(t)
	get := func(opts db.HallOfFameOpts) *db.HallOfFame {
		t.Helper()
		r, err := s.GetHallOfFame(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	empty := get(db.HallOfFameOpts{View: "artists", Page: 1})
	if empty.CertifiedSongs != 0 || empty.CertifiedArtists != 0 || empty.Songs == nil || empty.Artists == nil {
		t.Fatalf("bad empty result: %+v", empty)
	}
	plays := []int{199, 200, 400, 600, 800, 1000, 1200, 2000}
	for i, n := range plays {
		hallSong(t, s, i+1, n)
	}
	if _, err := s.db.Exec("INSERT INTO artist_tracks VALUES(2,6,0),(3,1,1)"); err != nil {
		t.Fatal(err)
	}
	// Future listens must not confer a certificate or change its date.
	if _, err := s.db.Exec("INSERT INTO listens VALUES(1,?,1,'')", time.Now().AddDate(1, 0, 0).Unix()); err != nil {
		t.Fatal(err)
	}
	all := get(db.HallOfFameOpts{View: "songs", Page: 1})
	if all.CertifiedSongs != 7 || all.CertifiedArtists != 2 || all.Counts["gold"] != 2 || all.Counts["platinum"] != 3 || all.Counts["diamond"] != 2 {
		t.Fatalf("wrong counts: %+v", all)
	}
	for _, song := range all.Songs {
		individual, err := s.GetTrackCertifications(ctx, song.TrackID)
		if err != nil {
			t.Fatal(err)
		}
		if individual.Plays != song.Plays || individual.Current.Tier != song.Current.Tier || individual.Current.Multiple != song.Current.Multiple || individual.Current.Plays != song.Current.Plays || !individual.Current.ReachedAt.Equal(*song.Current.ReachedAt) {
			t.Fatalf("individual/global disagreement: %+v / %+v", song, individual)
		}
	}

	for _, tt := range []struct {
		view, sort, order string
		first             int32
	}{
		{"songs", "count", "asc", 2}, {"songs", "name", "desc", 8},
		{"songs", "date", "asc", 2}, {"songs", "date", "desc", 8},
		{"artists", "count", "asc", 2}, {"artists", "name", "asc", 1},
	} {
		r := get(db.HallOfFameOpts{View: tt.view, Sort: tt.sort, Order: tt.order, Page: 1})
		first := int32(0)
		if tt.view == "songs" {
			first = r.Songs[0].TrackID
		} else {
			first = r.Artists[0].ID
		}
		if first != tt.first {
			t.Fatalf("incorrect sort %+v: first %d", tt, first)
		}
	}
	artists := get(db.HallOfFameOpts{View: "artists", Page: 1})
	if artists.Artists[0].ID != 1 || artists.Artists[0].CertifiedSongs != 7 || artists.Artists[1].CertifiedSongs != 1 {
		t.Fatalf("credit inflated: %+v", artists.Artists)
	}
	filtered := get(db.HallOfFameOpts{View: "songs", Tier: "diamond", Query: "coAST", Page: 1})
	if len(filtered.Songs) != 1 || filtered.Songs[0].TrackID != 6 || filtered.CertifiedSongs != 7 {
		t.Fatalf("bad filter/counters: %+v", filtered)
	}
	filtered = get(db.HallOfFameOpts{View: "artists", Tier: "gold", Page: 1})
	if len(filtered.Artists) != 1 || filtered.Artists[0].ID != 1 {
		t.Fatal("artist tier filter")
	}
	if _, err := s.db.Exec("UPDATE artist_aliases SET alias='Renamed' WHERE artist_id=2"); err != nil {
		t.Fatal(err)
	}
	if len(get(db.HallOfFameOpts{View: "artists", Query: "renamed", Page: 1}).Artists) != 1 {
		t.Fatal("metadata stale")
	}
	if _, err := s.db.Exec("UPDATE track_aliases SET alias='Updated title' WHERE track_id=6"); err != nil {
		t.Fatal(err)
	}
	renamed := get(db.HallOfFameOpts{View: "songs", Query: "updated title", Page: 1})
	if len(renamed.Songs) != 1 || renamed.Songs[0].Plays != 1000 {
		t.Fatal("track metadata stale")
	}
	if _, err := s.db.Exec("DELETE FROM listens WHERE track_id=6 AND listened_at>1800"); err != nil {
		t.Fatal(err)
	}
	corrected := get(db.HallOfFameOpts{View: "artists", Tier: "diamond", Query: "renamed", Page: 1})
	if corrected.TotalRecordCount != 0 || corrected.Counts["diamond"] != 1 {
		t.Fatal("deletion stale")
	}
	if _, err := s.db.Exec("INSERT INTO listens VALUES(1,999,1,'')"); err != nil {
		t.Fatal(err)
	}
	if get(db.HallOfFameOpts{View: "songs", Page: 1}).CertifiedSongs != 8 {
		t.Fatal("first certification not updated")
	}
}

func TestHallOfFamePaginationAndSort(t *testing.T) {
	s := hallFixture(t)
	ctx := context.Background()
	for id := 1; id <= 23; id++ {
		hallSong(t, s, id, 200)
	}
	for _, tt := range []struct {
		sort, order        string
		page, first, count int
	}{{"count", "desc", 1, 1, 20}, {"name", "desc", 1, 23, 20}, {"date", "asc", 2, 21, 3}, {"count", "asc", 999, 21, 3}} {
		r, err := s.GetHallOfFame(ctx, db.HallOfFameOpts{View: "songs", Sort: tt.sort, Order: tt.order, Page: tt.page})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Songs) != tt.count || int(r.Songs[0].TrackID) != tt.first || r.TotalRecordCount != 23 || r.HasNextPage != (r.CurrentPage == 1) {
			t.Fatalf("bad pagination/sort %+v: %+v", tt, r)
		}
	}
}

func TestHallOfFameCrossingWithTrailingListens(t *testing.T) {
	ctx := context.Background()
	s := hallFixture(t)
	hallSong(t, s, 1, 1399)
	// A duplicate primary alias must not duplicate the song's artist link or award.
	if _, err := s.db.Exec("INSERT INTO artist_aliases VALUES(1,'Amber legacy alias','test',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("INSERT INTO listens VALUES(1,?,1,'')", time.Now().AddDate(1, 0, 0).Unix()); err != nil {
		t.Fatal(err)
	}

	result, err := s.GetHallOfFame(ctx, db.HallOfFameOpts{View: "songs"})
	if err != nil {
		t.Fatal(err)
	}
	if result.CertifiedSongs != 1 || result.CertifiedArtists != 1 || len(result.Songs) != 1 {
		t.Fatalf("unexpected totals: %+v", result)
	}
	song := result.Songs[0]
	if song.Plays != 1399 || song.Current.Tier != "platinum" || song.Current.Multiple != 3 ||
		song.Current.ReachedAt == nil || song.Current.ReachedAt.Unix() != 2200 || len(song.Artists) != 1 {
		t.Fatalf("unexpected song: %+v", song)
	}
}
