import type { RepeatSession, RepeatSessionPage } from "api/repeatSessions";
import { useId, useState } from "react";
import { CalendarDays, ChevronRight, Headphones, ListFilter, Repeat2, Search, Trophy } from "lucide-react";
import { Form, Link, useLoaderData, useSearchParams, type LoaderFunctionArgs } from "react-router";
import Image from "~/components/primitives/Image";
import { repeatSessionDateBounds } from "~/utils/chartCalendar";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const params = new URL(request.url).searchParams;
  let dates: ReturnType<typeof repeatSessionDateBounds>;
  try {
    dates = repeatSessionDateBounds(params.get("from") ?? "", params.get("to") ?? "");
  } catch {
    throw new Response("Choose a valid start-date range.", { status: 400 });
  }
  for (const [key, value] of [["from", dates.start], ["to", dates.end]] as const) {
    if (value === undefined) params.delete(key);
    else params.set(key, String(value));
  }
  const response = await fetch(`/apis/web/v1/repeat-sessions?${params}`, { signal: request.signal });
  if (!response.ok) throw new Response("Failed to load Repeat Sessions", { status: response.status });
  return (await response.json()) as RepeatSessionPage;
}

function durationLabel(seconds: number, exact = false) {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor(seconds / 60) % 60;
  const remainder = seconds % 60;
  return [
    hours ? `${hours.toLocaleString()}h` : "",
    minutes ? `${minutes}m` : "",
    !hours && !minutes || exact && remainder ? `${remainder}s` : "",
  ].filter(Boolean).join(" ");
}

function ListenDate({ value, compact = false }: { value: string; compact?: boolean }) {
  const date = new Date(value);
  return <time dateTime={value}>{compact ? date.toLocaleDateString(undefined, {
    year: "numeric", month: "short", day: "numeric",
  }) : date.toLocaleString()}</time>;
}

const muted = "text-(--color-fg-secondary)";
const accentText = "text-[color:color-mix(in_srgb,var(--color-primary)_56%,var(--color-fg))]";
const control = "rounded-lg border border-(--color-bg-tertiary) bg-(--color-bg) text-sm";
const trigger = `${control} flex min-h-10 cursor-pointer list-none items-center gap-2 px-3 [&::-webkit-details-marker]:hidden`;
const sortLabels = { plays: "Plays", date: "Date", time: "Listening time" };

function SessionFilters({ params }: { params: URLSearchParams }) {
  const [dateError, setDateError] = useState("");
  const hasDates = !!(params.get("from") || params.get("to"));
  const sort = params.get("sort") || "plays";
  const sortLabel = sortLabels[sort as keyof typeof sortLabels] ?? sortLabels.plays;
  return (
    <Form method="get" className="mt-4" onSubmit={event => {
      const fields = new FormData(event.currentTarget);
      try {
        repeatSessionDateBounds(String(fields.get("from")), String(fields.get("to")));
        setDateError("");
      } catch {
        event.preventDefault();
        setDateError("Choose a valid date range. The end date must be on or after the start date.");
      }
    }}>
      <div className="flex flex-wrap items-start gap-2">
        <label className={`${control} flex min-h-10 min-w-0 basis-full items-center gap-2 px-3 sm:basis-52 sm:flex-1`}>
          <Search size={16} aria-hidden="true" className={`shrink-0 ${muted}`} />
          <span className="sr-only">Search track or artist</span>
          <input type="search" name="q" maxLength={200} placeholder="Search music" defaultValue={params.get("q") ?? ""}
            className="min-w-0 flex-1 py-2" />
        </label>
        <details name="session-filters" className="relative">
          <summary className={trigger}><CalendarDays size={15} aria-hidden="true" />{hasDates ? "Date range" : "All time"}</summary>
          <div className="card mt-2 grid max-w-full gap-3 p-3 sm:absolute sm:right-0 sm:z-10 sm:w-56 sm:max-w-none">
            <label className="grid gap-1 text-sm">Started from
              <input type="date" name="from" defaultValue={params.get("from") ?? ""} className={`${control} w-full min-w-0 p-2`} />
            </label>
            <label className="grid gap-1 text-sm">Through
              <input type="date" name="to" defaultValue={params.get("to") ?? ""} className={`${control} w-full min-w-0 p-2`} />
            </label>
          </div>
        </details>
        <details name="session-filters" className="relative min-w-0">
          <summary className={trigger}><ListFilter size={15} aria-hidden="true" />{sortLabel}</summary>
          <div className="card mt-2 grid max-w-full gap-3 p-3 sm:absolute sm:right-0 sm:z-10 sm:w-56 sm:max-w-none">
            <label className="grid gap-1 text-sm">Sort by
              <select name="sort" defaultValue={sort} className={`${control} w-full min-w-0 p-2`}>
                <option value="plays">Recorded plays</option>
                <option value="date">Start date</option>
                <option value="time">Estimated listening time</option>
              </select>
            </label>
            <label className="grid gap-1 text-sm">Order
              <select name="order" defaultValue={params.get("order") || "desc"} className={`${control} w-full min-w-0 p-2`}>
                <option value="desc">Descending</option>
                <option value="asc">Ascending</option>
              </select>
            </label>
          </div>
        </details>
        <button className={`${control} min-h-10 px-3 hover:bg-(--color-bg-secondary)`} type="submit">Apply</button>
        {params.size > 0 && <Link to="/listening-evolution" className={`self-center px-1 text-sm underline ${muted}`}>Reset</Link>}
      </div>
      {dateError && <p role="alert" className="mt-2 text-sm">{dateError}</p>}
    </Form>
  );
}

