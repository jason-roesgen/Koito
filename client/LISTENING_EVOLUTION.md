# Listening Evolution

Repeat Sessions lists runs of at least three consecutive recorded plays of the same track by one user. The gap between adjacent plays can be up to the track’s stored duration plus five minutes. When the duration is unknown, the maximum gap is ten minutes. A different track ends the run. When one user has different tracks recorded in the same second, that second ends the run and those plays are excluded from both adjacent runs.

Detection uses the recorded timestamps, including imported history. The page can search track and artist names, filter on the local calendar date of the first play, and sort by first play, play count or estimated listening time. A run can cross midnight. Filtering leaves its boundaries and play count intact.

- **Recorded plays** is the number of listening events in the run.
- **Estimated listening time** is the stored track duration multiplied by the recorded play count. It is unavailable when the duration is unknown.
- **Elapsed span** is the time between the first and last recorded plays.

Repeat Sessions describes a pattern in the recorded history. Edits, imports, deletions and duration corrections can change the results. Viewing the page makes no changes to listening history. Detection is per user and the results share the instance’s existing web statistics visibility settings.

The read-only `GET /apis/web/v1/repeat-sessions` endpoint accepts `q`, `sort` (`plays`, `date`, `time`), `order` (`asc`, `desc`), `page`, and Unix-second `from` and `to` bounds. The start bound is inclusive and the end bound is exclusive. One chronological scan per request computes the all-time overview and finds matching runs under the same SQLite snapshot. Matching runs are kept in memory for sorting and pagination. Track metadata is loaded for the current page and the featured session together.

Search, sort and pagination each trigger a fresh scan. Large histories and simultaneous requests can take longer and use more memory, even when a filter matches no sessions. A long read can also delay SQLite WAL checkpoint progress. New or edited plays can change the ordering between requests. To benchmark the read path and concurrent writes with disposable databases, run `go test ./internal/db/sqlite -run '^$' -bench '^BenchmarkRepeatSessions' -benchmem`.

The overview and featured longest session cover all available history, including when the session history is filtered. The estimated total adds only sessions with known track duration. Ties for longest favor the more recent start, then the lower track ID. Expand a session in the history to see the first and last recorded timestamps and elapsed span.
