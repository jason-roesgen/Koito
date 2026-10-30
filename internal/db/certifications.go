package db

import (
	"context"
	"time"
)

type Certification struct {
	Tier      string     `json:"tier"`
	Multiple  int64      `json:"multiple"`
	Plays     int64      `json:"plays"`
	ReachedAt *time.Time `json:"reached_at,omitempty"`
}

// CertificationTarget labels the latest 200-play step. Diamond wins at shared
// thresholds, followed by Platinum. Thus 1,000 is Diamond, 1,200 is 3× Platinum.
func CertificationTarget(plays int64) Certification {
	threshold := plays / 200 * 200
	award := Certification{Plays: threshold, Tier: "gold", Multiple: threshold / 200}
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
	Next    Certification   `json:"next"`
	History []Certification `json:"history"`
}

func NewTrackCertifications(id int32, title string) *TrackCertifications {
	return &TrackCertifications{TrackID: id, Title: title, Next: CertificationTarget(200), History: []Certification{}}
}

// AddListen expects recorded listens in ascending timestamp order.
func (track *TrackCertifications) AddListen(at time.Time) {
	track.Plays++
	if track.Plays%200 == 0 {
		award := CertificationTarget(track.Plays)
		utc := at.UTC()
		award.ReachedAt = &utc
		track.History = append(track.History, award)
		track.Current = &track.History[len(track.History)-1]
	}
	track.Next = CertificationTarget((track.Plays/200 + 1) * 200)
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
	GetTrackCertifications(context.Context, int32) (*TrackCertifications, error)
	GetArtistCertifications(context.Context, int32) (*ArtistCertifications, error)
	GetMilestoneHistory(context.Context) (*MilestoneHistory, error)
}
