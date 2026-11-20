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
	rows, err := tx.QueryContext(ctx, `
		WITH eligible AS (
			SELECT track_id, COUNT(*) AS plays
			FROM listens WHERE listened_at <= ?
			GROUP BY track_id HAVING COUNT(*) >= ?
		)
		SELECT t.id, t.title, e.plays, r.image, a.id, a.name, a.image
		FROM eligible e JOIN tracks_with_title t ON t.id = e.track_id
		JOIN releases r ON r.id = t.release_id
		LEFT JOIN artist_tracks credit ON credit.track_id = t.id
		LEFT JOIN artists_with_name a ON a.id = credit.artist_id
		ORDER BY t.id, a.id, a.name, t.title`, now, db.CertificationPlayStep)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	songs := []db.HallSong{}
	for rows.Next() {
		var id int32
		var title string
		var plays int64
		var image, artistImage, artistName sql.NullString
		var artistID sql.NullInt64
		if err := rows.Scan(&id, &title, &plays, &image, &artistID, &artistName, &artistImage); err != nil {
			return nil, err
		}
		if len(songs) == 0 || songs[len(songs)-1].TrackID != id {
			songs = append(songs, db.HallSong{
				TrackID: id, Title: title, Plays: plays, Current: db.CertificationTarget(plays),
				Artists: []db.HallCredit{}, Image: catalog.BuildImageList(parseNullableUUID(image)),
			})
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
		// The crossing listen is at most 199 entries back from the latest
		// counted listen. The (track_id, listened_at) index serves each lookup
		// without numbering all the earlier scrobbles.
		crossing, err := tx.PrepareContext(ctx, `
			SELECT listened_at FROM listens WHERE track_id = ? AND listened_at <= ?
			ORDER BY listened_at DESC LIMIT 1 OFFSET ?`)
		if err != nil {
			return nil, err
		}
		defer crossing.Close()
		for i := range songs {
			song := &songs[i]
			var unix int64
			if err := crossing.QueryRowContext(ctx, song.TrackID, now, song.Plays-song.Current.Plays).Scan(&unix); err != nil {
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
