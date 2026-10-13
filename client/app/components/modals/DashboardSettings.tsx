import { useAppContext } from "~/providers/AppProvider";
import { defaults, sections, sectionKeys } from "~/hooks/useDashboardPreferences";

export default function DashboardSettings() {
  const { dashboard } = useAppContext();
  const { rankBy, updateRankBy, visible, ready, storageError, updateVisibility } = dashboard;
  return (
    <section>
      <h2 className="text-xl font-semibold">Dashboard</h2>
      <p className="text-sm mt-3 text-(--color-fg-secondary)">
        Choose which sections to show. Saved in this browser.
      </p>
      <fieldset disabled={!ready} className="mt-4">
        <legend className="sr-only">Visible dashboard sections</legend>
        <div className="flex flex-col gap-2 mb-6">
          <label htmlFor="dashboard-ranking">Rank by</label>
          <select id="dashboard-ranking" value={rankBy}
            className="p-2 border rounded-(--border-radius) bg-(--color-bg)"
            onChange={event => updateRankBy(event.target.value === "time" ? "time" : "plays")}>
            <option value="plays">Plays</option>
            <option value="time">Listening time (minutes)</option>
          </select>
          <p className="text-sm text-(--color-fg-secondary)">
            Applies to dashboard rankings and Artist, Album, and Track pages. Time is estimated from track durations per play; unknown durations count as zero.
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {sectionKeys.map(key => (
            <label key={key} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={visible[key]}
                onChange={event => updateVisibility({ ...visible, [key]: event.target.checked })}
                className="h-4 w-4 accent-(--color-primary)"
              />
              {sections[key]}
            </label>
          ))}
        </div>
        <button className="default mt-4" type="button" onClick={() => { updateVisibility({ ...defaults }); updateRankBy("plays"); }}>
          Reset to defaults
        </button>
      </fieldset>
      {storageError && (
        <p role="status" className="text-sm mt-3">
          Browser preferences could not be loaded or saved. You can still customize this page for this visit.
        </p>
      )}
    </section>
  );
}
