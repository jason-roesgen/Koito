package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func (s *Sqlite) GetTrackCertifications(ctx context.Context, id int32) (*db.TrackCertifications, error) {
	var title string
	if err := s.db.QueryRowContext(ctx, "SELECT title FROM tracks_with_title WHERE id = ?", id).Scan(&title); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}
	track := db.NewTrackCertifications(id, title)
	rows, err := s.db.QueryContext(ctx, `SELECT listened_at FROM listens WHERE track_id = ? AND listened_at <= ? ORDER BY listened_at`, id, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var unix int64
		if err := rows.Scan(&unix); err != nil {
			return nil, err
		}
		track.AddListen(time.Unix(unix, 0))
	}
	return track, rows.Err()
}

func (s *Sqlite) GetArtistCertifications(ctx context.Context, id int32) (*db.ArtistCertifications, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, "SELECT 1 FROM artists WHERE id = ?", id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}
	board := &db.ArtistCertifications{Counts: map[string]int{"gold": 0, "platinum": 0, "diamond": 0}, Tracks: []*db.TrackCertifications{}}
	// EXISTS gives each track one entry even when it has multiple artist credits.
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.title, l.listened_at
		FROM tracks_with_title t JOIN listens l ON l.track_id = t.id
		WHERE EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id = t.id AND a.artist_id = ?)
		AND l.listened_at <= ?
		ORDER BY t.id, l.listened_at`, id, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var track *db.TrackCertifications
	finish := func() {
		if track != nil && track.Current != nil {
			board.Tracks = append(board.Tracks, track)
			board.Counts[track.Current.Tier]++
		}
	}
	for rows.Next() {
		var trackID int32
		var title string
		var unix int64
		if err := rows.Scan(&trackID, &title, &unix); err != nil {
			return nil, err
		}
		if track == nil || track.TrackID != trackID {
			finish()
			track = db.NewTrackCertifications(trackID, title)
		}
		track.AddListen(time.Unix(unix, 0))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	finish()
	return board, nil
}

func (s *Sqlite) GetMilestoneHistory(ctx context.Context) (*db.MilestoneHistory, error) {
	dates := func(targets []int64) []db.MilestoneDate {
		result := make([]db.MilestoneDate, len(targets))
		for i, target := range targets {
			result[i].Target = target
		}
		return result
	}
	history := &db.MilestoneHistory{
		PlayMilestones:   dates([]int64{1, 100, 500, 1000, 5000, 10000, 25000, 50000, 100000, 250000, 500000, 1000000}),
		MinuteMilestones: dates([]int64{60, 600, 3000, 6000, 30000, 60000, 150000, 300000, 600000}),
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.listened_at, t.duration FROM listens l JOIN tracks t ON t.id = l.track_id
		WHERE l.listened_at <= ? ORDER BY l.listened_at, l.track_id`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seconds int64
	playIndex, minuteIndex := 0, 0
	for rows.Next() {
		var unix, duration int64
		if err := rows.Scan(&unix, &duration); err != nil {
			return nil, err
		}
		at := time.Unix(unix, 0).UTC()
		history.Plays++
		if duration > 0 {
			seconds += duration
		}
		for playIndex < len(history.PlayMilestones) && history.Plays >= history.PlayMilestones[playIndex].Target {
			history.PlayMilestones[playIndex].ReachedAt = &at
			playIndex++
		}
		for minuteIndex < len(history.MinuteMilestones) && seconds >= history.MinuteMilestones[minuteIndex].Target*60 {
			history.MinuteMilestones[minuteIndex].ReachedAt = &at
			minuteIndex++
		}
	}
	history.Minutes = seconds / 60
	return history, rows.Err()
}
