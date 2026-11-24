import type { CertificationMetric } from "api/certifications";

export default function CertificationMetricSwitch({
  value,
  onChange,
}: {
  value: CertificationMetric;
  onChange: (metric: CertificationMetric) => void;
}) {
  return (
    <div role="group" aria-label="Certification metric" className="card flex gap-2 w-fit max-w-full p-1">
      {([["plays", "Plays"], ["time", "Listening time"]] as const).map(([metric, label]) => (
        <button
          key={metric}
          type="button"
          className="period-selector px-3 py-2 rounded-(--border-radius) aria-pressed:bg-(--color-bg-tertiary)"
          aria-pressed={value === metric}
          onClick={() => onChange(metric)}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
