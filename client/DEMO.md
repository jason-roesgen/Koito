# Local synthetic listening history

Set `VITE_KOITO_DEMO=true` in `client/.env.local`, then run `corepack yarn dev`
from `client/`. Refresh the browser after switching modes. Set the flag to
`false` and restart Vite to return to the local backend data.

The Vite development middleware generates 451 calendar days of fictional
listens, with varied play counts, gaps, timestamps, and track durations.
Statistics, rankings, activity, detail pages, search, and rewind all derive
from the same in-memory history. Artwork is generated locally as SVG data URLs.
The date window is anchored when Vite starts and moves forward on restart.

A banner labels the demo. API mutations are rejected, and demo API requests
never reach the Go backend. No database or production instance is accessed.
The middleware only runs during Vite development; production builds do not
serve these fixtures. This is a read-only UI fixture, not a backend analytics
correctness test. Date grouping follows the development server's timezone.
