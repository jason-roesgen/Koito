import { useAppContext } from "~/providers/AppProvider";
import { defaults, sections, sectionKeys } from "~/hooks/useDashboardPreferences";

export default function DashboardSettings() {
  const { dashboard } = useAppContext();
  const { visible, ready, storageError, updateVisibility } = dashboard;
  return (
    <section>
      <h2 className="text-xl font-semibold">Dashboard</h2>
      <p className="text-sm mt-3 text-(--color-fg-secondary)">
        Choose which sections to show. Saved in this browser.
      </p>
      <fieldset disabled={!ready} className="mt-4">
        <legend className="sr-only">Visible dashboard sections</legend>
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
        <button className="default mt-4" type="button" onClick={() => updateVisibility({ ...defaults })}>
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
