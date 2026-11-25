package db

import (
	"context"
	"time"
)

const CertificationPlayStep = 200

const (
	CertificationPlays       = "plays"
	CertificationTime        = "time"
	ListeningTimeGoldSeconds = 500 * 60
	listeningTimeDiamondStep = 2500 * 60
)

var listeningTimeTargets = [...]Certification{
	{Tier: "gold", Multiple: 1, Seconds: ListeningTimeGoldSeconds},
	{Tier: "platinum", Multiple: 1, Seconds: 1000 * 60},
}

func listeningTimeDiamond(multiple int64) Certification {
	return Certification{Tier: "diamond", Multiple: multiple, Seconds: multiple * listeningTimeDiamondStep}
}

// ListeningTimeAward returns the highest time threshold reached, or a zero award.
func ListeningTimeAward(seconds int64) Certification {
	if seconds >= listeningTimeDiamondStep {
		return listeningTimeDiamond(seconds / listeningTimeDiamondStep)
	}
	for i := len(listeningTimeTargets) - 1; i >= 0; i-- {
		if seconds >= listeningTimeTargets[i].Seconds {
			return listeningTimeTargets[i]
		}
	}
	return Certification{}
}

type Certification struct {
	Tier      string     `json:"tier"`
	Multiple  int64      `json:"multiple"`
	Plays     int64      `json:"plays"`
	Seconds   int64      `json:"seconds,omitempty"`
	ReachedAt *time.Time `json:"reached_at,omitempty"`
}

// CertificationTarget labels the latest 200-play step. Diamond wins at shared
// thresholds, followed by Platinum. Thus 1,000 is Diamond, 1,200 is 3× Platinum.
func CertificationTarget(plays int64) Certification {
	threshold := plays / CertificationPlayStep * CertificationPlayStep
	award := Certification{Plays: threshold, Tier: "gold", Multiple: threshold / CertificationPlayStep}
	if threshold > 0 && threshold%1000 == 0 {
		award.Tier, award.Multiple = "diamond", threshold/1000
	} else if threshold > 0 && threshold%400 == 0 {
		award.Tier, award.Multiple = "platinum", threshold/400
	}
	return award
}

type TrackCertifications struct {
	TrackID int32           `json:"track_id"`
	Title   string          `json:"title"`
	Plays   int64           `json:"plays"`
	Current *Certification  `json:"current"`
	Next    *Certification  `json:"next"`
	Seconds int64           `json:"seconds,omitempty"`
	History []Certification `json:"history"`
}

func NewTrackCertifications(id int32, title string) *TrackCertifications {
	next := CertificationTarget(CertificationPlayStep)
	return &TrackCertifications{TrackID: id, Title: title, Next: &next, History: []Certification{}}
}

// AddListen expects recorded listens in ascending timestamp order.
func (track *TrackCertifications) AddListen(at time.Time) {
	track.Plays++
	if track.Plays%CertificationPlayStep == 0 {
		award := CertificationTarget(track.Plays)
		utc := at.UTC()
		award.ReachedAt = &utc
		track.History = append(track.History, award)
		track.Current = &track.History[len(track.History)-1]
		next := CertificationTarget(track.Plays + CertificationPlayStep)
		track.Next = &next
	}
}

func NewTrackTimeCertifications(id int32, title string) *TrackCertifications {
	next := listeningTimeTargets[0]
	return &TrackCertifications{TrackID: id, Title: title, Next: &next, History: []Certification{}}
}

// AddTimedListen uses the current stored track length for each dated scrobble.
func (track *TrackCertifications) AddTimedListen(at time.Time, duration int64) {
	track.Plays++
	track.Seconds += max(0, duration)
	for track.Next != nil && track.Seconds >= track.Next.Seconds {
		award := *track.Next
		utc := at.UTC()
		award.ReachedAt = &utc
		track.History = append(track.History, award)
		track.Current = &track.History[len(track.History)-1]
		var next Certification
		switch award.Tier {
		case "gold":
			next = listeningTimeTargets[1]
		case "platinum":
			next = listeningTimeDiamond(1)
		case "diamond":
			next = listeningTimeDiamond(award.Multiple + 1)
		}
		track.Next = &next
	}
}

type ArtistCertifications struct {
	Counts map[string]int         `json:"counts"`
	Tracks []*TrackCertifications `json:"tracks"`
}

type MilestoneDate struct {
	Target    int64      `json:"target"`
	ReachedAt *time.Time `json:"reached_at"`
}

type MilestoneHistory struct {
	Plays            int64           `json:"plays"`
	Minutes          int64           `json:"minutes"`
	PlayMilestones   []MilestoneDate `json:"play_milestones"`
	MinuteMilestones []MilestoneDate `json:"minute_milestones"`
}

type CertificationStore interface {
	GetTrackCertifications(context.Context, int32, string) (*TrackCertifications, error)
	GetArtistCertifications(context.Context, int32, string) (*ArtistCertifications, error)
	GetMilestoneHistory(context.Context) (*MilestoneHistory, error)
}
