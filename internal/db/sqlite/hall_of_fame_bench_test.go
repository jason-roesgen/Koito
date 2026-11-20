package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gabehf/koito/internal/db"
)

// Each 20k-listen block has 30 certified songs (including collaborations) and
// 600 less-played songs. The blocks simulate more history and a growing library.
func BenchmarkHallOfFame(b *testing.B) {
	for _, blocks := range []int{2, 8, 32} {
		b.Run(fmt.Sprintf("%dk-listens", blocks*20), func(b *testing.B) {
			dsn := seedHallBenchmark(b, blocks)
			for _, tt := range []struct {
				name            string
				opts            db.HallOfFameOpts
				freshConnection bool
			}{
				{"initial-artists", db.HallOfFameOpts{View: "artists", Page: 1}, true},
				{"sort-song-date", db.HallOfFameOpts{View: "songs", Sort: "date", Page: 1}, false},
				{"search-artist-gold", db.HallOfFameOpts{View: "artists", Query: "coast", Tier: "gold", Page: 1}, false},
				{"songs-page-3", db.HallOfFameOpts{View: "songs", Page: 3}, false},
			} {
				b.Run(tt.name, func(b *testing.B) {
					open := func() *Sqlite {
						conn, err := sql.Open("sqlite", dsn)
						if err != nil {
							b.Fatal(err)
						}
						return &Sqlite{db: conn}
					}
					ctx := context.Background()
					s := open()
					defer s.Close(ctx)
					first, err := s.GetHallOfFame(ctx, tt.opts)
					if err != nil {
						b.Fatal(err)
					}
					if first.CertifiedSongs != 30*blocks || first.TotalRecordCount == 0 || first.CurrentPage != tt.opts.Page {
						b.Fatalf("fixture does not exercise %s: %d certified songs, %d matches, page %d", tt.name, first.CertifiedSongs, first.TotalRecordCount, first.CurrentPage)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						conn := s
						if tt.freshConnection {
							conn = open()
						}
						_, err := conn.GetHallOfFame(ctx, tt.opts)
						if tt.freshConnection {
							conn.Close(ctx)
						}
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}

func seedHallBenchmark(b *testing.B, blocks int) string {
	b.Helper()
	s := hallFixture(b)
	// Each block has 10 tracks × 237, 8 × 486, 8 × 713 and 4 × 1,137.
	// 490 tracks at 6 plays and 110 at 5 fill the long tail to 20,000.
	for _, query := range []string{
		`WITH RECURSIVE ids(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM ids WHERE i<` + fmt.Sprint(630*blocks) + `)
		 INSERT INTO artists(id) SELECT i+3 FROM ids`,
		`INSERT INTO artist_aliases SELECT id,printf('%s %06d',CASE WHEN id%4=0 THEN 'Coast' ELSE 'Artist' END,id),'test',1 FROM artists WHERE id>3`,
		`INSERT INTO tracks(id,release_id,duration) SELECT id-3,1,120+(id%200) FROM artists WHERE id>3`,
		`INSERT INTO track_aliases SELECT id, printf('Song %06d',id),'test',1 FROM tracks`,
		`INSERT INTO artist_tracks SELECT id+3,id,1 FROM tracks`,
		`INSERT INTO artist_tracks SELECT 1,id,0 FROM tracks WHERE id%5=0`,
		`WITH RECURSIVE seq(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM seq WHERE n<1137)
		 INSERT INTO listens(track_id,listened_at,user_id)
		 SELECT id,1700000000+(n*12+id)*60,1 FROM tracks JOIN seq ON n<=CASE
		    WHEN (id-1)%630<10 THEN 237 WHEN (id-1)%630<18 THEN 486
		    WHEN (id-1)%630<26 THEN 713 WHEN (id-1)%630<30 THEN 1137
		    WHEN (id-1)%630<520 THEN 6 ELSE 5 END`,
	} {
		if _, err := s.db.Exec(query); err != nil {
			b.Fatal(err)
		}
	}
	count, err := s.Count("SELECT COUNT(*) FROM listens")
	if err != nil || count != blocks*20000 {
		b.Fatalf("fixture listen count: %d (%v)", count, err)
	}
	filename := filepath.Join(b.TempDir(), "hall.db")
	if _, err := s.db.Exec("VACUUM INTO ?", filename); err != nil {
		b.Fatal(err)
	}
	s.Close(context.Background())
	return filename + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
}
