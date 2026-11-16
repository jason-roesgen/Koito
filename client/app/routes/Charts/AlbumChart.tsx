import RankedChart from "./RankedChart";
import { chartApiParams } from "~/utils/chartCalendar";
import { redirect, useLoaderData, type LoaderFunctionArgs } from "react-router";
import { type Album, type PaginatedResponse, type Ranked } from "api/api";

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const params = chartApiParams(url.searchParams);
  const res = await fetch(`/apis/web/v1/top/albums?${params}`, { signal: request.signal });
  if (!res.ok) {
    throw new Response("Failed to load top albums", { status: 500 });
  }

  const top_albums: PaginatedResponse<Ranked<Album>> = await res.json();
  if (top_albums.items.length === 0 && top_albums.current_page > 1) {
    url.searchParams.set("page", "1");
    return redirect(`${url.pathname}?${url.searchParams}`);
  }
  return { top_albums };
}

export default function AlbumChart() {
  const { top_albums } = useLoaderData<{
    top_albums: PaginatedResponse<Ranked<Album>>;
  }>();

  return <RankedChart type="album" data={top_albums} />;
}
