import assert from "node:assert/strict";
import { test } from "node:test";
import {
  isoWeek, isoWeekStart, localDateValue, moveChartPeriod, readChartCalendar,
  chartApiParams, chartCalendarParams, chartCalendarRange, rankingTimeframe, rankingLinkParams, fetchRankingPeriod,
} from "./chartCalendar.ts";

const now = new Date(2026, 10, 9, 21, 0);
const params = (query) => new URLSearchParams(query);
const selected = (period, date) => readChartCalendar(params({ period, date }), now);

test("day selection uses local midnight and inclusive last second", () => {
  const query = chartApiParams(params("period=day&date=2024-02-29"), now);
  assert.equal(Number(query.get("from")), new Date(2024, 1, 29).getTime() / 1000);
  assert.equal(Number(query.get("to")), new Date(2024, 2, 1).getTime() / 1000 - 1);
  assert.equal(query.get("page"), "1");
  assert.equal(query.has("date"), false);
});

test("calendar days and weeks span DST rather than fixed 24-hour intervals", () => {
  for (const [day, hours] of [["2026-03-29", 23], ["2026-10-25", 25]]) {
    const { start, end } = chartCalendarRange(selected("day", day));
    assert.equal(start.getHours(), 0);
    assert.equal(end.getHours(), 0);
    if (process.env.TZ === "Europe/Berlin") assert.equal((end - start) / 3600000, hours);
    assert.equal(localDateValue(moveChartPeriod(selected("day", day), 1)), day === "2026-03-29" ? "2026-03-30" : "2026-10-26");
  }
  if (process.env.TZ === "America/New_York") {
    for (const [day, hours] of [["2026-03-08", 23], ["2026-11-01", 25]]) {
      const range = chartCalendarRange(selected("day", day));
      assert.equal((range.end - range.start) / 3600000, hours);
    }
  }
  const { start, end } = chartCalendarRange(selected("week", "2026-03-29"));
  assert.equal(localDateValue(start), "2026-03-23");
  assert.equal(localDateValue(end), "2026-03-30");
  if (process.env.TZ === "Europe/Berlin") assert.equal((end - start) / 3600000, 167);
});

test("ISO weeks cross calendar years with the correct ISO year", () => {
  assert.deepEqual(isoWeek(new Date(2021, 0, 1)), { year: 2020, week: 53 });
  assert.deepEqual(isoWeek(new Date(2024, 11, 30)), { year: 2025, week: 1 });
  assert.equal(localDateValue(isoWeekStart(2020, 53)), "2020-12-28");
  const query = chartApiParams(params("year=2020&week=53"), now);
  assert.equal(query.get("from"), String(new Date(2020, 11, 28).getTime() / 1000));
  assert.equal(query.get("to"), String(new Date(2021, 0, 4).getTime() / 1000 - 1));
});

test("month and year navigation handles leap days and short months", () => {
  assert.equal(localDateValue(moveChartPeriod(selected("month", "2024-03-31"), -1)), "2024-02-01");
  assert.equal(localDateValue(moveChartPeriod(selected("month", "2024-01-31"), -1)), "2023-12-01");
  assert.equal(localDateValue(moveChartPeriod(selected("year", "2024-02-29"), 1)), "2025-01-01");
  assert.equal(localDateValue(moveChartPeriod(selected("week", "2021-01-01"), 1)), "2021-01-04");
  const { start, end } = chartCalendarRange(selected("year", "2024-06-15"));
  assert.equal(localDateValue(start), "2024-01-01");
  assert.equal(localDateValue(end), "2025-01-01");
});

test("all periods produce API boundaries matching the displayed calendar range", () => {
  for (const period of ["day", "week", "month", "year"]) {
    for (const date of ["2024-02-29", "2026-03-29", "2026-10-25", "2021-01-01"]) {
      const calendar = selected(period, date);
      const { start, end } = chartCalendarRange(calendar);
      const query = chartApiParams(params({ period, date }), now);
      assert.equal(Number(query.get("from")), start.getTime() / 1000);
      assert.equal(Number(query.get("to")), end.getTime() / 1000 - 1);
      assert.ok(Number(query.get("to")) < end.getTime() / 1000);
    }
  }
});

