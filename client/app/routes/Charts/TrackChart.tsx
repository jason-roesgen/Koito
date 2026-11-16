import RankedChart from "./RankedChart";
import { chartApiParams } from "~/utils/chartCalendar";
import { redirect, useLoaderData, type LoaderFunctionArgs } from "react-router";
import { type Track, type PaginatedResponse, type Ranked } from "api/api";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const params = chartApiParams(url.searchParams);
  const res = await fetch(`/apis/web/v1/top/tracks?${params}`, { signal: request.signal });
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
  const { top_tracks } = useLoaderData<{
    top_tracks: PaginatedResponse<Ranked<Track>>;
  }>();

  return <RankedChart type="track" data={top_tracks} />;
}
