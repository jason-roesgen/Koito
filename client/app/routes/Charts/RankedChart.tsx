import type { Album, Artist, Track, Ranked, PaginatedResponse } from "api/api";
import TopItemList from "~/components/TopItemList";
import ChartCalendarControls from "~/components/ChartCalendarControls";
import ChartLayout from "./ChartLayout";

const charts = {
  track: { title: "Top Tracks", plural: "tracks", label: "Track" },
  album: { title: "Top Albums", plural: "albums", label: "Album" },
  artist: { title: "Top Artists", plural: "artists", label: "Artist" },
} as const;

export default function RankedChart({ type, data }: {
  type: keyof typeof charts;
  data: PaginatedResponse<Ranked<Track | Album | Artist>>;
}) {
  const chart = charts[type];
  return (
    <ChartLayout
      title={chart.title}
      initialData={data}
      endpoint={`chart/top/${chart.plural}`}
      controls={<ChartCalendarControls />}
      render={({ data, page, onNext, onPrev }) => (
        <div className="flex flex-col gap-5 text-sm md:text-[16px] w-11/12 max-w-[1000px]">
          {data.items.length === 0 && <p role="status">No {chart.plural} listened to in this period.</p>}
          <TopItemList ranked separators data={data} className="w-full" type={type} slug="" />
          <nav aria-label={`${chart.label} result pages`} className="flex flex-wrap items-center justify-center gap-3 sm:gap-6">
            <button className="default" onClick={onPrev} disabled={page <= 1}>
              Previous page
            </button>
            <span aria-live="polite">
              Page {page} of {Math.max(1, Math.ceil(data.total_record_count / data.items_per_page))}
            </span>
            <button className="default" onClick={onNext} disabled={!data.has_next_page}>
              Next page
            </button>
          </nav>
        </div>
      )}
    />
  );
}
