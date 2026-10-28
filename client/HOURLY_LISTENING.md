# Listening by hour

The dashboard shows 24 hourly totals for its selected Day/Week/Month/Year/All
period, using the same rolling periods as the existing rankings. Settings →
Dashboard controls visibility and selects Plays or Listening time (minutes).
Existing saved section preferences remain intact; the new section defaults to visible.

Hours use the browser's IANA timezone, unless `KOITO_FORCE_TZ` overrides it on
the backend. The chart displays the timezone actually used. Each timestamp is
converted separately, including daylight-saving changes. Repeated clock hours
are combined. Empty hours contain zero. The expandable table provides all values.

A play counts toward the hour in which it started. Minutes are the sum of full
known track durations for those plays, divided by 60; they are not split across
clock hours. Missing durations contribute zero minutes while still counting as
plays. These are estimates rather than measured playback time.

`GET /apis/web/v1/listening-by-hour?period=week&tz=Europe/Berlin` returns a timezone
and 24 `{ hour, plays, seconds }` buckets. The endpoint follows the existing login
gate. No schema migration is needed. The database streams matching listens and
keeps 24 counters in memory; runtime grows with the selected history.

Development demo mode computes the same hourly totals from its in-memory history.
It accepts the requested timezone; it does not read backend environment settings.

Backend regression tests cover spring/fall DST, quarter-hour offsets, unknown
durations, date-range filtering, and empty history:

```sh
go test ./internal/db/sqlite -run TestHourlyListening
```

Manual check: switch dashboard periods and Plays/Minutes, hover bars or expand
Hourly values, hide/show the section in Settings, and refresh to verify preferences.
