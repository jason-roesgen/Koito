import Rewind from "~/components/rewind/Rewind";
import { type RewindStats } from "api/api";
import { useEffect, useState } from "react";
import type { LoaderFunctionArgs } from "react-router";
import { useLoaderData, useLocation, useNavigate } from "react-router";
import { getRewindParams } from "~/utils/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useAppContext } from "~/providers/AppProvider";

const months = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

const parseParams = (
  params: URLSearchParams,
): { year: number; month?: number } => {
  const rawYear = params.get("year");
  const rawMonth = params.get("month");

  if (!rawYear && !rawMonth) return getRewindParams();
  if (rawYear && rawMonth)
    return { year: parseInt(rawYear), month: parseInt(rawMonth) };
  if (rawYear) return { year: parseInt(rawYear) };

  const month = parseInt(rawMonth!);
  const now = new Date();
  const currentYear = now.getFullYear();
  const currentMonth = now.getMonth() + 1;
  return { year: month <= currentMonth ? currentYear : currentYear - 1, month };
};

export async function clientLoader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const isYtd = url.searchParams.get("period") === "ytd";
  const now = new Date();
  const { year, month } = isYtd
    ? { year: now.getFullYear(), month: undefined }
    : parseParams(url.searchParams);
  const params = new URLSearchParams();
  const rankBy = url.searchParams.get("rank_by") === "time" ? "time" : "plays";
  params.set("rank_by", rankBy);
  if (isYtd) {
    params.set("from", String(Math.floor(new Date(year, 0, 1).getTime() / 1000)));
    params.set("to", String(Math.floor(now.getTime() / 1000)));
  } else {
    params.set("year", String(year));
    if (month) params.set("month", String(month));
  }
  const res = await fetch(`/apis/web/v1/summary?${params}`);
  if (!res.ok) throw new Response("Failed to load summary", { status: 500 });

  const stats: RewindStats = await res.json();
  stats.title = isYtd
    ? `Your ${year} Year to Date`
    : `Your ${month && month > 0 ? months[month - 1] + " " : ""}${year} Rewind`;
  return { stats, year, month, isYtd, rankBy, asOf: now.toISOString() };
}

