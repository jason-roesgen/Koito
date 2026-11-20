import type {
  HallOfFame,
  HallSong,
  HallArtist,
} from "../../api/certifications";

// Synthetic demo counterpart of the server projection. Never used for real data.
export function demoHallOfFame(
  songs: HallSong[],
  params: URLSearchParams,
): HallOfFame {
  const counts = { gold: 0, platinum: 0, diamond: 0 };
  const artists = new Map<number, HallArtist>();
  for (const song of songs) {
    counts[song.current.tier]++;
    for (const credit of new Map(song.artists.map((a) => [a.id, a])).values()) {
      const artist = artists.get(credit.id) ?? {
        ...credit,
        counts: { gold: 0, platinum: 0, diamond: 0 },
        certified_songs: 0,
        reached_at: song.current.reached_at,
      };
      artist.certified_songs++;
      artist.counts[song.current.tier]++;
      if (artist.reached_at < song.current.reached_at)
        artist.reached_at = song.current.reached_at;
      artists.set(artist.id, artist);
    }
  }
  const view = params.get("view") || "artists";
  const tier = params.get("tier") as keyof typeof counts | null;
  const q = (params.get("q") || "").trim().toLowerCase();
  const sort = params.get("sort") || "count",
    order = params.get("order") || "desc";
  let selectedSongs = songs.filter(
    (s) =>
      (!tier || s.current.tier === tier) &&
      [s.title, ...s.artists.map((a) => a.name)].some((name) =>
        name.toLowerCase().includes(q),
      ),
  );
  let selectedArtists = [...artists.values()].filter(
    (a) => (!tier || a.counts[tier] > 0) && a.name.toLowerCase().includes(q),
  );
  const compare = (
    a: string | number,
    b: string | number,
    aid: number,
    bid: number,
  ) => {
    const cmp = a < b ? -1 : a > b ? 1 : 0;
    return cmp === 0 ? aid - bid : order === "asc" ? cmp : -cmp;
  };
  selectedSongs.sort((a, b) =>
    compare(
      sort === "name"
        ? a.title.toLowerCase()
        : sort === "date"
          ? a.current.reached_at
          : a.plays,
      sort === "name"
        ? b.title.toLowerCase()
        : sort === "date"
          ? b.current.reached_at
          : b.plays,
      a.track_id,
      b.track_id,
    ),
  );
  selectedArtists.sort((a, b) =>
    compare(
      sort === "name"
        ? a.name.toLowerCase()
        : sort === "date"
          ? a.reached_at
          : tier
            ? a.counts[tier]
            : a.certified_songs,
      sort === "name"
        ? b.name.toLowerCase()
        : sort === "date"
          ? b.reached_at
          : tier
            ? b.counts[tier]
            : b.certified_songs,
      a.id,
      b.id,
    ),
  );
  const total =
    view === "songs" ? selectedSongs.length : selectedArtists.length;
  const page = Math.min(
    Math.max(1, Number(params.get("page")) || 1),
    Math.max(1, Math.ceil(total / 20)),
  );
  const offset = (page - 1) * 20;
  selectedSongs =
    view === "songs" ? selectedSongs.slice(offset, offset + 20) : [];
  selectedArtists =
    view === "artists" ? selectedArtists.slice(offset, offset + 20) : [];
  return {
    counts,
    certified_songs: songs.length,
    certified_artists: artists.size,
    songs: selectedSongs,
    artists: selectedArtists,
    total_record_count: total,
    current_page: page,
    items_per_page: 20,
    has_next_page: offset + 20 < total,
  };
}
