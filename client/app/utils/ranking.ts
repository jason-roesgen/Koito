export function rankingLabel(item: { listen_count: number; time_listened: number }, rankBy: string) {
  return rankBy === "time"
    ? `${Math.floor(item.time_listened / 60).toLocaleString()} min (${item.listen_count.toLocaleString()} plays)`
    : `${item.listen_count.toLocaleString()} plays`;
}
