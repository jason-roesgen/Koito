import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "api/api";
import type { MilestoneHistory, MilestoneDate } from "api/certifications";
import { CertificationDate } from "./CertificationBoard";
import CardHeader from "./primitives/CardHeader";
import { milestoneProgress, minuteMilestones, playMilestones } from "~/utils/milestones";

export default function ListeningMilestones() {
  const { data, isPending, isError, error } = useQuery({
    queryKey: ["milestone-history"],
    queryFn: () => apiFetch<MilestoneHistory>("/apis/web/v1/milestones"),
  });

  return (
    <section className="w-full max-w-[900px] min-w-0" aria-label="Listening milestones">
      <CardHeader isOffset>Listening milestones</CardHeader>
      {isPending ? (
        <p role="status" className="card p-6">Loading milestones…</p>
      ) : isError ? (
        <p role="alert" className="card p-6 error">Error: {error.message}</p>
      ) : (
        <>
          <p className="text-sm text-(--color-fg-secondary) mb-4 px-1">
            All-time progress · Includes imported listening history
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <MilestoneCard title="Plays" value={data.plays} targets={playMilestones} dates={data.play_milestones} />
            <MilestoneCard title="Listening hours" value={data.minutes} targets={minuteMilestones} dates={data.minute_milestones} hours />
          </div>
          <p className="text-xs text-(--color-fg-secondary) mt-3 px-1">
            Hours are estimated from known track durations. Unknown durations contribute zero time. Reached dates use your local timezone and current duration metadata.
          </p>
        </>
      )}
    </section>
  );
}

function MilestoneCard({ title, value, targets, dates, hours = false }: {
  title: string;
  value: number;
  targets: readonly number[];
  dates: MilestoneDate[];
  hours?: boolean;
}) {
  const { current, reached, latest, next, percent } = milestoneProgress(value, targets);
  const dateByTarget = new Map(dates.map(date => [date.target, date.reached_at]));
  const latestDate = latest === undefined ? null : dateByTarget.get(latest);
  const unit = hours ? "hours" : "plays";
  const amount = (value: number, decimals = 0) => {
    const scaled = hours ? value / 60 : value;
    // Do not round an incomplete hour milestone up to the target.
    const display = decimals ? Math.floor(scaled * 10 ** decimals) / 10 ** decimals : scaled;
    return display.toLocaleString(undefined, { maximumFractionDigits: decimals });
  };
  const remaining = next === undefined ? "" : hours && next - current < 60
    ? `${Math.ceil(next - current).toLocaleString()} minutes`
    : `${amount(next - current, hours ? 1 : 0)} ${unit}`;
  const nextLabel = next === undefined ? "All targets reached" : `${amount(next)} ${unit}`;

  return (
    <div className="card p-5 sm:p-6 min-w-0">
      <h3 className="text-sm text-(--color-fg-secondary)">{title}</h3>
      <p className="text-2xl font-semibold text-(--color-primary) mt-3">
        {latest === undefined ? "Your first milestone awaits" : `${amount(latest)} ${unit} reached`}
      </p>
      {latestDate && <p className="text-xs text-(--color-fg-secondary) mt-2">Reached <CertificationDate value={latestDate} /></p>}
      <p className="text-sm mt-3">
        {next === undefined ? "You’ve reached every target in this collection." : `Next milestone: ${nextLabel}`}
      </p>
      <div
        role="progressbar"
        aria-label={`${title}: ${nextLabel}`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.floor(percent)}
        aria-valuetext={`${amount(current, hours ? 1 : 0)} ${unit}${next === undefined ? ", all targets reached" : ` of ${nextLabel}`}`}
        className="h-2 rounded-full bg-(--color-bg-tertiary) mt-4 overflow-hidden"
      >
        <div className="h-full bg-(--color-primary) rounded-full" style={{ width: `${percent}%` }} />
      </div>
      <p className="text-sm text-(--color-fg-secondary) mt-2">
        {amount(current, hours ? 1 : 0)} {unit} total
        {next !== undefined && ` · ${remaining} to go`}
      </p>
      <details className="mt-4 text-sm">
        <summary className="cursor-pointer">Reached milestones ({reached.length}/{targets.length})</summary>
        {reached.length === 0 ? (
          <p className="text-(--color-fg-secondary) mt-3">No milestones reached yet. Your listening history will build your progress.</p>
        ) : (
          <ul className="flex flex-wrap gap-2 mt-3" aria-label={`${title} milestones reached`}>
            {reached.map(target => (
              <li key={target} className="border border-(--color-bg-tertiary) rounded-(--border-radius) px-3 py-1">
                <div>{amount(target)} {unit}</div>
                {dateByTarget.get(target) && <div className="text-xs text-(--color-fg-secondary) mt-1"><CertificationDate value={dateByTarget.get(target)!} /></div>}
              </li>
            ))}
          </ul>
        )}
      </details>
    </div>
  );
}
