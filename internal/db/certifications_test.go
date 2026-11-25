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

func TestListeningTimeCertificationThresholds(t *testing.T) {
	track := NewTrackTimeCertifications(1, "song")
	goldAt := time.Unix(2, 0)
	platinumAt := time.Unix(4, 0)
	diamondAt := time.Unix(5, 0)
	doubleAt := time.Unix(7, 0)
	tripleAt := time.Unix(8, 0)
	track.AddTimedListen(time.Unix(1, 0), 29999)
	if track.Current != nil || track.Next.Seconds != 30000 {
		t.Fatal("a play just below Gold earned an award")
	}
	track.AddTimedListen(goldAt, 1)
	track.AddTimedListen(time.Unix(3, 0), -1)
	track.AddTimedListen(platinumAt, 30000)
	track.AddTimedListen(diamondAt, 90000)
	track.AddTimedListen(time.Unix(6, 0), 149999)
	if track.Current.Multiple != 1 || track.Next.Seconds != 300000 {
		t.Fatal("2× Diamond must require all 5,000 minutes")
	}
	track.AddTimedListen(doubleAt, 1)
	track.AddTimedListen(tripleAt, 150000)

	if track.Seconds != 450000 || track.Plays != 8 || track.Next == nil ||
		track.Next.Seconds != 600000 || track.Next.Multiple != 4 || len(track.History) != 5 {
		t.Fatalf("unexpected time progress: %+v", track)
	}
	for i, want := range []struct {
		tier     string
		multiple int64
		at       time.Time
	}{{"gold", 1, goldAt}, {"platinum", 1, platinumAt}, {"diamond", 1, diamondAt}, {"diamond", 2, doubleAt}, {"diamond", 3, tripleAt}} {
		got := track.History[i]
		if got.Tier != want.tier || got.Multiple != want.multiple || !got.ReachedAt.Equal(want.at) {
			t.Fatalf("threshold %d: got %+v", i, got)
		}
	}
	if track.Current.Tier != "diamond" || track.Current.Multiple != 3 || !track.Current.ReachedAt.Equal(tripleAt) {
		t.Fatal("3× Diamond should become the current award")
	}
}
