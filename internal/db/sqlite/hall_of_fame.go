package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/db"
)

func (s *Sqlite) GetHallOfFame(ctx context.Context, opts db.HallOfFameOpts) (*db.HallOfFame, error) {
	// Keep play counts, credits and crossing dates on the same read snapshot.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	isTime := opts.Metric == db.CertificationTime
	eligibility := `SELECT track_id, COUNT(*) AS plays FROM listens WHERE listened_at <= ?
		GROUP BY track_id HAVING COUNT(*) >= ?`
	minimum := db.CertificationPlayStep
	if isTime {
		eligibility = `SELECT l.track_id, COUNT(*) AS plays FROM listens l
			JOIN tracks t ON t.id = l.track_id
			WHERE l.listened_at <= ? AND t.duration > 0
			GROUP BY l.track_id HAVING COUNT(*) * t.duration >= ?`
		minimum = db.ListeningTimeGoldSeconds
	}
	rows, err := tx.QueryContext(ctx, `WITH eligible AS (`+eligibility+`)`+`
		SELECT t.id, t.title, e.plays, t.duration, r.image, a.id, a.name, a.image
		FROM eligible e JOIN tracks_with_title t ON t.id = e.track_id
		JOIN releases r ON r.id = t.release_id
		LEFT JOIN artist_tracks credit ON credit.track_id = t.id
		LEFT JOIN artists_with_name a ON a.id = credit.artist_id
		ORDER BY t.id, a.id, a.name, t.title`, now, minimum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	songs := []db.HallSong{}
	var durations []int64
	for rows.Next() {
		var id int32
		var title string
		var plays, duration int64
		var image, artistImage, artistName sql.NullString
		var artistID sql.NullInt64
		if err := rows.Scan(&id, &title, &plays, &duration, &image, &artistID, &artistName, &artistImage); err != nil {
			return nil, err
		}
		if len(songs) == 0 || songs[len(songs)-1].TrackID != id {
			song := db.HallSong{
				TrackID: id, Title: title, Plays: plays,
				Artists: []db.HallCredit{}, Image: catalog.BuildImageList(parseNullableUUID(image)),
			}
			if isTime {
				song.Seconds = plays * duration
				song.Current = db.ListeningTimeAward(song.Seconds)
				durations = append(durations, duration)
			} else {
				song.Current = db.CertificationTarget(plays)
			}
			songs = append(songs, song)
		}
		if artistID.Valid {
			song := &songs[len(songs)-1]
			// Older metadata can contain multiple primary aliases for one credit.
			// Rows are ordered by artist ID and name, so retain one stable entry.
			id := int32(artistID.Int64)
			if n := len(song.Artists); n == 0 || song.Artists[n-1].ID != id {
				song.Artists = append(song.Artists, db.HallCredit{
					ID: id, Name: artistName.String, Image: catalog.BuildImageList(parseNullableUUID(artistImage)),
				})
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if len(songs) > 0 {
		// Use the (track_id, listened_at) index for both crossing lookups.
		// A play award is at most 199 entries from the end; a time award
		// uses its threshold position from the beginning.
		order := "DESC"
		if isTime {
			order = "ASC"
		}
		crossing, err := tx.PrepareContext(ctx, `
			SELECT listened_at FROM listens WHERE track_id = ? AND listened_at <= ?
			ORDER BY listened_at `+order+` LIMIT 1 OFFSET ?`)
		if err != nil {
			return nil, err
		}
		defer crossing.Close()
		for i := range songs {
			song := &songs[i]
			var unix int64
			offset := song.Plays - song.Current.Plays
			if isTime {
				// At constant duration the threshold crossing is ceil(seconds/duration).
				// Its zero-based position is (seconds - 1) / duration.
				offset = (song.Current.Seconds - 1) / durations[i]
			}
			if err := crossing.QueryRowContext(ctx, song.TrackID, now, offset).Scan(&unix); err != nil {
				return nil, err
			}
			at := time.Unix(unix, 0).UTC()
			song.Current.ReachedAt = &at
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return db.BuildHallOfFame(songs, opts), nil
}