function SessionTrack({ session, featured = false }: { session: RepeatSession; featured?: boolean }) {
  return (
    <div className="min-w-0 [overflow-wrap:anywhere]">
      <Link to={`/track/${session.track.id}`} className={`font-semibold hover:underline ${featured ? "text-xl sm:text-2xl" : "text-sm sm:text-base"}`}>
        {session.track.title}
      </Link>
      {session.track.artists.length > 0 && <p className={`${muted} mt-0.5 ${featured ? "text-sm sm:text-base" : "text-xs sm:text-sm"}`}>
        {session.track.artists.map((artist, index) => (
          <span key={artist.id}>{index > 0 && ", "}<Link to={`/artist/${artist.id}`} className="hover:underline">{artist.name}</Link></span>
        ))}
      </p>}
      <p className={`${muted} mt-0.5 ${featured ? "text-sm" : "text-xs"}`}><ListenDate value={session.started_at} compact /></p>
    </div>
  );
}

function Overview({ overview }: { overview: RepeatSessionPage["overview"] }) {
  const stats = [
    { icon: Repeat2, label: "Repeat sessions", value: overview.session_count.toLocaleString() },
    { icon: Trophy, label: "Longest run", value: (overview.longest_session?.play_count ?? 0).toLocaleString() },
    { icon: Headphones, label: "Est. listening", value: overview.sessions_with_estimates ? durationLabel(overview.estimated_seconds) : "—" },
  ];
  return (
    <>
      <dl aria-label="All-time repeat session statistics" className="grid grid-cols-3 gap-2 sm:gap-3">
        {stats.map(({ icon: Icon, label, value }) => (
          <div className="card flex min-w-0 flex-col rounded-xl! p-2.5 sm:p-3.5" key={label}>
            <dt className={`order-last mt-0.5 text-xs ${muted}`}>{label}</dt>
            <dd className="text-xl font-semibold tabular-nums [overflow-wrap:anywhere] sm:text-2xl">
              <Icon size={17} className={`mb-1.5 ${accentText}`} aria-hidden="true" />
              {value}
            </dd>
          </div>
        ))}
      </dl>
      {overview.session_count > overview.sessions_with_estimates &&
        <p className={`mt-2 text-xs ${muted}`}>Estimated listening excludes sessions with unknown track duration.</p>}
    </>
  );
}

function FeaturedSession({ session }: { session: RepeatSession }) {
  const bars = Math.min(session.play_count, 58);
  return (
    <article className="my-4 rounded-2xl border border-(--color-bg-tertiary) bg-[color-mix(in_srgb,var(--color-bg-secondary)_88%,var(--color-primary))] p-4 sm:p-5">
      <h3 className={`flex items-center gap-2 text-xs! font-semibold! uppercase tracking-wider sm:text-sm! ${accentText}`}>
        <Trophy size={16} aria-hidden="true" /> All-time longest session
      </h3>
      <div className="my-4 flex items-center gap-3 sm:gap-4">
        <Image src={session.track.image.medium} size={100} alt="" className="shrink-0 !rounded-xl max-sm:!size-[74px]" />
        <SessionTrack session={session} featured />
      </div>
      <dl className="grid grid-cols-2 gap-3">
        <div className="flex min-w-0 flex-col">
          <dt className={`order-last mt-1 text-xs sm:text-sm ${muted}`}>Consecutive plays</dt>
          <dd className={`text-3xl font-semibold tabular-nums [overflow-wrap:anywhere] sm:text-5xl ${accentText}`}>{session.play_count.toLocaleString()}</dd>
        </div>
        <div className="flex min-w-0 flex-col">
          <dt className={`order-last mt-1 text-xs sm:text-sm ${muted}`}>Estimated listening</dt>
          <dd className={`font-semibold tabular-nums [overflow-wrap:anywhere] ${session.estimated_seconds === null ? "pt-1 text-lg sm:text-2xl" : "text-3xl sm:text-5xl"}`}>
            {session.estimated_seconds === null ? "Unavailable" : durationLabel(session.estimated_seconds)}
          </dd>
        </div>
      </dl>
      <div aria-hidden="true" className="mt-5 flex h-11 items-center gap-0.5 text-(--color-primary) sm:h-14 sm:gap-1">
        {Array.from({ length: bars }, (_, index) => <span key={index} className="w-full min-w-0 rounded-full bg-current"
          style={{ height: `${[92, 72, 55, 76, 59][index % 5]}%`, opacity: 0.48 + (index / bars) * 0.4 }} />)}
      </div>
    </article>
  );
}

