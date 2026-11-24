# Hall of Fame

Open Hall of Fame from the trophy icon in the sidebar, or `/hall-of-fame`.
The default Artists tab combines every artist with at least one currently
certified song. Songs lists the certified tracks themselves. The Plays / Listening
time switch selects which award category the lists, counts, filters and histories use.

- Global counters show certified songs, credited artists, and songs by current tier.
  Search and tier filters do not change those counters.
- Each song counts once under its current certificate. A collaboration counts
  once for each credited artist, so artist totals can exceed the song total.
- In Plays mode, existing 200/400/1,000-play rules and 200-play steps apply. A song at 1,200
  plays is currently 3× Platinum, with its earlier Diamond in its track history.
- In Listening time mode, Gold requires 500 estimated minutes and Platinum 1,000.
  Diamond repeats every 2,500 minutes (2× Diamond at 5,000, and so on). The award date is the
  recorded listen that first crosses that threshold.
- Gold/Platinum/Diamond filters match the current tier. Artist filters include
  artists with at least one song in that tier; their breakdown still shows all
  their certified songs. Count sorting uses the selected tier when filtering.
- Song search matches titles or artist names; artist search matches artist names.
- Sort by count (artist certified songs / song plays), name, or date, ascending
  or descending. Songs can sort by estimated time in Listening time mode. Artist dates show the latest current award across all tiers.
- Results use 20 entries per page. Equal sort values use ascending IDs for stable
  pagination. Filters, tabs, sorting and pages are stored in the URL for browser
  navigation and sharing. Changing filters resets to page one.
- Dates use browser-local time. Expand a song's history to see its first and
  current certifications and every award in between, newest first. A search
  returning one song opens its history automatically. History loads on demand
  from the existing track certification endpoint.

The read-only `GET /apis/web/v1/hall-of-fame` endpoint is protected by the existing
login gate. Query parameters: `metric=plays|time` (defaults to plays),
`view=artists|songs`, `q` (up to 200 Unicode code points),
`tier=gold|platinum|diamond` (omit for all), `sort=count|name|date`,
`order=asc|desc`, and `page` (1–1,000,000). Defaults: artists, count, descending,
page one. Out-of-range pages are clamped to the final available page.

The database counts plays before joining artist credits. Within the same read
transaction, an indexed lookup finds the award date. Play awards are at most 199
listens back from the latest eligible listen. Time awards look up the first listen
that reaches the threshold using the stored track duration. A single timestamp cutoff
excludes future-dated listens from both counts and award dates. The browser loads
one page per request and does not poll. No migration or response cache is needed.
Imports, deletions and metadata corrections are reflected on the next page load.

Synthetic development mode supports the page without accessing real listening data:

```sh
cd ~/Projects/Koito/client
VITE_KOITO_DEMO=true corepack yarn dev --host 127.0.0.1 --port 5187 --strictPort
```

Validation:

```sh
go test ./...
go test ./internal/db/sqlite ./engine/handlers -run HallOfFame -race
cd client
corepack yarn test:hall-of-fame
corepack yarn typecheck
corepack yarn build
```

Run the benchmark from the repository root.

```sh
GOMAXPROCS=2 go test ./internal/db/sqlite -run '^$' -bench '^BenchmarkHallOfFame$' -benchtime=9x
```

The fixture uses Koito's migrations and creates disposable SQLite files with WAL,
a long tail of less-played tracks, several award tiers and shared artist credits.
The numbers below are Go store call averages on an AMD Ryzen 7 5800X, in
milliseconds per request (original query → indexed lookup). The first load opens a
fresh SQLite connection. Other operations reuse a warmed connection.

| Listens | Initial Artists | Songs by date | Artist search + Gold | Songs page 3 |
| ---: | ---: | ---: | ---: | ---: |
| 40,000 | 55.5 → 7.3 | 54.0 → 6.2 | 54.0 → 6.3 | 54.0 → 5.8 |
| 160,000 | 233.6 → 27.9 | 229.9 → 24.7 | 230.4 → 24.2 | 229.6 → 23.9 |
| 640,000 | 957.0 → 105.4 | 932.8 → 99.9 | 923.2 → 99.8 | 928.4 → 99.2 |

The temp files were on tmpfs. The benchmark excludes HTTP, rendering, physical
disk I/O and concurrent ingestion. Work still grows with the amount of recorded
history, since each request recalculates all-time counters. Measurements on
Raspberry Pi hardware remain outstanding.
