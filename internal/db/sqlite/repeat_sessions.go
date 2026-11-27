package sqlite

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/models"
)

type repeatListen struct {
	trackID int32
	at      int64
	length  int64
}

type repeatRun struct {
	repeatListen // First recorded play, including the track's catalog duration.
	end, count   int64
}

func (run repeatRun) session() db.RepeatSession {
	item := db.RepeatSession{
		Track: models.SimpleTrack{ID: run.trackID}, StartedAt: time.Unix(run.at, 0).UTC(),
		EndedAt: time.Unix(run.end, 0).UTC(), PlayCount: run.count, ElapsedSeconds: run.end - run.at,
	}
	if run.length > 0 {
		seconds := run.count * run.length
		item.EstimatedSeconds = &seconds
	}
	return item
}

type repeatStream struct {
	pending   repeatListen
	collision bool
	run       repeatRun
}

func (s *repeatStream) accept(next repeatListen, finish func(repeatRun)) {
	if next.at == s.pending.at {
		s.collision = true
		return
	}
	s.flushPending(finish)
	s.pending, s.collision = next, false
}

func (s *repeatStream) flushPending(finish func(repeatRun)) {
	if s.collision {
		// The order within this second is unknown. Neither event contributes
		// to a repeat, and the boundary still interrupts the previous run.
		finish(s.run)
		s.run = repeatRun{}
		return
	}
	play := s.pending
	maxGap := int64(db.RepeatSessionFallbackGapSeconds)
	if play.length > 0 {
		maxGap = play.length + db.RepeatSessionGraceSeconds
	}
	if s.run.count == 0 || play.trackID != s.run.trackID || play.at-s.run.end > maxGap {
		finish(s.run)
		s.run = repeatRun{repeatListen: play}
	}
	s.run.end = play.at
	s.run.count++
}

