# Personal certifications and listening milestones

Track pages show a song's current personal certificate, progress to the next
200-play step, and an expandable history with dates/times for each certificate.
Artist pages show counts and boards of certified songs. Each song counts once
under its current certificate; all artists credited on the track receive credit.
Earlier awards remain in that song's history and do not inflate artist totals.

These are personal listening awards inspired by the BVMI system, not official
industry certifications. The rules apply to all stored history, including imports:

| Plays | Certificate |
| ---: | --- |
| 200 | Gold |
| 400 | Platinum |
| 600 | 3× Gold |
| 800 | 2× Platinum |
| 1,000 | Diamond |
| 1,200 | 3× Platinum |
| 1,400 | 7× Gold |
| 1,600 | 4× Platinum |
| 1,800 | 9× Gold |
| 2,000 | 2× Diamond |

The latest completed 200-play step determines the current certificate. At a shared
threshold, Diamond takes priority over Platinum, which takes priority over Gold.
Multipliers use 200/400/1,000 respectively, without an upper limit. The system counts
recorded listens; it does not enforce premium-stream or playback-completion rules.
The dashboard's Plays/Minutes display preference does not change certification
eligibility. A separate minutes-based certification scheme is not implemented yet.

General dashboard milestones remain available, with all-time play/hour targets,
progress, and dates/times for reached targets. Hide/show this section in Settings →
Dashboard. Existing visibility preferences are preserved; this section defaults to
visible. Dashboard period filters do not affect all-time milestones.

Play targets: 1, 100, 500, 1,000, 5,000, 10,000, 25,000, 50,000, 100,000,
250,000, 500,000, and 1,000,000. Hour targets: 1, 10, 50, 100, 500, 1,000,
2,500, 5,000, and 10,000.

Dates are calculated by sorting recorded listens and locating the timestamp that
crossed each threshold. They are displayed in the browser's local timezone, with
UTC timestamps in the API. Future-dated listens are excluded until their date.
Listening-hour targets use cumulative known track durations, not measured playback
or exact completion times. Unknown durations contribute zero time. Correcting a
duration may change calculated listening-hour dates.

All awards and dates are reconstructed from current history rather than stored as
permanent awards. Importing older listens, deleting plays, merging tracks, changing
artist credits, or correcting metadata may change boards and crossing dates.
A song here means a Koito track ID; separate versions remain separate until merged.
The old 20/40-play thresholds are not applied automatically to pre-2022 history.

Read-only APIs, protected by the existing login gate:

- `GET /apis/web/v1/track/{id}/certifications`
- `GET /apis/web/v1/artist/{id}/certifications`
- `GET /apis/web/v1/milestones`

No database migration is needed. Calculations stream the relevant listens; runtime
grows with history size. The milestone endpoint supplies totals and crossing dates
in one query. Demo mode uses the same rules and includes examples of all tiers,
including a song below Gold and one at 1,200 plays with Diamond in its history.

Validation:

```sh
go test ./internal/db ./internal/db/sqlite ./engine/handlers
cd client
corepack yarn typecheck
corepack yarn build
```

Manual check: open track and artist pages, expand award history, compare a 1,000-play
song with a 1,200-play song, inspect reached dates, and toggle dashboard milestones.
Check empty history against the isolated development backend with demo mode off.
