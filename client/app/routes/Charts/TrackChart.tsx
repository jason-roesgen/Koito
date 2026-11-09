import TopItemList from "~/components/TopItemList";
import ChartLayout from "./ChartLayout";
import TrackCalendarControls from "~/components/TrackCalendarControls";
import { trackApiParams } from "~/utils/trackCalendar";
import { redirect, useLoaderData, type LoaderFunctionArgs } from "react-router";
import { type Track, type PaginatedResponse, type Ranked } from "api/api";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const params = trackApiParams(url.searchParams);

  const res = await fetch(
    `/apis/web/v1/top/tracks?${params}`,
    { signal: request.signal },
  );
  if (!res.ok) {
    throw new Response("Failed to load top tracks", { status: 500 });
  }

  const top_tracks: PaginatedResponse<Ranked<Track>> = await res.json();
  if (top_tracks.items.length === 0 && top_tracks.current_page > 1) {
    url.searchParams.set("page", "1");
    return redirect(`${url.pathname}?${url.searchParams}`);
  }
  return { top_tracks };
}

export default function TrackChart() {
  const { top_tracks: initialData } = useLoaderData<{
    top_tracks: PaginatedResponse<Ranked<Track>>;
  }>();

  return (
    <ChartLayout
      title="Top Tracks"
      initialData={initialData}
      endpoint="chart/top/tracks"
      controls={<TrackCalendarControls />}
      render={({ data, page, onNext, onPrev }) => (
        <div className="flex flex-col gap-5 text-sm md:text-[16px] w-11/12 max-w-[1000px]">
          {data.items.length === 0 && <p role="status">No tracks listened to in this period.</p>}
          <TopItemList
            ranked
            separators
            data={data}
            className="w-full"
            type="track"
            slug=""
          />
          <nav aria-label="Track result pages" className="flex flex-wrap items-center justify-center gap-3 sm:gap-6">
            <button className="default" onClick={onPrev} disabled={page <= 1}>
              Previous page
            </button>
            <span aria-live="polite">
              Page {page} of {Math.max(1, Math.ceil(data.total_record_count / data.items_per_page))}
            </span>
            <button
              className="default"
              onClick={onNext}
              disabled={!data.has_next_page}
            >
              Next page
            </button>
          </nav>
        </div>
      )}
    />
  );
}
