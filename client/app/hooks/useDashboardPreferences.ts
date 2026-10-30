import { useEffect, useState } from "react";

export const sections = {
  stats: "All time stats",
  activity: "Activity",
  hourly: "Listening by hour",
  milestones: "Listening milestones",
  artists: "Top artists",
  albums: "Top albums",
  tracks: "Top tracks",
  recent: "Last played",
} as const;
type Section = keyof typeof sections;
type Visibility = Record<Section, boolean>;
export const sectionKeys = Object.keys(sections) as Section[];
export const defaults: Visibility = {
  stats: true, activity: true, hourly: true, milestones: true, artists: true,
  albums: true, tracks: true, recent: true,
};
const storageKey = "koito.dashboard.visibility.v1";

export function useDashboardPreferences() {
  const [rankBy, setRankBy] = useState<"plays" | "time">("plays");
  const [visible, setVisible] = useState<Visibility>(defaults);
  const [ready, setReady] = useState(false);
  const [storageError, setStorageError] = useState(false);

  // Read after mounting so browser preferences do not cause hydration mismatches.
  useEffect(() => {
    try {
      setRankBy(localStorage.getItem("koito.dashboard.rankBy.v1") === "time" ? "time" : "plays");
      const raw = localStorage.getItem(storageKey);
      const saved: unknown = raw ? JSON.parse(raw) : null;
      if (saved && typeof saved === "object" && !Array.isArray(saved)) {
        const next = { ...defaults };
        for (const key of sectionKeys) {
          const value = (saved as Record<string, unknown>)[key];
          if (typeof value === "boolean") next[key] = value;
        }
        setVisible(next);
      }
    } catch {
      setStorageError(true);
    }
    setReady(true);
  }, []);

  function updateVisibility(next: Visibility) {
    setVisible(next);
    try {
      localStorage.setItem(storageKey, JSON.stringify(next));
      setStorageError(false);
    } catch {
      setStorageError(true);
    }
  }

  function updateRankBy(value: "plays" | "time") {
    setRankBy(value);
    try {
      localStorage.setItem("koito.dashboard.rankBy.v1", value);
      setStorageError(false);
    } catch {
      setStorageError(true);
    }
  }

  return { rankBy, updateRankBy, visible, ready, storageError, updateVisibility };
}
