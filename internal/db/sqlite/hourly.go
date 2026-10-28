package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/gabehf/koito/internal/db"
)

func (s *Sqlite) GetHourlyListening(ctx context.Context, timeframe db.Timeframe) ([]db.HourlyListeningBucket, error) {
	from, to := db.TimeframeToTimeRange(timeframe)
	loc := timeframe.Timezone
	if loc == nil {
		loc = time.UTC
	}
	buckets := make([]db.HourlyListeningBucket, 24)
	for hour := range buckets {
		buckets[hour].Hour = hour
	}

	// Stream timestamps rather than applying a fixed UTC offset in SQL: offsets
	// can change within the range, and some zones have half/quarter-hour offsets.
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.listened_at, t.duration
		FROM listens l JOIN tracks t ON t.id = l.track_id
		WHERE l.listened_at BETWEEN ? AND ?`, from.Unix(), to.Unix())
	if err != nil {
		return nil, fmt.Errorf("GetHourlyListening: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var unix, duration int64
		if err := rows.Scan(&unix, &duration); err != nil {
			return nil, err
		}
		hour := time.Unix(unix, 0).In(loc).Hour()
		buckets[hour].Plays++
		if duration > 0 {
			buckets[hour].Seconds += duration
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buckets, nil
}
