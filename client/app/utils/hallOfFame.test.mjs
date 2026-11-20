import test from "node:test";
import assert from "node:assert/strict";
import { demoHallOfFame } from "./hallOfFame.ts";

const artist = (id, name) => ({ id, name, image: {} });
const song = (id, tier, plays, credits) => ({
  track_id: id,
  title: `Song ${String(id).padStart(2, "0")}`,
  plays,
  artists: credits,
  image: {},
  current: { tier, multiple: 1, plays, reached_at: "2026-01-01T00:00:00Z" },
});
const params = (value) => new URLSearchParams(value);
test("empty Hall of Fame has finite counters and arrays", () => {
  const result = demoHallOfFame([], params(""));
  assert.equal(result.certified_songs, 0);
  assert.equal(result.certified_artists, 0);
  assert.deepEqual(result.artists, []);
  assert.deepEqual(result.songs, []);
  assert.equal(result.current_page, 1);
  assert.equal(result.has_next_page, false);
});
test("collaborations count for each artist without duplicating songs or credits", () => {
  const a = artist(1, "Amber"),
    b = artist(2, "Coast");
  const songs = [
    song(1, "gold", 200, [a, a, b]),
    song(2, "platinum", 1200, [a]),
  ];
  const result = demoHallOfFame(songs, params(""));
  assert.equal(result.certified_songs, 2);
  assert.equal(result.certified_artists, 2);
  assert.deepEqual(result.counts, { gold: 1, platinum: 1, diamond: 0 });
  assert.equal(result.artists[0].certified_songs, 2);
  const filtered = demoHallOfFame(
    songs,
    params("view=songs&tier=platinum&q=AMBER"),
  );
  assert.equal(filtered.total_record_count, 1);
  assert.equal(filtered.songs[0].track_id, 2);
  assert.equal(filtered.certified_songs, 2);
  assert.equal(
    demoHallOfFame(songs, params("tier=platinum")).artists.length,
    1,
  );
});
test("stable sorting, pagination, clamping and empty search", () => {
  const songs = Array.from({ length: 23 }, (_, i) =>
    song(i + 1, "gold", 200, [artist(1, "Artist")]),
  );
  const result = demoHallOfFame(
    songs,
    params("view=songs&page=2&sort=name&order=asc"),
  );
  assert.equal(result.total_record_count, 23);
  assert.equal(result.songs.length, 3);
  assert.equal(result.songs[0].track_id, 21);
  assert.equal(result.has_next_page, false);
  assert.equal(
    demoHallOfFame(songs, params("view=songs&page=999")).current_page,
    2,
  );
  assert.equal(
    demoHallOfFame(songs, params("view=songs&order=desc")).songs[0].track_id,
    1,
  );
  assert.equal(
    demoHallOfFame(songs, params("q=missing")).total_record_count,
    0,
  );
});