test("period changes reset pagination, remove conflicting dates and retain filters", () => {
  const original = params("period=month&year=2024&month=2&week=3&from=1&to=2&page=4&limit=100&rank_by=time&artist_id=7&album_id=9&custom=x");
  const before = original.toString();
  const next = chartCalendarParams(original, selected("day", "2024-02-29"));
  assert.equal(next.get("page"), "1");
  assert.equal(next.get("date"), "2024-02-29");
  for (const key of ["year", "month", "week", "from", "to"]) assert.equal(next.has(key), false);
  for (const key of ["limit", "rank_by", "artist_id", "album_id", "custom"]) assert.equal(next.get(key), original.get(key));
  assert.equal(original.toString(), before);
});

test("pagination and URL history preserve the selected period and ranking", () => {
  const first = chartCalendarParams(params("rank_by=time&artist_id=7&limit=100"), selected("day", "2024-02-29"));
  const history = [first.toString()];
  first.set("page", "2");
  history.push(first.toString());
  history.push(chartCalendarParams(first, selected("month", "2024-01-01")).toString());
  for (const query of history.slice(0, 2)) {
    assert.equal(localDateValue(readChartCalendar(params(query), now).date), "2024-02-29");
    assert.equal(chartApiParams(params(query), now).get("rank_by"), "time");
  }
  assert.equal(chartApiParams(params(history[1]), now).get("page"), "2");
  assert.equal(chartApiParams(params(history[2]), now).get("page"), "1");
});

test("all time removes date restrictions and preserves unrelated filters", () => {
  const selected = chartCalendarParams(params("date=2024-02-29&year=2024&from=1&to=2&rank_by=time"), { period: "all_time", date: now });
  selected.set("page", "3");
  const query = chartApiParams(selected, now);
  for (const key of ["date", "year", "month", "week", "from", "to"]) assert.equal(query.has(key), false);
  assert.equal(query.get("period"), "all_time");
  assert.equal(query.get("rank_by"), "time");
  assert.equal(query.get("page"), "3");
});

test("legacy year/month/week links select calendar periods", () => {
  assert.equal(readChartCalendar(params("year=2024"), now).period, "year");
  assert.equal(readChartCalendar(params("year=2024&month=2"), now).period, "month");
  assert.equal(readChartCalendar(params("year=2020&week=53"), now).period, "week");
  assert.equal(localDateValue(readChartCalendar(params("year=2020&week=53"), now).date), "2020-12-28");
});

test("malformed or future dates fall back safely and invalid pages use page one", () => {
  for (const date of ["2024-02-30", "garbage", "2027-01-01"]) {
    assert.equal(localDateValue(readChartCalendar(params({ period: "day", date }), now).date), "2026-11-09");
  }
  for (const page of ["0", "-1", "NaN", "1.5", "Infinity", "9007199254740992"]) {
    assert.equal(chartApiParams(params({ page }), now).get("page"), "1");
  }
});


test("dashboard query and drilldown preserve the same rolling window", () => {
  for (const period of ["day", "week", "month", "year", "all_time"]) {
    const source = rankingTimeframe(period, now);
    const link = rankingLinkParams({ ...source, rank_by: "time", artist_id: 7, album_id: 9 });
    const destination = chartApiParams(link, new Date(now.getTime() + 60000));
    for (const key of ["from", "to"]) {
      assert.equal(destination.get(key), source[key] === undefined ? null : String(source[key]));
    }
    assert.equal(destination.get("rank_by"), "time");
    assert.equal(destination.get("artist_id"), "7");
    assert.equal(destination.get("album_id"), "9");
    assert.equal(Boolean(readChartCalendar(link, now).range), period !== "all_time");
  }
  const source = rankingTimeframe("week", now);
  assert.equal(source.from, new Date(2026, 10, 2, 21).getTime() / 1000);
  assert.equal(source.to, now.getTime() / 1000);
});

