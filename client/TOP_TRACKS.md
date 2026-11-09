# Top Tracks calendar navigation

Top Tracks uses local calendar days, ISO weeks (Monday–Sunday), calendar months
and calendar years. Other chart pages retain their existing rolling shortcuts.
The Day picker selects a historical date; previous/next controls move one period.
The next control is disabled in the current period. All Time has no date controls.

The URL stores the period and a local `YYYY-MM-DD` anchor, so refresh and browser
back/forward restore the selection. Existing `year`/`month`/`week` links remain
supported. Changing periods resets the result page and retains ranking, limit,
artist/album and other non-date filters. Result pagination appears below the list.
An empty out-of-range result page redirects to page 1 with its filters preserved.

One helper calculates both displayed dates and explicit API boundaries. Local
calendar arithmetic handles daylight-saving transitions; the API's inclusive
integer-second end is the next period's local midnight minus one second. The
read-only demo applies the same inclusive endpoint convention.

## Verification

From `client/`, with Node supporting `--experimental-strip-types`:

```sh
TZ=Europe/Berlin corepack yarn test:calendar
TZ=America/New_York corepack yarn test:calendar
TZ=UTC corepack yarn test:calendar
corepack yarn typecheck
corepack yarn build
```

Browser checks: select a historical day, cross an ISO week/year boundary, move
between periods, paginate a multi-page list and use back/forward. Confirm ranking
and artist/album filters remain in the URL, empty results are explicit, and the
controls fit a mobile viewport. Demo data is synthetic; these checks do not require
production data or a running production backend.
