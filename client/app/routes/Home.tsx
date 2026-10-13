import type { Route } from "./+types/Home";
import TopTracks from "~/components/TopTracks";
import LastPlayed from "~/components/LastPlayed";
import ActivityGrid from "~/components/ActivityGrid";
import TopArtistsCard from "~/components/TopArtistsCard";
import AllTimeStats from "~/components/AllTimeStats";
import { useState } from "react";
import { useAppContext } from "~/providers/AppProvider";
import { sectionKeys } from "~/hooks/useDashboardPreferences";
import PeriodSelector from "~/components/PeriodSelector";
import TopAlbumsCard from "~/components/TopAlbumsCard";

export function meta({}: Route.MetaArgs) {
  return [{ title: "Koito" }, { name: "description", content: "Koito" }];
}

export default function Home() {
  const [period, setPeriod] = useState("week");
  const { dashboard } = useAppContext();
  const { visible, ready } = dashboard;

  const showRankings = visible.artists || visible.albums || visible.tracks;
  const showAny = sectionKeys.some(key => visible[key]);

  return (
    <main className="flex grow justify-center pb-4 w-full">
      <div className="flex-1 flex flex-col items-center gap-10 md:gap-12 min-h-0 mt-8 sm:mt-10 mx-4 sm:mx-10">
        {ready && (
          <>
            {!showAny && (
              <p role="status" className="text-(--color-fg-secondary) text-center">
                All sections are hidden. Open Settings → Dashboard in the sidebar to show a section or reset to defaults.
              </p>
            )}
            {(visible.stats || visible.activity) && (
              <div className="flex flex-col lg:flex-row gap-10 md:gap-20">
                {visible.stats && <AllTimeStats />}
                {visible.activity && <ActivityGrid configurable />}
              </div>
            )}
            {showRankings && <PeriodSelector setter={setPeriod} current={period} />}
            {(showRankings || visible.recent) && (
              <div className="container justify-center flex flex-wrap gap-10">
                {visible.artists && <TopArtistsCard period={period} />}
                {visible.albums && <TopAlbumsCard period={period} />}
                {visible.tracks && (
                  <TopTracks
                    period={period}
                    limit={10}
                    className="min-w-[350px] w-full max-w-[750px] 2xl:max-w-[450px]"
                  />
                )}
                {visible.recent && <LastPlayed showNowPlaying={true} limit={28} showSeeMore />}
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}
