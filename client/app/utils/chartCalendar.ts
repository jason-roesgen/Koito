export type ChartPeriod = "day" | "week" | "month" | "year" | "all_time";
export type ChartCalendar = {
  period: ChartPeriod;
  date: Date;
  range?: { start: Date; end: Date };
};
export type RankingTimeframe = { period: string; from?: number; to?: number };

export function localDateValue(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

export function isoWeek(date: Date): { year: number; week: number } {
  const thursday = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  thursday.setUTCDate(thursday.getUTCDate() + 4 - (thursday.getUTCDay() || 7));
  const year = thursday.getUTCFullYear();
  const week = Math.ceil(((thursday.getTime() - Date.UTC(year, 0, 1)) / 86400000 + 1) / 7);
  return { year, week };
}

export function isoWeekStart(year: number, week: number): Date {
  const jan4 = new Date(year, 0, 4);
  return new Date(year, 0, 4 - ((jan4.getDay() + 6) % 7) + (week - 1) * 7);
}

// Dashboard cards retain rolling periods, but freeze the query and drilldown to
// the same integer-second window instead of reinterpreting it as a calendar period.
export function rankingTimeframe(period: string, now = new Date()): RankingTimeframe {
  if (period === "all_time") return { period };
  const start = new Date(now);
  switch (period) {
    case "week": start.setDate(start.getDate() - 7); break;
    case "month": start.setMonth(start.getMonth() - 1); break;
    case "year": start.setFullYear(start.getFullYear() - 1); break;
    default: start.setDate(start.getDate() - 1);
  }
  return { period, from: Math.floor(start.getTime() / 1000), to: Math.floor(now.getTime() / 1000) };
}

// Capture a fresh window on each query/refetch, and keep its boundaries with the
// returned page so a later render cannot move the drilldown beyond its data.
export async function fetchRankingPeriod<T extends object>(
  period: string,
  fetchPage: (timeframe: RankingTimeframe) => Promise<T>,
  now = new Date(),
): Promise<T & { timeframe: RankingTimeframe }> {
  const timeframe = rankingTimeframe(period, now);
  return { ...await fetchPage(timeframe), timeframe };
}

export function rankingLinkParams(args: RankingTimeframe & {
  rank_by?: string; artist_id?: number; album_id?: number;
}): URLSearchParams {
  const params = new URLSearchParams();
  for (const key of ["period", "from", "to", "rank_by", "artist_id", "album_id"] as const) {
    if (args[key] !== undefined) params.set(key, String(args[key]));
  }
  return params;
}

export function readChartCalendar(params: URLSearchParams, now = new Date()): ChartCalendar {
  const requested = params.get("period");
  const legacyPeriod = params.get("date") ? undefined
    : Number(params.get("month")) > 0 ? "month" : Number(params.get("week")) > 0 ? "week" : Number(params.get("year")) > 0 ? "year" : undefined;
  const period = (legacyPeriod ?? (["day", "week", "month", "year", "all_time"].includes(requested ?? "")
    ? requested : "day")) as ChartPeriod;
  let date = new Date(now);
  // Explicit API ranges have the same precedence here as in the backend.
  const from = Number(params.get("from"));
  const to = params.has("to") ? Number(params.get("to")) : Math.floor(now.getTime() / 1000);
  if (params.get("from") && Number.isSafeInteger(from) && Number.isSafeInteger(to) && to >= from) {
    const start = new Date(from * 1000);
    const end = new Date((to + 1) * 1000);
    if (Number.isFinite(start.getTime()) && Number.isFinite(end.getTime())) {
      return { period, date: new Date(Math.min(to * 1000, now.getTime())), range: { start, end } };
    }
  }
  const value = params.get("date");
  if (value && /^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const [year, month, day] = value.split("-").map(Number);
    const parsed = new Date(year, month - 1, day);
    if (localDateValue(parsed) === value) date = parsed;
  } else if (params.get("year") || params.get("month") || params.get("week")) {
    const month = Number(params.get("month") || 1);
    const week = Number(params.get("week") || 1);
    let year = Number(params.get("year"));
    if (!params.get("year")) {
      year = period === "week" ? isoWeek(now).year : now.getFullYear();
      if (period === "month" && month > now.getMonth() + 1) year--;
      if (period === "week" && week > isoWeek(now).week) year--;
    }
    if (Number.isInteger(year) && year >= 1000 && year <= 9999 &&
        Number.isInteger(month) && month >= 1 && month <= 12 &&
        Number.isInteger(week) && week >= 1 && week <= isoWeek(new Date(year, 11, 28)).week) {
      date = period === "week" ? isoWeekStart(year, week) : new Date(year, month - 1, 1);
    }
  }
  if (date > now) date = new Date(now);
  return { period, date };
}

// Construct both boundaries independently: a nonexistent midnight can normalize
// to 01:00, which must not be carried into the following date/month.
export function chartCalendarRange({ period, date, range }: ChartCalendar): { start: Date; end: Date } {
  if (range) return range;
  const year = date.getFullYear(), month = date.getMonth(), day = date.getDate();
  switch (period) {
    case "week": {
      const start = isoWeekStart(isoWeek(date).year, isoWeek(date).week);
      return { start, end: new Date(start.getFullYear(), start.getMonth(), start.getDate() + 7) };
    }
    case "month": return { start: new Date(year, month, 1), end: new Date(year, month + 1, 1) };
    case "year": return { start: new Date(year, 0, 1), end: new Date(year + 1, 0, 1) };
    default: return { start: new Date(year, month, day), end: new Date(year, month, day + 1) };
  }
}

export function moveChartPeriod(calendar: ChartCalendar, direction: number): Date {
  const { start } = chartCalendarRange(calendar);
  const year = start.getFullYear(), month = start.getMonth(), day = start.getDate();
  switch (calendar.period) {
    case "week": return new Date(year, month, day + direction * 7);
    case "month": return new Date(year, month + direction, 1);
    case "year": return new Date(year + direction, 0, 1);
    default: return new Date(year, month, day + direction);
  }
}

export function chartCalendarParams(params: URLSearchParams, calendar: ChartCalendar): URLSearchParams {
  const next = new URLSearchParams(params);
  for (const key of ["year", "month", "week", "from", "to"]) next.delete(key);
  next.set("period", calendar.period);
  if (calendar.period === "all_time") next.delete("date");
  else next.set("date", localDateValue(calendar.date));
  next.set("page", "1");
  return next;
}

export function chartApiParams(params: URLSearchParams, now = new Date()): URLSearchParams {
  const calendar = readChartCalendar(params, now);
  const next = chartCalendarParams(params, calendar);
  const page = Number(params.get("page"));
  next.set("page", String(Number.isSafeInteger(page) && page > 0 ? page : 1));
  next.delete("date");
  if (calendar.range || calendar.period !== "all_time") {
    const { start, end } = chartCalendarRange(calendar);
    next.set("from", String(start.getTime() / 1000));
    // SQLite uses BETWEEN on integer-second timestamps (inclusive endpoints).
    next.set("to", String(end.getTime() / 1000 - 1));
  }
  return next;
}
