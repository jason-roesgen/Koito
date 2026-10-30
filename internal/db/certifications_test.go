package db

import (
	"testing"
	"time"
)

func TestCertificationLadder(t *testing.T) {
	tests := []struct {
		plays    int64
		tier     string
		multiple int64
	}{
		{200, "gold", 1}, {400, "platinum", 1}, {600, "gold", 3}, {800, "platinum", 2},
		{1000, "diamond", 1}, {1200, "platinum", 3}, {1400, "gold", 7}, {1600, "platinum", 4},
		{1800, "gold", 9}, {2000, "diamond", 2}, {2400, "platinum", 6},
	}
	for _, tt := range tests {
		for _, count := range []int64{tt.plays, tt.plays + 199} {
			award := CertificationTarget(count)
			if award.Tier != tt.tier || award.Multiple != tt.multiple || award.Plays != tt.plays {
				t.Fatalf("%d: %+v", count, award)
			}
		}
	}
	track := NewTrackCertifications(1, "test")
	for play := int64(1); play <= 1200; play++ {
		track.AddListen(time.Unix(play, 0))
		if play == 199 && track.Current != nil {
			t.Fatal("Gold awarded early")
		}
		if play == 200 && track.Current.ReachedAt.Unix() != 200 {
			t.Fatal("incorrect crossing date")
		}
	}
	if len(track.History) != 6 || track.History[4].Tier != "diamond" || track.Current.Tier != "platinum" || track.Current.Multiple != 3 || track.Next.Plays != 1400 {
		t.Fatalf("incorrect history: %+v", track)
	}
}
