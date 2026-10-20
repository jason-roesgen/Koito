package sqlite

import "github.com/gabehf/koito/internal/db"

// Only fixed SQL identifiers may be interpolated into ranking queries.
func rankingColumn(opts db.GetItemsOpts) string {
	if opts.RankByTime {
		return "time_listened"
	}
	return "listen_count"
}
