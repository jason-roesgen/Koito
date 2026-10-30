// Fixed all-time targets. Listening-time values are stored in minutes so that
// eligibility does not depend on rounding the displayed hours.
export const playMilestones = [1, 100, 500, 1_000, 5_000, 10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000];
export const minuteMilestones = [1, 10, 50, 100, 500, 1_000, 2_500, 5_000, 10_000].map(hours => hours * 60);

export function milestoneProgress(value: number, targets: readonly number[]) {
  const current = Number.isFinite(value) ? Math.max(0, value) : 0;
  const reached = targets.filter(target => target <= current);
  const next = targets.find(target => target > current);
  const latest = reached.at(-1);
  const percent = next === undefined ? 100 : Math.min(100, current / next * 100);
  return { current, reached, latest, next, percent };
}