test("explicit legacy ranges survive pagination and retain precedence", () => {
  const query = params("period=all_time&from=1700000000&to=1700600000&page=4&rank_by=time");
  for (const page of ["1", "2", "4"]) {
    query.set("page", page);
    const api = chartApiParams(query, now);
    assert.equal(api.get("from"), "1700000000");
    assert.equal(api.get("to"), "1700600000");
    assert.equal(api.get("page"), page);
  }
});

test("switching an exact dashboard range to Week explicitly selects an ISO week", () => {
  const source = rankingLinkParams({ ...rankingTimeframe("week", now), rank_by: "time" });
  const range = readChartCalendar(source, now);
  const selectedWeek = chartCalendarParams(source, { period: "week", date: range.date });
  assert.equal(selectedWeek.has("from"), false);
  assert.equal(selectedWeek.has("to"), false);
  const api = chartApiParams(selectedWeek, now);
  assert.equal(api.get("from"), String(new Date(2026, 10, 9).getTime() / 1000));
  assert.equal(api.get("to"), String(new Date(2026, 10, 16).getTime() / 1000 - 1));
});

test("changing calendar tabs preserves the selected historical anchor", () => {
  const selectedDay = selected("day", "2021-01-01");
  const week = readChartCalendar(chartCalendarParams(params("rank_by=time"), { period: "week", date: selectedDay.date }), now);
  assert.deepEqual(isoWeek(week.date), { year: 2020, week: 53 });
  const range = chartCalendarRange(week);
  assert.equal(localDateValue(range.start), "2020-12-28");
  assert.equal(localDateValue(range.end), "2021-01-04");
});

test("legacy month and ISO week work without an explicit year", () => {
  assert.equal(localDateValue(readChartCalendar(params("month=2"), now).date), "2026-02-01");
  assert.equal(localDateValue(readChartCalendar(params("month=12"), now).date), "2025-12-01");
  assert.equal(localDateValue(readChartCalendar(params("week=3"), now).date), "2026-01-12");
  assert.equal(localDateValue(readChartCalendar(params("week=50"), now).date), "2025-12-08");
  assert.deepEqual(isoWeek(readChartCalendar(params("week=53"), new Date(2021, 0, 1)).date), { year: 2020, week: 53 });
  assert.equal(readChartCalendar(params("year=2024&month=&week="), now).period, "year");
  assert.equal(readChartCalendar(params("period=week&year=2024&month=2"), now).period, "month");
});

test("midnight DST skips never carry 01:00 into the next boundary", () => {
  if (process.env.TZ !== "America/Santiago") return;
  const day = selected("day", "2026-09-06");
  const range = chartCalendarRange(day);
  assert.equal(range.start.getHours(), 1); // Midnight does not exist on this date.
  assert.equal(range.end.getTime(), new Date(2026, 8, 7).getTime());
  const api = chartApiParams(params("period=day&date=2026-09-06"), now);
  assert.equal(Number(api.get("to")), 1788749999);
  assert.ok(new Date(2026, 8, 7, 0, 30).getTime() / 1000 > Number(api.get("to")));
  assert.equal(moveChartPeriod(day, 1).getHours(), 0);
  const month = chartCalendarRange(selected("month", "2019-09-08"));
  assert.equal(month.start.getTime(), new Date(2019, 8, 1).getTime());
  assert.equal(month.end.getTime(), new Date(2019, 9, 1).getTime());
});


test("refetches capture a fresh rolling window and keep each response's own range", async () => {
  const requested = [];
  const request = async timeframe => { requested.push(timeframe); return { items: [] }; };
  const first = await fetchRankingPeriod("week", request, now);
  const later = new Date(now.getTime() + 60000);
  const second = await fetchRankingPeriod("week", request, later);
  assert.equal(second.timeframe.to - first.timeframe.to, 60);
  assert.equal(first.timeframe.to, Math.floor(now.getTime() / 1000));
  for (const page of [first, second]) {
    const destination = chartApiParams(rankingLinkParams(page.timeframe), later);
    assert.equal(destination.get("from"), String(page.timeframe.from));
    assert.equal(destination.get("to"), String(page.timeframe.to));
  }
  assert.equal(requested.length, 2);
});
