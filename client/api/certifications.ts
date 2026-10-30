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