func (s *Sqlite) GetRepeatSessions(ctx context.Context, opts db.RepeatSessionOpts) (*db.RepeatSessionPage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var matches map[int32]bool
	if query := strings.ToLower(strings.TrimSpace(opts.Query)); query != "" {
		matches, err = matchRepeatTracks(ctx, tx, query)
		if err != nil {
			return nil, err
		}
	}

	runs, overview, err := scanRepeatRuns(ctx, tx, opts, matches)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sortRepeatRuns(runs, opts)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := makeRepeatSessionPage(runs, opts.Page)
	result.Overview = overview
	items := result.Items
	if overview.LongestSession != nil {
		items = append(items, *overview.LongestSession)
	}
	if len(items) > 0 {
		if err := loadRepeatMetadata(ctx, tx, items); err != nil {
			return nil, err
		}
		result.Items = items[:len(result.Items)]
		if overview.LongestSession != nil {
			result.Overview.LongestSession = &items[len(result.Items)]
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanRepeatRuns(ctx context.Context, tx *sql.Tx, opts db.RepeatSessionOpts, matches map[int32]bool) ([]repeatRun, db.RepeatSessionOverview, error) {
	var overview db.RepeatSessionOverview
	rows, err := tx.QueryContext(ctx, `
		SELECT l.user_id, l.track_id, l.listened_at, t.duration
		FROM listens l JOIN tracks t ON t.id = l.track_id
		WHERE l.listened_at <= ?
		ORDER BY l.listened_at`, time.Now().Unix())
	if err != nil {
		return nil, overview, err
	}
	defer rows.Close()

	streams := map[int32]*repeatStream{}
	var runs []repeatRun
	var longest repeatRun
	finish := func(run repeatRun) {
		if run.count < db.RepeatSessionMinimumPlays {
			return
		}
		overview.SessionCount++
		if run.length > 0 {
			overview.EstimatedSeconds += run.count * run.length
			overview.SessionsWithEstimates++
		}
		if compareRepeatRuns(run, longest, db.RepeatSessionOpts{}) < 0 {
			longest = run
		}
		// Apply filters only to completed runs. Even listens that cannot match
		// the search or date range must be allowed to interrupt a sequence.
		if (opts.From == nil || run.at >= *opts.From) &&
			(opts.To == nil || run.at < *opts.To) &&
			(matches == nil || matches[run.trackID]) {
			runs = append(runs, run)
		}
	}
	var userID int32
	var play repeatListen
	for rows.Next() {
		if err := rows.Scan(&userID, &play.trackID, &play.at, &play.length); err != nil {
			return nil, overview, err
		}
		if stream := streams[userID]; stream != nil {
			stream.accept(play, finish)
		} else {
			streams[userID] = &repeatStream{pending: play}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, overview, err
	}
	for _, stream := range streams {
		stream.flushPending(finish)
		finish(stream.run)
	}
	if longest.count > 0 {
		session := longest.session()
		overview.LongestSession = &session
	}
	return runs, overview, nil
}

func matchRepeatTracks(ctx context.Context, tx *sql.Tx, query string) (map[int32]bool, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT track_id, alias FROM track_aliases
		UNION ALL
		SELECT at2.track_id, aa.alias
		FROM artist_tracks at2 JOIN artist_aliases aa ON aa.artist_id = at2.artist_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := map[int32]bool{}
	for rows.Next() {
		var trackID int32
		var alias string
		if err := rows.Scan(&trackID, &alias); err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(alias), query) {
			matches[trackID] = true
		}
	}
	return matches, rows.Err()
}

func sortRepeatRuns(runs []repeatRun, opts db.RepeatSessionOpts) {
	slices.SortFunc(runs, func(a, b repeatRun) int {
		return compareRepeatRuns(a, b, opts)
	})
}

func compareRepeatRuns(a, b repeatRun, opts db.RepeatSessionOpts) int {
	result := cmp.Compare(a.count, b.count)
	switch opts.Sort {
	case "date":
		result = cmp.Compare(a.at, b.at)
	case "time":
		if a.length > 0 && b.length <= 0 {
			return -1
		}
		if a.length <= 0 && b.length > 0 {
			return 1
		}
		result = cmp.Compare(a.count*max(a.length, 0), b.count*max(b.length, 0))
	}
	if result == 0 {
		if byStart := cmp.Compare(b.at, a.at); byStart != 0 {
			return byStart
		}
		// (track ID, first timestamp) is unique in the listens table.
		return cmp.Compare(a.trackID, b.trackID)
	}
	if opts.Order == "asc" {
		return result
	}
	return -result
}

func makeRepeatSessionPage(runs []repeatRun, page int) *db.RepeatSessionPage {
	last := max(1, (len(runs)+defaultItemsPerPage-1)/defaultItemsPerPage)
	page = min(max(page, 1), last)
	start := (page - 1) * defaultItemsPerPage
	result := &db.RepeatSessionPage{
		PaginatedResponse: db.PaginatedResponse[db.RepeatSession]{
			Items: []db.RepeatSession{}, TotalCount: int64(len(runs)), ItemsPerPage: defaultItemsPerPage,
			CurrentPage: int32(page), HasNextPage: page < last,
		},
		MinimumPlays: db.RepeatSessionMinimumPlays,
	}
	for _, run := range runs[start:min(start+defaultItemsPerPage, len(runs))] {
		result.Items = append(result.Items, run.session())
	}
	return result
}

func loadRepeatMetadata(ctx context.Context, tx *sql.Tx, page []db.RepeatSession) error {
	tracks := map[int32]*models.SimpleTrack{}
	args := []any{}
	for _, item := range page {
		if tracks[item.Track.ID] == nil {
			tracks[item.Track.ID] = &models.SimpleTrack{
				ID: item.Track.ID, Title: "Unknown track", Artists: []models.SimpleArtist{},
			}
			args = append(args, item.Track.ID)
		}
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf(`
		SELECT t.id, COALESCE((SELECT MIN(alias) FROM track_aliases WHERE track_id = t.id AND is_primary = 1), 'Unknown track'),
			r.image, at2.artist_id, MIN(awn.name)
		FROM tracks t JOIN releases r ON r.id = t.release_id
		LEFT JOIN artist_tracks at2 ON at2.track_id = t.id
		LEFT JOIN artists_with_name awn ON awn.id = at2.artist_id
		WHERE t.id IN (%s)
		GROUP BY t.id, at2.artist_id
		ORDER BY t.id, MAX(at2.is_primary) DESC, MIN(awn.name), at2.artist_id`, placeholders)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var trackID int32
		var title string
		var image, artistName sql.NullString
		var artistID sql.NullInt64
		if err := rows.Scan(&trackID, &title, &image, &artistID, &artistName); err != nil {
			return err
		}
		track := tracks[trackID]
		track.Title = title
		track.Image = catalog.BuildImageList(parseNullableUUID(image))
		if artistID.Valid && artistName.Valid {
			track.Artists = append(track.Artists, models.SimpleArtist{ID: int32(artistID.Int64), Name: artistName.String})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range page {
		page[i].Track = *tracks[page[i].Track.ID]
	}
	return nil
}
