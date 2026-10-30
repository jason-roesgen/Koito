package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func TestCertificationHistory(t *testing.T) {
	ctx := context.Background()
	s, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(ctx)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO users(id,username,password) VALUES(1,'test','test')")
	exec("INSERT INTO artists(id) VALUES(1),(2)")
	exec("INSERT INTO releases(id) VALUES(1)")
	exec("INSERT INTO tracks(id,release_id,duration) VALUES(1,1,60),(2,1,0),(3,1,60)")
	exec("INSERT INTO track_aliases VALUES(1,'shared','test',1),(2,'short history','test',1),(3,'unplayed','test',1)")
	exec("INSERT INTO artist_tracks VALUES(1,1,1),(2,1,0),(1,2,1)")
	// Insert reverse chronological order to exercise imported history.
	for play := int64(1200); play > 0; play-- {
		exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(1,?,1)", 1000+play)
	}
	for play := int64(199); play > 0; play-- {
		exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(2,?,1)", 1000+play)
	}
	exec("INSERT INTO listens(track_id,listened_at,user_id) VALUES(1,?,1)", time.Now().AddDate(1, 0, 0).Unix())
	track, err := s.GetTrackCertifications(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if track.Plays != 1200 || track.Current.Multiple != 3 || track.Current.Tier != "platinum" || track.Current.ReachedAt.Unix() != 2200 || track.History[0].ReachedAt.Unix() != 1200 || track.History[4].Tier != "diamond" {
		t.Fatalf("incorrect track: %+v", track)
	}
	for _, id := range []int32{1, 2} {
		board, err := s.GetArtistCertifications(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(board.Tracks) != 1 || board.Counts["platinum"] != 1 || board.Counts["gold"] != 0 || board.Counts["diamond"] != 0 {
			t.Fatalf("incorrect credit for artist %d: %+v", id, board)
		}
	}
	empty, err := s.GetTrackCertifications(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Plays != 0 || empty.Current != nil || len(empty.History) != 0 || empty.Next.Plays != 200 {
		t.Fatal("incorrect empty history")
	}
	if _, err := s.GetTrackCertifications(ctx, 99); !errors.Is(err, db.ErrNotFound) {
		t.Fatal("missing track should return not found")
	}
	if _, err := s.GetArtistCertifications(ctx, 99); !errors.Is(err, db.ErrNotFound) {
		t.Fatal("missing artist should return not found")
	}
	history, err := s.GetMilestoneHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if history.Plays != 1399 || history.Minutes != 1200 || history.PlayMilestones[0].ReachedAt.Unix() != 1001 || history.MinuteMilestones[0].ReachedAt.Unix() != 1060 || history.MinuteMilestones[1].ReachedAt.Unix() != 1600 || history.MinuteMilestones[2].ReachedAt != nil {
		t.Fatalf("incorrect milestone dates: %+v", history)
	}
	// A correction is reflected immediately rather than leaving a stale award.
	exec("DELETE FROM listens WHERE track_id=1 AND listened_at BETWEEN 2001 AND 2200")
	corrected, err := s.GetTrackCertifications(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Plays != 1000 || corrected.Current.Tier != "diamond" || len(corrected.History) != 5 {
		t.Fatalf("stale certification: %+v", corrected)
	}
}
