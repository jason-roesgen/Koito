import { useLocation, useNavigate } from "react-router";
import PeriodSelector from "./PeriodSelector";
import { useAppContext } from "~/providers/AppProvider";
import {
  isoWeek, isoWeekStart, localDateValue, moveTrackPeriod, readTrackCalendar,
  trackCalendarParams, trackCalendarRange, type TrackPeriod,
} from "~/utils/trackCalendar";

export default function TrackCalendarControls() {
  const location = useLocation();
  const navigate = useNavigate();
  const { firstActivity } = useAppContext();
  const params = new URLSearchParams(location.search);
  const now = new Date();
  const calendar = readTrackCalendar(params, now);
  const { period, date } = calendar;
  const { start, end } = trackCalendarRange(calendar);
  const selectedWeek = isoWeek(date);
  const year = period === "week" ? selectedWeek.year : date.getFullYear();
  const currentYear = period === "week" ? isoWeek(now).year : now.getFullYear();
  const earliestYear = Math.min(firstActivity?.getFullYear() ?? currentYear - 10, year);
  const years = Array.from({ length: currentYear - earliestYear + 1 }, (_, i) => currentYear - i);
  const formatter = new Intl.DateTimeFormat(undefined, { year: "numeric", month: "long", day: "numeric" });
  const lastDay = new Date(end.getTime() - 1000);
  const selectClass = "px-2 py-1 rounded border border-gray-400";

  const select = (nextDate: Date, nextPeriod: TrackPeriod = period) => {
    const next = trackCalendarParams(params, { period: nextPeriod, date: nextDate });
    navigate(`${location.pathname}?${next}`);
  };

  return (
    <div className="flex flex-col gap-3 items-start">
      <div className="max-w-full overflow-x-auto">
        <PeriodSelector current={period} setter={(value: TrackPeriod) => select(now, value)} disableCache />
      </div>
      {period !== "all_time" && (
        <div className="flex flex-wrap items-center gap-3">
          <button className="default" onClick={() => select(moveTrackPeriod(calendar, -1))}>
            Previous {period}
          </button>
          {period === "day" ? (
            <label className="flex items-center gap-2">
              Date
              <input type="date" className={selectClass} value={localDateValue(date)} max={localDateValue(now)}
                onChange={(event) => {
                  if (event.target.value) {
                    const next = readTrackCalendar(new URLSearchParams({ period: "day", date: event.target.value }), now);
                    select(next.date);
                  }
                }} />
            </label>
          ) : (
            <>
              <label className="flex items-center gap-2">
                {period === "week" ? "ISO year" : "Year"}
                <select aria-label={period === "week" ? "ISO year" : "Year"} className={selectClass} value={year} onChange={(event) => {
                  const nextYear = Number(event.target.value);
                  select(period === "week"
                    ? isoWeekStart(nextYear, Math.min(selectedWeek.week, isoWeek(new Date(nextYear, 11, 28)).week))
                    : new Date(nextYear, date.getMonth(), 1));
                }}>
                  {years.map((value) => <option key={value} value={value}>{value}</option>)}
                </select>
              </label>
              {period === "month" && (
                <label className="flex items-center gap-2">
                  Month
                  <select aria-label="Month" className={selectClass} value={date.getMonth()} onChange={(event) => select(new Date(year, Number(event.target.value), 1))}>
                    {Array.from({ length: 12 }, (_, month) => (
                      <option key={month} value={month} disabled={new Date(year, month, 1) > now}>
                        {new Intl.DateTimeFormat(undefined, { month: "long" }).format(new Date(year, month, 1))}
                      </option>
                    ))}
                  </select>
                </label>
              )}
              {period === "week" && (
                <label className="flex items-center gap-2">
                  Week
                  <select aria-label="Week" className={selectClass} value={selectedWeek.week} onChange={(event) => select(isoWeekStart(year, Number(event.target.value)))}>
                    {Array.from({ length: isoWeek(new Date(year, 11, 28)).week }, (_, i) => i + 1).map((week) => (
                      <option key={week} value={week} disabled={isoWeekStart(year, week) > now}>{week}</option>
                    ))}
                  </select>
                </label>
              )}
            </>
          )}
          <button className="default" disabled={start >= trackCalendarRange({ period, date: now }).start}
            onClick={() => select(moveTrackPeriod(calendar, 1))}>
            Next {period}
          </button>
        </div>
      )}
      <p className="text-sm text-color-fg-secondary">
        {period === "all_time" ? "All Time" : period === "day" ? formatter.format(start)
          : `${formatter.format(start)} – ${formatter.format(lastDay)}`}
        {period !== "all_time" && " · Your local timezone"}
      </p>
    </div>
  );
}