export default function RewindPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { stats, year: loadedYear, month: loadedMonth, isYtd, rankBy, asOf } = useLoaderData<typeof clientLoader>();
  const { firstActivity } = useAppContext();

  const [year, setYear] = useState(loadedYear);
  const [month, setMonth] = useState(loadedMonth);
  const [showYearly, setShowYearly] = useState(
    !loadedMonth || loadedMonth === 0,
  );
  const [showTime, setShowTime] = useState(false);

  useEffect(() => {
    setYear(loadedYear);
    setMonth(loadedMonth);
    setShowYearly(!loadedMonth);
  }, [loadedYear, loadedMonth, isYtd]);

  const updateParams = (updates: Record<string, string | null>) => {
    const nextParams = new URLSearchParams(location.search);
    for (const [key, val] of Object.entries(updates)) {
      val !== null ? nextParams.set(key, val) : nextParams.delete(key);
    }
    navigate(`/rewind?${nextParams.toString()}`, { replace: false });
  };

  const navigateMonth = (newMonth: number) => {
    setMonth(newMonth);
    updateParams({ period: null, year: year.toString(), month: newMonth.toString() });
  };

  const navigateYear = (direction: "prev" | "next") => {
    setYear(direction === "next" ? year + 1 : year - 1);
  };

  // month is 0-indexed (array index), matching JS Date convention
  function isFuture(month: number, year: number) {
    return new Date(year, month + 1) > new Date();
  }

  function isBeforeFirstActivity(month: number, year: number) {
    return !!firstActivity && new Date(year, month + 1) < firstActivity;
  }

  function isUnavailable(monthIndex: number, year: number) {
    return (
      isFuture(monthIndex, year) || isBeforeFirstActivity(monthIndex, year)
    );
  }

  const pgTitle = `${stats.title} - Koito`;

  return (
    <div className="w-full min-h-screen">
      <title>{pgTitle}</title>
      <meta property="og:title" content={pgTitle} />
      <meta name="description" content={pgTitle} />
      <div className="flex flex-col lg:flex-row items-center lg:items-start lg:mt-15 mt-5 gap-10 w-19/20 mx-auto px-5 md:px-10">
        <div className="flex flex-col items-start gap-4">
          <div className="flex flex-col items-center gap-4 mt-4 sm:mt-14 w-[250px] mx-auto">
            {!isYtd && !showYearly && (
              <>
                <div className="flex items-center gap-6 justify-around">
                  <button
                    onClick={() => navigateYear("prev")}
                    className="p-2 disabled:opacity-0"
                    disabled={
                      new Date(year - 1, month || 0) > new Date() ||
                      (firstActivity && year - 1 < firstActivity.getFullYear())
                    }
                  >
                    <ChevronLeft size={20} />
                  </button>
                  <p className="font-medium text-xl text-center w-30">{year}</p>
                  <button
                    onClick={() => navigateYear("next")}
                    className="p-2 disabled:opacity-0"
                    disabled={isFuture(0, year + 1)}
                  >
                    <ChevronRight size={20} />
                  </button>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  {months.map((m, i) => {
                    const isSelected =
                      month !== undefined &&
                      i === month - 1 &&
                      loadedYear === year;
                    const unavailable = isUnavailable(i, year);
                    return (
                      <button
                        key={m}
                        className={`px-12 py-2 rounded-(--border-radius)
                          ${isSelected ? "card" : "border-1 border-(--color-bg)"}
                          ${unavailable ? "color-fg-tertiary" : "hover:bg-(--color-bg-secondary)"}
                        `}
                        onClick={() => navigateMonth(i + 1)}
                        disabled={unavailable || isSelected}
                      >
                        {m.substring(0, 3)}
                      </button>
                    );
                  })}
                </div>
              </>
            )}
            {!isYtd && showYearly && firstActivity && (
              <div className="grid grid-cols-2 gap-2">
                {Array.from(
                  {
                    length:
                      new Date().getFullYear() - firstActivity.getFullYear(),
                  },
                  (_, index) => firstActivity.getFullYear() + index,
                ).map((y) => (
                  <button
                    key={y}
                    className={`px-10 py-2 rounded-(--border-radius) hover:bg-(--color-bg-secondary)
                      ${y === year && !month ? "card" : "border-1 border-(--color-bg)"}
                    `}
                    onClick={() => {
                      setYear(y);
                      setMonth(undefined);
                      updateParams({ period: null, year: y.toString(), month: null });
                    }}
                    disabled={(!month || month === 0) && y === year}
                  >
                    {y}
                  </button>
                ))}
              </div>
            )}
            <div className="flex flex-col gap-2 w-full">
              <label htmlFor="rewind-ranking">Rank by</label>
              <select
                id="rewind-ranking"
                className="p-2 border rounded-(--border-radius) bg-(--color-bg)"
                value={rankBy}
                onChange={event => updateParams({ rank_by: event.target.value })}
              >
                <option value="plays">Plays</option>
                <option value="time">Listening time</option>
              </select>
              {rankBy === "time" && (
                <p className="text-sm text-(--color-fg-secondary)">
                  Estimated from track durations per play. Unknown durations count as zero.
                </p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <input
                id="show-time-checkbox"
                type="checkbox"
                checked={showTime || rankBy === "time"}
                disabled={rankBy === "time"}
                onChange={(e) => setShowTime(e.target.checked)}
              />
              <label htmlFor="show-time-checkbox">Show time listened?</label>
            </div>
          </div>
        </div>
        <div className="flex flex-col items-center gap-4">
          <div className="flex flex-wrap justify-center gap-6 w-full mb-4">
            <button
              className="period-selector"
              onClick={() => {
                setShowYearly(false);
                if (isYtd) {
                  const now = new Date();
                  const previous = new Date(now.getFullYear(), now.getMonth() - 1, 1);
                  updateParams({ period: null, year: String(previous.getFullYear()), month: String(previous.getMonth() + 1) });
                }
              }}
              disabled={!isYtd && !showYearly}
            >
              MONTHLY
            </button>
            <button
              className="period-selector"
              onClick={() => {
                setShowYearly(true);
                if (isYtd) updateParams({ period: null, year: String(new Date().getFullYear() - 1), month: null });
              }}
              disabled={!isYtd && showYearly}
            >
              YEARLY
            </button>
            <button
              className="period-selector"
              onClick={() => navigate(`/rewind?period=ytd&rank_by=${rankBy}`)}
              disabled={isYtd}
            >
              YEAR TO DATE
            </button>
          </div>
          {isYtd && (
            <p className="text-sm text-(--color-fg-secondary) text-center">
              January 1 – {new Date(asOf).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" })}
              {" · Your local timezone · Through the time this summary loaded"}
            </p>
          )}
          {stats && <Rewind stats={stats} rankBy={rankBy} includeTime={showTime || rankBy === "time"} />}
        </div>
      </div>
    </div>
  );
}
