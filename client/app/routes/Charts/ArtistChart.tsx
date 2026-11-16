import RankedChart from "./RankedChart";
import { chartApiParams } from "~/utils/chartCalendar";
import { redirect, useLoaderData, type LoaderFunctionArgs } from "react-router";
import { type Artist, type PaginatedResponse, type Ranked } from "api/api";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const params = chartApiParams(url.searchParams);
  const res = await fetch(`/apis/web/v1/top/artists?${params}`, { signal: request.signal });
  if (!res.ok) {
    throw new Response("Failed to load top artists", { status: 500 });
  }

  const top_artists: PaginatedResponse<Ranked<Artist>> = await res.json();
  if (top_artists.items.length === 0 && top_artists.current_page > 1) {
    url.searchParams.set("page", "1");
    return redirect(`${url.pathname}?${url.searchParams}`);
  }
  return { top_artists };
}

export default function ArtistChart() {
  const { top_artists } = useLoaderData<{
    top_artists: PaginatedResponse<Ranked<Artist>>;
  }>();

  return <RankedChart type="artist" data={top_artists} />;
}
