package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func (s *Sqlite) GetTrackCertifications(ctx context.Context, id int32, metric string) (*db.TrackCertifications, error) {
	var title string
	var duration int64
	// A single read transaction keeps the duration and the list of listens in sync.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, "SELECT title, duration FROM tracks_with_title WHERE id = ? ORDER BY title LIMIT 1", id).Scan(&title, &duration); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}
	track := newTrackCertification(id, title, metric)
	rows, err := tx.QueryContext(ctx, `SELECT listened_at FROM listens WHERE track_id = ? AND listened_at <= ? ORDER BY listened_at`, id, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var unix int64
		if err := rows.Scan(&unix); err != nil {
			return nil, err
		}
		addCertificationListen(track, time.Unix(unix, 0), duration, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return track, tx.Commit()
}

func (s *Sqlite) GetArtistCertifications(ctx context.Context, id int32, metric string) (*db.ArtistCertifications, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, "SELECT 1 FROM artists WHERE id = ?", id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}
	board := &db.ArtistCertifications{Counts: map[string]int{"gold": 0, "platinum": 0, "diamond": 0}, Tracks: []*db.TrackCertifications{}}
	// Pick one primary title per credited track before joining listens.
	rows, err := s.db.QueryContext(ctx, `
		WITH credited AS MATERIALIZED (
			SELECT t.id, t.duration, (
				SELECT MIN(alias) FROM track_aliases WHERE track_id = t.id AND is_primary = 1
			) AS title
			FROM artist_tracks a JOIN tracks t ON t.id = a.track_id
			WHERE a.artist_id = ?
		)
		SELECT t.id, t.title, l.listened_at, t.duration
		FROM credited t JOIN listens l ON l.track_id = t.id
		WHERE t.title IS NOT NULL AND l.listened_at <= ?
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
		var unix, duration int64
		if err := rows.Scan(&trackID, &title, &unix, &duration); err != nil {
			return nil, err
		}
		if track == nil || track.TrackID != trackID {
			finish()
			track = newTrackCertification(trackID, title, metric)
		}
		addCertificationListen(track, time.Unix(unix, 0), duration, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	finish()
	return board, nil
}

func newTrackCertification(id int32, title, metric string) *db.TrackCertifications {
	if metric == db.CertificationTime {
		return db.NewTrackTimeCertifications(id, title)
	}
	return db.NewTrackCertifications(id, title)
}

func addCertificationListen(track *db.TrackCertifications, at time.Time, duration int64, metric string) {
	if metric == db.CertificationTime {
		track.AddTimedListen(at, duration)
		return
	}
	track.AddListen(at)
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
