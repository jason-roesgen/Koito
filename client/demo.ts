import type { Plugin } from "vite";
import type { Album, Artist, Track, ImageList } from "./api/api";

// In-memory fixtures used only by the explicitly enabled Vite development server.
export function demoDataPlugin(): Plugin {
  const now = new Date();
  const day = 86400000;
  const names = ["Neon Orchard", "Paper Satellites", "Juniper Coast", "Velvet Transit", "Lunar Arcade", "Amber Current", "Quiet Atlas", "Glass Meadow"];
  const titles = ["Afterglow", "Soft Signals", "Tidal Patterns", "Night Windows", "Parallel Skies", "Golden Static", "Slow Maps", "Morning Geometry"];
  const artwork = (i: number): ImageList => {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="300"><rect width="300" height="300" fill="hsl(${i * 43},45%,23%)"/><circle cx="${80 + i * 12}" cy="120" r="85" fill="hsl(${i * 43 + 45},65%,65%)"/><path d="M0 230L300 120V300H0Z" fill="hsl(${i * 43 + 90},50%,40%)"/></svg>`;
    const url = `data:image/svg+xml,${encodeURIComponent(svg)}`;
    return { xs: url, small: url, medium: url, large: url, xl: url };
  };
  const base = { listen_count: 0, time_listened: 0, first_listen: 0, all_time_rank: 0, musicbrainz_id: "" };
  const artists: Artist[] = names.map((name, i) => ({ ...base, id: i + 1, name, image: artwork(i), aliases: [], is_primary: true }));
  const albums: Album[] = titles.map((title, i) => ({ ...base, id: i + 1, title, image: artwork(i), artists: [{ id: i + 1, name: names[i] }], is_various_artists: false }));
  const tracks: Track[] = albums.flatMap((album, i) => ["First Light", "Side Streets", "Drift", "Open Water", "Homeward", "Last Train"].map((name, j) => ({ ...base, id: i * 6 + j + 1, title: `${name} (${album.title})`, image: album.image, artists: album.artists, album_id: album.id })));
  const duration = (track: Track) => 150 + (track.id * 37) % 210;
  const history: { time: string; track: Track }[] = [];
  for (let d = 450; d >= 0; d--) {
    if (d > 7 && d % 13 === 0) continue;
    const date = new Date(now);
    date.setDate(date.getDate() - d);
    const count = 12 + (d * 17) % 57;
    for (let j = 0; j < count; j++) {
      const at = new Date(date);
      at.setHours(7 + Math.floor(j / count * 16), (j * 7) % 60, 0, 0);
      if (at > now) continue;
      history.push({ time: at.toISOString(), track: tracks[(d * 7 + j * j + j) % tracks.length] });
    }
  }
  history.sort((a, b) => b.time.localeCompare(a.time));
  type Listen = typeof history[number];
  const matches = (l: Listen, p: URLSearchParams) =>
    (!Number(p.get("artist_id")) || l.track.artists[0].id === Number(p.get("artist_id"))) &&
    (!Number(p.get("album_id")) || l.track.album_id === Number(p.get("album_id"))) &&
    (!Number(p.get("track_id")) || l.track.id === Number(p.get("track_id")));
  function select(p: URLSearchParams) {
    let start = 0, end = now.getTime() + 1;
    const period = p.get("period");
    const days = { day: 1, week: 7, month: 30, year: 365 }[period ?? ""];
    if (days) start = now.getTime() - days * day;
    const year = Number(p.get("year")), month = Number(p.get("month"));
    if (year) {
      start = new Date(year, month ? month - 1 : 0, 1).getTime();
      end = new Date(year + (month ? 0 : 1), month || 0, 1).getTime();
    }
    const week = Number(p.get("week"));
    if (year && week) {
      const first = new Date(year, 0, 4);
      first.setDate(first.getDate() - (first.getDay() + 6) % 7 + (week - 1) * 7);
      start = first.getTime();
      const last = new Date(first); last.setDate(last.getDate() + 7);
      end = last.getTime();
    }
    if (Number(p.get("from"))) start = Number(p.get("from")) * 1000;
    if (Number(p.get("to"))) end = Number(p.get("to")) * 1000;
    return history.filter(l => matches(l, p) && Date.parse(l.time) >= start && Date.parse(l.time) < end);
  }
  const entityId = (l: Listen, kind: string) => kind === "artists" ? l.track.artists[0].id : kind === "albums" ? l.track.album_id : l.track.id;
  function ranked(kind: string, rows: Listen[], byTime = false) {
    const entities = kind === "artists" ? artists : kind === "albums" ? albums : tracks;
    return entities.map(entity => {
      const plays = rows.filter(l => entityId(l, kind) === entity.id);
      return { item: { ...entity, listen_count: plays.length, time_listened: plays.reduce((sum, l) => sum + duration(l.track), 0), first_listen: plays.length ? Date.parse(plays[plays.length - 1].time) / 1000 : 0 }, rank: 0 };
    }).filter(r => r.item.listen_count > 0).sort((a, b) => (byTime ? b.item.time_listened - a.item.time_listened : b.item.listen_count - a.item.listen_count) || a.item.id - b.item.id).map((r, i) => ({ ...r, rank: i + 1, item: { ...r.item, all_time_rank: i + 1 } }));
  }
  function page<T>(items: T[], p: URLSearchParams) {
    const limit = Math.max(1, Math.min(100, Number(p.get("limit")) || 20));
    const currentPage = Math.max(1, Number(p.get("page")) || 1);
    const index = currentPage - 1;
    return { items: items.slice(index * limit, (index + 1) * limit), total_record_count: items.length, has_next_page: (index + 1) * limit < items.length, current_page: currentPage, items_per_page: limit };
  }
  const dateKey = (time: string | Date) => new Date(time).toLocaleDateString("en-CA");
  function stats(rows: Listen[]) {
    const dates = new Set(rows.map(l => dateKey(l.time)));
    let longest = 0, streak = 0;
    for (let d = 450; d >= 0; d--) {
      const at = new Date(now); at.setDate(at.getDate() - d);
      streak = dates.has(dateKey(at)) ? streak + 1 : 0;
      longest = Math.max(longest, streak);
    }
    const a = ranked("artists", rows).length, b = ranked("albums", rows).length, t = ranked("tracks", rows).length;
    return { listen_count: rows.length, track_count: t, album_count: b, artist_count: a, minutes_listened: Math.floor(rows.reduce((sum, l) => sum + duration(l.track), 0) / 60), days_active: dates.size, avg_daily_plays: rows.length / (dates.size || 1), tracks_per_artist: t / (a || 1), albums_per_artist: b / (a || 1), longest_streak: longest };
  }
  return {
    name: "koito-synthetic-history", apply: "serve",
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const url = new URL(req.url ?? "/", "http://localhost");
        if (!url.pathname.startsWith("/apis/web/v1/")) return next();
        const send = (body: unknown, status = 200) => { res.statusCode = status; res.setHeader("Content-Type", "application/json"); res.setHeader("Cache-Control", "no-store"); res.end(JSON.stringify(body)); };
        if (req.method !== "GET") return send({ error: "Synthetic demo is read-only. Disable demo mode to edit local data." }, 403);
        const path = url.pathname.slice("/apis/web/v1/".length), p = url.searchParams;
        const rows = select(p);
        if (path === "health") return send({ demo: true });
        if (path === "config") return send({ default_theme: "yuu" });
        if (path === "user") return send({ error: "Read-only demo" }, 401);
        if (path === "first-activity") return send({ time: history[history.length - 1].time });
        if (path === "now-playing") return send({ currently_playing: false });
        if (path === "listens") return send(page(rows, p));
        if (path === "stats") return send(stats(rows));
        if (path.startsWith("top/")) return send(page(ranked(path.slice(4), rows, p.get("rank_by") === "time"), p));
        if (path === "listen-activity") {
          const counts = new Map<string, number>();
          rows.forEach(l => counts.set(dateKey(l.time), (counts.get(dateKey(l.time)) ?? 0) + 1));
          const activity = Array.from({ length: Math.min(451, Number(p.get("range")) || 252) }, (_, i) => {
            const at = new Date(now); at.setHours(0, 0, 0, 0); at.setDate(at.getDate() - i);
            return { start_time: at.toISOString(), listens: counts.get(dateKey(at)) ?? 0 };
          }).reverse();
          let streak = 0;
          for (const item of [...activity].reverse()) { if (!item.listens) break; streak++; }
          return send({ activity, streak });
        }
        const detail = path.match(/^(artist|album|track)\/(\d+)(?:\/(interest|artists|aliases))?$/);
        if (detail) {
          const [, kind, id, suffix] = detail;
          const filtered = history.filter(l => entityId(l, `${kind}s`) === Number(id));
          const item = ranked(`${kind}s`, history).find(r => r.item.id === Number(id))?.item;
          if (!item) return send({ error: "Demo item not found" }, 404);
          if (suffix === "aliases") return send([]);
          if (suffix === "artists") return send(artists.filter(a => filtered.some(l => l.track.artists[0].id === a.id)));
          if (suffix === "interest") {
            const count = Math.max(1, Math.min(100, Number(p.get("buckets")) || 16));
            return send(Array.from({ length: count }, (_, i) => {
              const start = now.getTime() - 451 * day + i * 451 * day / count, end = start + 451 * day / count;
              return { bucket_start: new Date(start), bucket_end: new Date(end), listen_count: filtered.filter(l => Date.parse(l.time) >= start && Date.parse(l.time) < end).length };
            }));
          }
          return send(item);
        }
        if (path === "search") {
          const q = (p.get("q") ?? "").toLowerCase();
          return send(Object.fromEntries(["artists", "albums", "tracks"].map(kind => [kind, ranked(kind, history).map(r => r.item).filter(item => ("name" in item ? item.name : item.title).toLowerCase().includes(q))])));
        }
        if (path === "summary") {
          const s = stats(rows);
          const prior = history.filter(l => rows.length && l.time < rows[rows.length - 1].time);
          const discovered = (kind: string) => ranked(kind, rows).filter(r => !prior.some(l => entityId(l, kind) === r.item.id)).length;
          return send({ title: "Synthetic listening rewind", top_artists: ranked("artists", rows, p.get("rank_by") === "time").slice(0, 5), top_albums: ranked("albums", rows, p.get("rank_by") === "time").slice(0, 5), top_tracks: ranked("tracks", rows, p.get("rank_by") === "time").slice(0, 5), minutes_listened: s.minutes_listened, avg_minutes_listened_per_day: s.minutes_listened / (s.days_active || 1), plays: rows.length, avg_plays_per_day: s.avg_daily_plays, unique_tracks: s.track_count, unique_albums: s.album_count, unique_artists: s.artist_count, new_tracks: discovered("tracks"), new_albums: discovered("albums"), new_artists: discovered("artists") });
        }
        return send({ error: "This endpoint is unavailable in synthetic demo mode." }, 404);
      });
    },
  };
}
