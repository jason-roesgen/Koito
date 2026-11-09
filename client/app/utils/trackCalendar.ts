export type TrackPeriod = "day" | "week" | "month" | "year" | "all_time";
export type TrackCalendar = { period: TrackPeriod; date: Date };

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

export function readTrackCalendar(params: URLSearchParams, now = new Date()): TrackCalendar {
  const requested = params.get("period");
  const period = (["day", "week", "month", "year", "all_time"].includes(requested ?? "")
    ? requested
    : params.has("week") ? "week" : params.has("month") ? "month" : params.has("year") ? "year" : "day") as TrackPeriod;
  let date = new Date(now);
  const value = params.get("date");
  if (value && /^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const [year, month, day] = value.split("-").map(Number);
    const parsed = new Date(year, month - 1, day);
    if (localDateValue(parsed) === value) date = parsed;
  } else if (params.has("year")) {
    const year = Number(params.get("year"));
    const month = Number(params.get("month") || 1);
    const week = Number(params.get("week") || 1);
    if (Number.isInteger(year) && year >= 1000 && year <= 9999 &&
        Number.isInteger(month) && month >= 1 && month <= 12 &&
        Number.isInteger(week) && week >= 1 && week <= isoWeek(new Date(year, 11, 28)).week) {
      date = period === "week" ? isoWeekStart(year, week) : new Date(year, month - 1, 1);
    }
  } else if (Number(params.get("from")) > 0) {
    const parsed = new Date(Number(params.get("from")) * 1000);
    if (Number.isFinite(parsed.getTime())) date = parsed;
  }
  if (date > now) date = new Date(now);
  return { period, date };
}

// Construct calendar boundaries in local time. Calendar arithmetic, rather than
// fixed millisecond offsets, keeps days and weeks correct across DST changes.
export function trackCalendarRange({ period, date }: TrackCalendar): { start: Date; end: Date } {
  let start = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const end = new Date(start);
  switch (period) {
    case "week":
      start = isoWeekStart(isoWeek(date).year, isoWeek(date).week);
      end.setTime(start.getTime());
      end.setDate(end.getDate() + 7);
      break;
    case "month":
      start.setDate(1);
      end.setDate(1);
      end.setMonth(end.getMonth() + 1);
      break;
    case "year":
      start = new Date(date.getFullYear(), 0, 1);
      end.setTime(new Date(date.getFullYear() + 1, 0, 1).getTime());
      break;
    default:
      end.setDate(end.getDate() + 1);
  }
  return { start, end }; // end is exclusive
}

export function moveTrackPeriod(calendar: TrackCalendar, direction: number): Date {
  const { start } = trackCalendarRange(calendar);
  switch (calendar.period) {
    case "day": start.setDate(start.getDate() + direction); break;
    case "week": start.setDate(start.getDate() + direction * 7); break;
    case "month": start.setMonth(start.getMonth() + direction); break;
    case "year": start.setFullYear(start.getFullYear() + direction); break;
  }
  return start;
}

export function trackCalendarParams(params: URLSearchParams, calendar: TrackCalendar): URLSearchParams {
  const next = new URLSearchParams(params);
  for (const key of ["year", "month", "week", "from", "to"]) next.delete(key);
  next.set("period", calendar.period);
  if (calendar.period === "all_time") next.delete("date");
  else next.set("date", localDateValue(calendar.date));
  next.set("page", "1");
  return next;
}

export function trackApiParams(params: URLSearchParams, now = new Date()): URLSearchParams {
  const calendar = readTrackCalendar(params, now);
  const next = trackCalendarParams(params, calendar);
  const page = Number(params.get("page"));
  next.set("page", String(Number.isSafeInteger(page) && page > 0 ? page : 1));
  next.delete("date");
  if (calendar.period !== "all_time") {
    const { start, end } = trackCalendarRange(calendar);
    next.set("from", String(start.getTime() / 1000));
    // SQLite uses BETWEEN on integer-second timestamps (inclusive endpoints).
    next.set("to", String(end.getTime() / 1000 - 1));
  }
  return next;
}
