package db

import "context"

// HourlyListeningBucket groups listens by their local start hour, 0–23.
// Seconds are estimated from known track durations; they are not measured playback.
type HourlyListeningBucket struct {
	Hour    int   `json:"hour"`
	Plays   int64 `json:"plays"`
	Seconds int64 `json:"seconds"`
}

type HourlyListeningStore interface {
	GetHourlyListening(context.Context, Timeframe) ([]HourlyListeningBucket, error)
}
