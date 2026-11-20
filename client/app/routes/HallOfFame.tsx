import {
  Form,
  Link,
  useLoaderData,
  useSearchParams,
  type LoaderFunctionArgs,
} from "react-router";
import { type HallOfFame as HallData } from "api/certifications";
import { AwardBadge, CertificationDate } from "~/components/CertificationBoard";
import Image from "~/components/primitives/Image";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const params = new URL(request.url).searchParams;
  const res = await fetch(`/apis/web/v1/hall-of-fame?${params}`, {
    signal: request.signal,
  });
  if (!res.ok)
    throw new Response("Failed to load Hall of Fame", { status: res.status });
  return (await res.json()) as HallData;
}

export default function HallOfFame() {
  const data = useLoaderData<typeof clientLoader>();
  const [params, setParams] = useSearchParams();
  const view = params.get("view") === "songs" ? "songs" : "artists";
  const tier = params.get("tier") || "";
  const update = (values: Record<string, string>) => {
    const next = new URLSearchParams(params);
    next.set("page", "1");
    for (const [key, value] of Object.entries(values))
      value ? next.set(key, value) : next.delete(key);
    setParams(next);
  };
  const pages = Math.max(
    1,
    Math.ceil(data.total_record_count / data.items_per_page),
  );
  return (
    <main className="w-11/12 max-w-[1100px] mx-auto py-8 sm:py-12 min-w-0">
      <h1 className="text-3xl sm:text-4xl font-bold">Hall of Fame</h1>
      <p className="mt-3 text-(--color-fg-secondary)">
        Your personal listening awards, across all recorded history.
      </p>
      <div
        className="grid grid-cols-2 sm:grid-cols-5 gap-3 my-6"
        aria-label="All-time certification totals"
      >
        {[
          ["Certified songs", data.certified_songs],
          ["Certified artists", data.certified_artists],
          ["📀 Gold songs", data.counts.gold],
          ["💿 Platinum songs", data.counts.platinum],
          ["💎 Diamond songs", data.counts.diamond],
        ].map(([label, value]) => (
          <div className="card p-4" key={label}>
            <p className="text-2xl font-semibold">{value.toLocaleString()}</p>
            <p className="text-sm text-(--color-fg-secondary)">{label}</p>
          </div>
        ))}
      </div>
      <p className="text-sm text-(--color-fg-secondary)">
        Each song counts once under its current certificate. Collaborations
        count for every credited artist. Counters show all-time totals,
        regardless of filters.
      </p>
      <nav
        aria-label="Hall of Fame sections"
        className="card flex gap-3 my-6 w-fit p-2"
      >
        {(["artists", "songs"] as const).map((tab) => (
          <button
            key={tab}
            className="period-selector capitalize px-5 py-2"
            disabled={view === tab}
            aria-pressed={view === tab}
            onClick={() => update({ view: tab })}
          >
            {tab}
          </button>
        ))}
      </nav>
      <div className="card p-4 flex flex-wrap items-end gap-4">
        <Form
          method="get"
          className="flex gap-2 items-end min-w-0 flex-wrap"
          key={params.toString()}
        >
          <input type="hidden" name="view" value={view} />
          <input type="hidden" name="tier" value={tier} />
          <input
            type="hidden"
            name="sort"
            value={params.get("sort") || "count"}
          />
          <input
            type="hidden"
            name="order"
            value={params.get("order") || "desc"}
          />
          <label className="flex flex-col gap-2">
            Search {view}
            <input
              type="text"
              className="max-w-full w-52 p-2 rounded-(--border-radius) bg-(--color-bg)"
              name="q"
              maxLength={200}
              defaultValue={params.get("q") || ""}
              placeholder={
                view === "artists" ? "Artist name" : "Song or artist"
              }
            />
          </label>
          <button type="submit" className="default">
            Search
          </button>
        </Form>
        <label className="flex flex-col gap-2">
          Current tier
          <select
            className="p-2 border rounded-(--border-radius) bg-(--color-bg)"
            value={tier}
            onChange={(e) => update({ tier: e.target.value })}
          >
            <option value="">All tiers</option>
            <option value="gold">Gold</option>
            <option value="platinum">Platinum</option>
            <option value="diamond">Diamond</option>
          </select>
        </label>
        <label className="flex flex-col gap-2">
          Sort by
          <select
            className="p-2 border rounded-(--border-radius) bg-(--color-bg)"
            value={params.get("sort") || "count"}
            onChange={(e) => update({ sort: e.target.value })}
          >
            <option value="count">
              {view === "artists"
                ? tier
                  ? "Songs in selected tier"
                  : "Certified songs"
                : "Plays"}
            </option>
            <option value="name">Name</option>
            <option value="date">
              {view === "artists" ? "Latest current award" : "Date earned"}
            </option>
          </select>
        </label>
        <label className="flex flex-col gap-2">
          Order
          <select
            className="p-2 border rounded-(--border-radius) bg-(--color-bg)"
            value={params.get("order") || "desc"}
            onChange={(e) => update({ order: e.target.value })}
          >
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
        </label>
      </div>
      <p className="text-sm mt-5 text-(--color-fg-secondary)" role="status">
        {data.total_record_count.toLocaleString()} matching {view}
      </p>
      {data.total_record_count === 0 ? (
        <div className="card p-6 mt-4">
          No certified {view} match these filters. Songs earn their first Gold
          at 200 plays.
        </div>
      ) : view === "artists" ? (
        <ul className="flex flex-col gap-3 mt-4">
          {data.artists.map((artist) => (
            <li
              key={artist.id}
              className="card p-4 flex gap-4 items-start sm:items-center flex-wrap"
            >
              <Link to={`/artist/${artist.id}`} className="shrink-0">
                <Image
                  src={artist.image.small}
                  size={56}
                  alt={artist.name}
                  lazy
                />
              </Link>
              <div className="flex-1 min-w-0">
                <Link
                  to={`/artist/${artist.id}`}
                  className="font-semibold hover:underline break-words"
                >
                  {artist.name}
                </Link>
                <p className="text-sm mt-1">
                  {artist.certified_songs.toLocaleString()} certified songs
                </p>
                <p className="text-xs mt-1 text-(--color-fg-secondary)">
                  Latest current award:{" "}
                  <CertificationDate value={artist.reached_at} />
                </p>
              </div>
              <div
                className="flex flex-wrap gap-x-5 gap-y-2 text-sm"
                aria-label="Current song certificates"
              >
                <span>📀 {artist.counts.gold.toLocaleString()} Gold</span>
                <span>
                  💿 {artist.counts.platinum.toLocaleString()} Platinum
                </span>
                <span>💎 {artist.counts.diamond.toLocaleString()} Diamond</span>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <ul className="flex flex-col gap-3 mt-4">
          {data.songs.map((song) => (
            <li
              key={song.track_id}
              className="card p-4 flex gap-4 items-start flex-wrap"
            >
              <Link to={`/track/${song.track_id}`} className="shrink-0">
                <Image src={song.image.small} size={56} alt={song.title} lazy />
              </Link>
              <div className="flex-1 min-w-0">
                <Link
                  to={`/track/${song.track_id}`}
                  className="font-semibold hover:underline break-words"
                >
                  {song.title}
                </Link>
                <div className="text-sm mt-1">
                  {song.artists.map((artist, i) => (
                    <span key={artist.id}>
                      {i > 0 && ", "}
                      <Link
                        to={`/artist/${artist.id}`}
                        className="hover:underline"
                      >
                        {artist.name}
                      </Link>
                    </span>
                  ))}
                </div>
                <p className="text-sm mt-1 text-(--color-fg-secondary)">
                  {song.plays.toLocaleString()} plays ·{" "}
                  <CertificationDate value={song.current.reached_at} />
                </p>
              </div>
              <AwardBadge award={song.current} />
            </li>
          ))}
        </ul>
      )}
      <nav
        aria-label="Result pages"
        className="flex justify-center flex-wrap items-center gap-4 mt-6 text-sm"
      >
        <button
          className="default"
          disabled={data.current_page <= 1}
          onClick={() => update({ page: String(data.current_page - 1) })}
        >
          Previous page
        </button>
        <span>
          Page {data.current_page} of {pages}
        </span>
        <button
          className="default"
          disabled={!data.has_next_page}
          onClick={() => update({ page: String(data.current_page + 1) })}
        >
          Next page
        </button>
      </nav>
      <p className="text-xs mt-6 text-(--color-fg-secondary)">
        Gold: 200 plays · Platinum: 400 · Diamond: 1,000. Current certificates
        follow each completed 200-play step; Diamond takes priority at shared
        thresholds. Dates use your local timezone. Open a song or artist for
        award history. Imports and corrections can change awards.
      </p>
    </main>
  );
}
