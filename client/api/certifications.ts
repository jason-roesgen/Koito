export type Certification = {
  tier: "gold" | "platinum" | "diamond";
  multiple: number;
  plays: number;
  reached_at: string;
};
export type TrackCertifications = {
  track_id: number;
  title: string;
  plays: number;
  current: Certification | null;
  next: Pick<Certification, "tier" | "multiple" | "plays">;
  history: Certification[];
};
export type ArtistCertifications = {
  counts: Record<Certification["tier"], number>;
  tracks: TrackCertifications[];
};
export type MilestoneDate = { target: number; reached_at: string | null };
export type MilestoneHistory = {
  plays: number;
  minutes: number;
  play_milestones: MilestoneDate[];
  minute_milestones: MilestoneDate[];
};
export function certificationLabel(award: Pick<Certification, "tier" | "multiple">) {
  const tier = award.tier[0].toUpperCase() + award.tier.slice(1);
  return award.multiple === 1 ? tier : `${award.multiple.toLocaleString()}× ${tier}`;
}

export type HallCredit = { id: number; name: string; image: import("./api").ImageList };
export type HallSong = {
  track_id: number; title: string; artists: HallCredit[];
  image: import("./api").ImageList; plays: number; current: Certification;
};
export type HallArtist = HallCredit & {
  counts: Record<Certification["tier"], number>;
  certified_songs: number; reached_at: string;
};
export type HallOfFame = {
  counts: Record<Certification["tier"], number>;
  certified_songs: number; certified_artists: number;
  songs: HallSong[]; artists: HallArtist[];
  total_record_count: number; current_page: number; items_per_page: number;
  has_next_page: boolean;
};
