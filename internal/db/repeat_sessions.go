package db

import (
	"context"
	"time"

	"github.com/gabehf/koito/internal/models"
)

const (
	RepeatSessionMinimumPlays       = 3
	RepeatSessionGraceSeconds       = 5 * 60
	RepeatSessionFallbackGapSeconds = 10 * 60
)

type RepeatSession struct {
	Track            models.SimpleTrack `json:"track"`
	StartedAt        time.Time          `json:"started_at"`
	EndedAt          time.Time          `json:"ended_at"`
	PlayCount        int64              `json:"play_count"`
	EstimatedSeconds *int64             `json:"estimated_seconds"`
	ElapsedSeconds   int64              `json:"elapsed_seconds"`
}

type RepeatSessionOpts struct {
	Query, Sort, Order string
	From, To           *int64 // Session start in [From, To), Unix seconds.
	Page               int
}

type RepeatSessionPage struct {
	PaginatedResponse[RepeatSession]
	MinimumPlays int                   `json:"minimum_plays"`
	Overview     RepeatSessionOverview `json:"overview"`
}

type RepeatSessionOverview struct {
	SessionCount          int64          `json:"session_count"`
	EstimatedSeconds      int64          `json:"estimated_seconds"`
	SessionsWithEstimates int64          `json:"sessions_with_estimates"`
	LongestSession        *RepeatSession `json:"longest_session"`
}

type RepeatSessionStore interface {
	GetRepeatSessions(context.Context, RepeatSessionOpts) (*RepeatSessionPage, error)
}
