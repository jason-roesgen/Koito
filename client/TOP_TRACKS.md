# Top charts calendar navigation

Top Tracks, Top Albums and Top Artists share local calendar days, ISO weeks
(Monday–Sunday), calendar months and calendar years. Last Played retains its
existing rolling shortcuts.
The Day picker selects a historical date; previous/next controls move one period.
The next control is disabled in the current period. All Time has no date controls.

The URL stores the period and a local `YYYY-MM-DD` anchor, so refresh and browser
back/forward restore the selection. Switching tabs preserves that historical
anchor. Weeks use the ISO week-year, including weeks spanning December/January.
Existing `year`/`month`/`week` links remain supported, including month/week links
without an explicit year. Changing periods resets the result page and retains ranking, limit,
artist/album and other non-date filters. Result pagination appears below the list.
An empty out-of-range result page redirects to page 1 with its filters preserved.

Dashboard cards keep their rolling date windows. Their query and drilldown link
carry the same frozen `from`/`to` timestamps, preserving displayed counts. Ranking
pages label these as a selected range; choosing a period tab explicitly switches
to calendar browsing. Explicit range links also survive pagination and history.

One helper calculates both displayed dates and explicit API boundaries. Local
calendar arithmetic handles daylight-saving transitions; the API's inclusive
integer-second end is the next period's local midnight minus one second.
Both midnights are constructed independently so a midnight DST skip cannot carry
a normalized 01:00 into the next boundary. The
read-only demo applies the same inclusive endpoint convention.

## Verification

From `client/`, with Node supporting `--experimental-strip-types`:

```sh
TZ=Europe/Berlin corepack yarn test:calendar
TZ=America/New_York corepack yarn test:calendar
TZ=UTC corepack yarn test:calendar
TZ=America/Santiago corepack yarn test:calendar
corepack yarn typecheck
corepack yarn build
```

On each ranking page, select a historical day, cross an ISO week/year boundary,
move between periods, paginate a multi-page list and use back/forward. Confirm
ranking and artist/album filters remain in the URL, empty results are explicit,
and the controls fit a mobile viewport. Inactive period labels share the muted
color. Demo data is synthetic; these checks do not require production data or a
running production backend.

The Calendar regression tests CI job runs the same suite in all four timezones
without installing frontend dependencies.
