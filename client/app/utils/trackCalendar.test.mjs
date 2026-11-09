import assert from "node:assert/strict";
import { test } from "node:test";
import {
  isoWeek, isoWeekStart, localDateValue, moveTrackPeriod, readTrackCalendar,
  trackApiParams, trackCalendarParams, trackCalendarRange,
} from "./trackCalendar.ts";

const now = new Date(2026, 10, 9, 21, 0);
const params = (query) => new URLSearchParams(query);
const selected = (period, date) => readTrackCalendar(params({ period, date }), now);

test("day selection uses local midnight and inclusive last second", () => {
  const query = trackApiParams(params("period=day&date=2024-02-29"), now);
  assert.equal(Number(query.get("from")), new Date(2024, 1, 29).getTime() / 1000);
  assert.equal(Number(query.get("to")), new Date(2024, 2, 1).getTime() / 1000 - 1);
  assert.equal(query.get("page"), "1");
  assert.equal(query.has("date"), false);
});

test("calendar days and weeks span DST rather than fixed 24-hour intervals", () => {
  for (const [day, hours] of [["2026-03-29", 23], ["2026-10-25", 25]]) {
    const { start, end } = trackCalendarRange(selected("day", day));
    assert.equal(start.getHours(), 0);
    assert.equal(end.getHours(), 0);
    if (process.env.TZ === "Europe/Berlin") assert.equal((end - start) / 3600000, hours);
    assert.equal(localDateValue(moveTrackPeriod(selected("day", day), 1)), day === "2026-03-29" ? "2026-03-30" : "2026-10-26");
  }
  if (process.env.TZ === "America/New_York") {
    for (const [day, hours] of [["2026-03-08", 23], ["2026-11-01", 25]]) {
      const range = trackCalendarRange(selected("day", day));
      assert.equal((range.end - range.start) / 3600000, hours);
    }
  }
  const { start, end } = trackCalendarRange(selected("week", "2026-03-29"));
  assert.equal(localDateValue(start), "2026-03-23");
  assert.equal(localDateValue(end), "2026-03-30");
  if (process.env.TZ === "Europe/Berlin") assert.equal((end - start) / 3600000, 167);
});

test("ISO weeks cross calendar years with the correct ISO year", () => {
  assert.deepEqual(isoWeek(new Date(2021, 0, 1)), { year: 2020, week: 53 });
  assert.deepEqual(isoWeek(new Date(2024, 11, 30)), { year: 2025, week: 1 });
  assert.equal(localDateValue(isoWeekStart(2020, 53)), "2020-12-28");
  const query = trackApiParams(params("year=2020&week=53"), now);
  assert.equal(query.get("from"), String(new Date(2020, 11, 28).getTime() / 1000));
  assert.equal(query.get("to"), String(new Date(2021, 0, 4).getTime() / 1000 - 1));
});

test("month and year navigation handles leap days and short months", () => {
  assert.equal(localDateValue(moveTrackPeriod(selected("month", "2024-03-31"), -1)), "2024-02-01");
  assert.equal(localDateValue(moveTrackPeriod(selected("month", "2024-01-31"), -1)), "2023-12-01");
  assert.equal(localDateValue(moveTrackPeriod(selected("year", "2024-02-29"), 1)), "2025-01-01");
  assert.equal(localDateValue(moveTrackPeriod(selected("week", "2021-01-01"), 1)), "2021-01-04");
  const { start, end } = trackCalendarRange(selected("year", "2024-06-15"));
  assert.equal(localDateValue(start), "2024-01-01");
  assert.equal(localDateValue(end), "2025-01-01");
});

test("all periods produce API boundaries matching the displayed calendar range", () => {
  for (const period of ["day", "week", "month", "year"]) {
    for (const date of ["2024-02-29", "2026-03-29", "2026-10-25", "2021-01-01"]) {
      const calendar = selected(period, date);
      const { start, end } = trackCalendarRange(calendar);
      const query = trackApiParams(params({ period, date }), now);
      assert.equal(Number(query.get("from")), start.getTime() / 1000);
      assert.equal(Number(query.get("to")), end.getTime() / 1000 - 1);
      assert.ok(Number(query.get("to")) < end.getTime() / 1000);
    }
  }
});

test("period changes reset pagination, remove conflicting dates and retain filters", () => {
  const original = params("period=month&year=2024&month=2&week=3&from=1&to=2&page=4&limit=100&rank_by=time&artist_id=7&album_id=9&custom=x");
  const before = original.toString();
  const next = trackCalendarParams(original, selected("day", "2024-02-29"));
  assert.equal(next.get("page"), "1");
  assert.equal(next.get("date"), "2024-02-29");
  for (const key of ["year", "month", "week", "from", "to"]) assert.equal(next.has(key), false);
  for (const key of ["limit", "rank_by", "artist_id", "album_id", "custom"]) assert.equal(next.get(key), original.get(key));
  assert.equal(original.toString(), before);
});

test("pagination and URL history preserve the selected period and ranking", () => {
  const first = trackCalendarParams(params("rank_by=time&artist_id=7&limit=100"), selected("day", "2024-02-29"));
  const history = [first.toString()];
  first.set("page", "2");
  history.push(first.toString());
  history.push(trackCalendarParams(first, selected("month", "2024-01-01")).toString());
  for (const query of history.slice(0, 2)) {
    assert.equal(localDateValue(readTrackCalendar(params(query), now).date), "2024-02-29");
    assert.equal(trackApiParams(params(query), now).get("rank_by"), "time");
  }
  assert.equal(trackApiParams(params(history[1]), now).get("page"), "2");
  assert.equal(trackApiParams(params(history[2]), now).get("page"), "1");
});

test("all time removes date restrictions and preserves unrelated filters", () => {
  const query = trackApiParams(params("period=all_time&date=2024-02-29&year=2024&from=1&to=2&rank_by=time&page=3"), now);
  for (const key of ["date", "year", "month", "week", "from", "to"]) assert.equal(query.has(key), false);
  assert.equal(query.get("period"), "all_time");
  assert.equal(query.get("rank_by"), "time");
  assert.equal(query.get("page"), "3");
});

test("legacy year/month/week links select calendar periods", () => {
  assert.equal(readTrackCalendar(params("year=2024"), now).period, "year");
  assert.equal(readTrackCalendar(params("year=2024&month=2"), now).period, "month");
  assert.equal(readTrackCalendar(params("year=2020&week=53"), now).period, "week");
  assert.equal(localDateValue(readTrackCalendar(params("year=2020&week=53"), now).date), "2020-12-28");
});

test("malformed or future dates fall back safely and invalid pages use page one", () => {
  for (const date of ["2024-02-30", "garbage", "2027-01-01"]) {
    assert.equal(localDateValue(readTrackCalendar(params({ period: "day", date }), now).date), "2026-11-09");
  }
  for (const page of ["0", "-1", "NaN", "1.5", "Infinity", "9007199254740992"]) {
    assert.equal(trackApiParams(params({ page }), now).get("page"), "1");
  }
});
