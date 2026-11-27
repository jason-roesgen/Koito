import assert from "node:assert/strict";
import test from "node:test";
import { repeatSessionDateBounds } from "./chartCalendar.ts";

test("inclusive local date filters use a half-open range", () => {
  const { start, end } = repeatSessionDateBounds("2024-02-29", "2024-02-29");
  assert.equal(start, new Date(2024, 1, 29).getTime() / 1000);
  assert.equal(end, new Date(2024, 2, 1).getTime() / 1000);
  assert.equal(repeatSessionDateBounds("", "2024-02-29").start, undefined);
  assert.equal(repeatSessionDateBounds("2024-02-29", "").end, undefined);
});

test("spring and autumn boundaries follow the local calendar", () => {
  for (const [value, month, day] of [["2024-03-31", 2, 31], ["2024-10-27", 9, 27]]) {
    const { start, end } = repeatSessionDateBounds(value, value);
    const localDayLength = (new Date(2024, month, day + 1) - new Date(2024, month, day)) / 1000;
    assert.equal(end - start, localDayLength);
    if (process.env.TZ === "Europe/Berlin") {
      assert.equal(end - start, (month === 2 ? 23 : 25) * 3600);
    }
  }
});

test("invalid and reversed date ranges are rejected", () => {
  assert.throws(() => repeatSessionDateBounds("2023-02-29", ""), RangeError);
  assert.throws(() => repeatSessionDateBounds("2024-03-31", "2024-03-30"), RangeError);
  assert.throws(() => repeatSessionDateBounds("not a date", ""), RangeError);
});