function SessionCard({ session, position, longest }: { session: RepeatSession; position: number; longest: number }) {
  const detailsId = useId();
  const [expanded, setExpanded] = useState(false);
  return (
    <li className="card min-w-0 rounded-xl! p-3 sm:p-3.5">
      <div className="grid grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-2 sm:gap-3">
        <span className={`text-xs tabular-nums ${muted}`}>#{position.toLocaleString()}</span>
        <Image src={session.track.image.small} size={46} alt="" lazy className="shrink-0 !rounded-lg max-sm:!size-9" />
        <SessionTrack session={session} />
        <button type="button" className="flex min-h-11 cursor-pointer items-center gap-1.5 text-right sm:gap-2"
          aria-label={`${expanded ? "Hide" : "Show"} session details for ${session.track.title}`}
          aria-controls={detailsId} aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>
          <span className="grid">
            <strong className={`text-lg font-semibold tabular-nums sm:text-xl ${accentText}`}>{session.play_count.toLocaleString()}</strong>
            <span className={`whitespace-nowrap text-xs ${muted}`}>plays{session.estimated_seconds !== null && ` · ${durationLabel(session.estimated_seconds)}`}</span>
          </span>
          <ChevronRight size={15} aria-hidden="true" className={`${muted} ${expanded ? "rotate-90" : ""}`} />
        </button>
      </div>
      <div aria-hidden="true" className="mt-2.5 h-[3px] overflow-hidden rounded-full bg-(--color-bg-tertiary)">
        <div className="h-full rounded-full bg-(--color-primary) opacity-75" style={{ width: `${session.play_count / longest * 100}%` }} />
      </div>
      <div id={detailsId} hidden={!expanded}>
        <dl className="mt-3 grid grid-cols-2 gap-3 border-t border-(--color-bg-tertiary) pt-3 text-xs sm:grid-cols-4 sm:text-sm">
          <div><dt className={muted}>First recorded play</dt><dd><ListenDate value={session.started_at} /></dd></div>
          <div><dt className={muted}>Last recorded play</dt><dd><ListenDate value={session.ended_at} /></dd></div>
          <div><dt className={muted}>Estimated listening</dt><dd>{session.estimated_seconds === null ? "Unavailable" : durationLabel(session.estimated_seconds, true)}</dd></div>
          <div><dt className={muted}>Elapsed span</dt><dd>{durationLabel(session.elapsed_seconds, true)}</dd></div>
        </dl>
      </div>
    </li>
  );
}

export default function ListeningEvolution() {
  const data = useLoaderData<typeof clientLoader>();
  const [params] = useSearchParams();
  const pages = Math.max(1, Math.ceil(data.total_record_count / data.items_per_page));
  const pageLink = (page: number) => {
    const next = new URLSearchParams(params);
    next.set("page", String(page));
    return `?${next}`;
  };
  const longest = data.overview.longest_session;
  return (
    <main className="mx-auto my-4 w-[calc(100%-1rem)] max-w-[960px] min-w-0 rounded-2xl border border-(--color-bg-tertiary) p-3 sm:my-8 sm:w-11/12 sm:p-5">
      <h1 className="text-2xl! leading-tight! font-semibold! sm:text-3xl!">Listening Evolution</h1>
      <p className={`mt-1 text-sm ${muted}`}>Discover the patterns behind your music history.</p>
      <section aria-labelledby="repeat-sessions" className="mt-4">
        <h2 id="repeat-sessions" className="mb-4 w-fit border-b-2 border-(--color-primary) pb-2 text-base! font-semibold!">Repeat Sessions</h2>
        <Overview overview={data.overview} />
        {longest && <FeaturedSession session={longest} />}
        <section aria-labelledby="session-history" className="mt-5">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h3 id="session-history" className="text-lg! font-semibold! normal-case!">Session history</h3>
            <p role="status" className={`text-sm ${muted}`}>
              {data.total_record_count.toLocaleString()} {data.total_record_count === 1 ? "session" : "sessions"}
            </p>
          </div>
          <SessionFilters key={params.toString()} params={params} />
          {data.items.length ? (
            <ul className="mt-4 grid gap-2.5">
              {data.items.map((session, index) => <SessionCard key={`${session.track.id}-${session.started_at}`} session={session}
                position={(data.current_page - 1) * data.items_per_page + index + 1} longest={longest?.play_count ?? session.play_count} />)}
            </ul>
          ) : <p className="card mt-4 rounded-xl p-5">No repeat sessions match these filters.</p>}
          {pages > 1 && (
            <nav aria-label="Result pages" className="mt-5 flex flex-wrap items-center justify-center gap-4 text-sm">
              {data.current_page > 1 && <Link className="underline" to={pageLink(data.current_page - 1)}>Previous page</Link>}
              <span>Page {data.current_page} of {pages}</span>
              {data.has_next_page && <Link className="underline" to={pageLink(data.current_page + 1)}>Next page</Link>}
            </nav>
          )}
        </section>
        <p className={`mt-5 text-xs ${muted}`}>
          Repeat sessions contain {data.minimum_plays} or more consecutive recorded plays. Listening time uses stored track duration. Elapsed span is the time between the first and last recorded plays.
        </p>
      </section>
    </main>
  );
}
