import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { apiFetch } from "api/api";
import { certificationLabel, certificationTotal, type CertificationMetric, type Certification, type TrackCertifications, type ArtistCertifications } from "api/certifications";
import CertificationMetricSwitch from "./CertificationMetricSwitch";
import CardHeader from "./primitives/CardHeader";

const colors = { gold: "#e5b85e", platinum: "#c8d6df", diamond: "#83e6eb" };
const awardIcons = { gold: "📀", platinum: "💿", diamond: "💎" };

export function CertificationDate({ value }: { value: string }) {
  return <time dateTime={value} title={new Date(value).toLocaleString()}>{new Date(value).toLocaleString()}</time>;
}

export function AwardBadge({ award }: { award: Certification }) {
  return (
    <span className="inline-flex items-center gap-2 font-semibold" style={{ color: colors[award.tier] }}>
      <span aria-hidden="true" className="inline-flex w-7 shrink-0 items-center justify-center text-2xl leading-none">{awardIcons[award.tier]}</span>
      {certificationLabel(award)}
    </span>
  );
}

function AwardHistory({ history, metric }: { history: Certification[]; metric: CertificationMetric }) {
  return (
    <details className="mt-4">
      <summary className="cursor-pointer text-sm">Certification history ({history.length})</summary>
      <ol className="mt-2 divide-y divide-(--color-bg-tertiary) border-y border-(--color-bg-tertiary)">
        {history.map(award => (
          <li key={award.seconds ?? award.plays} className="flex flex-wrap items-center gap-x-4 gap-y-1 px-2 py-2.5">
            <div className="min-w-32 max-w-full">
              <AwardBadge award={award} />
            </div>
            <p className="max-w-full text-sm text-(--color-fg-secondary)">
              <span className="text-(--color-fg)">{certificationTotal(award, metric)}</span>
              <span aria-hidden="true"> – </span>
              <CertificationDate value={award.reached_at} />
            </p>
          </li>
        ))}
      </ol>
    </details>
  );
}

export default function CertificationBoard({ type, id }: { type: "track" | "artist"; id: number }) {
  const [metric, setMetric] = useState<CertificationMetric>("plays");
  const isTime = metric === "time";
  const heading = type === "artist" ? "Artist certification board" : "Song certifications";
  const { data, isPending, isError, error } = useQuery({
    queryKey: ["certifications", type, id, metric],
    queryFn: () => apiFetch<TrackCertifications | ArtistCertifications>(`/apis/web/v1/${type}/${id}/certifications`, { metric }),
  });
  return (
    <section className="w-full max-w-[1000px] min-w-0" aria-label={heading}>
      <CardHeader isOffset>{heading}</CardHeader>
      <div className="card p-5 sm:p-6">
        <CertificationMetricSwitch value={metric} onChange={setMetric} />
        <p className="text-sm text-(--color-fg-secondary) my-4">
          {isTime
            ? "Gold: 500 min · Platinum: 1,000 min · Diamond: every 2,500 min"
            : "Gold: 200 plays · Platinum: 400 · Diamond: 1,000"}
        </p>
        {isPending ? <p role="status">Loading certifications…</p> : isError ? (
          <p role="alert" className="error">Error: {error.message}</p>
        ) : data && "track_id" in data ? (
          <TrackBoard track={data} metric={metric} />
        ) : data && "tracks" in data ? (
          <>
            <div className="grid grid-cols-3 gap-3 mb-5">
              {(["gold", "platinum", "diamond"] as const).map(tier => (
                <div key={tier} className="border border-(--color-bg-tertiary) rounded-(--border-radius) p-3">
                  <p className="font-semibold text-2xl" style={{ color: colors[tier] }}>{data.counts[tier].toLocaleString()}</p>
                  <p className="text-sm capitalize">{tier} songs</p>
                </div>
              ))}
            </div>
            <p className="text-xs text-(--color-fg-secondary) mb-4">Each song counts once under its current certificate. All credited artists receive credit.</p>
            {data.tracks.length === 0 ? <p>No certified songs yet. A song earns its first Gold at {isTime ? "500 estimated minutes" : "200 plays"}.</p> : (
              <ul className="flex flex-col gap-4">
                {[...data.tracks].sort((a, b) => (b.current?.seconds ?? b.current?.plays ?? 0) - (a.current?.seconds ?? a.current?.plays ?? 0) || a.track_id - b.track_id).map(track => (
                  <li key={track.track_id} className="border-t border-(--color-bg-tertiary) pt-4">
                    <Link to={`/track/${track.track_id}`} className="font-semibold hover:underline">{track.title}</Link>
                    {track.current && <div className="mt-2"><AwardBadge award={track.current} /></div>}
                    <p className="text-sm mt-2 text-(--color-fg-secondary)">
                      {certificationTotal(track, metric)}{track.current && <> · Reached <CertificationDate value={track.current.reached_at} /></>}
                    </p>
                    <AwardHistory history={track.history} metric={metric} />
                  </li>
                ))}
              </ul>
            )}
          </>
        ) : null}
        <p className="text-xs text-(--color-fg-secondary) mt-5">
          {isTime
            ? "Estimated from the stored track duration per recorded play. Tracks without a known duration earn no time credit. Duration corrections can change award dates."
            : "Each 200-play step gets one certificate. Diamond wins at shared thresholds."}
          {" "}Dates use your local timezone.
        </p>
      </div>
    </section>
  );
}

function TrackBoard({ track, metric }: { track: TrackCertifications; metric: CertificationMetric }) {
  const isTime = metric === "time";
  const value = isTime ? track.seconds ?? 0 : track.plays;
  const { next } = track;
  const target = next ? (isTime ? next.seconds ?? 0 : next.plays) : 0;
  const start = isTime ? track.current?.seconds ?? 0 : 0;
  const percent = target > start ? (value - start) / (target - start) * 100 : 0;
  return (
    <>
      {track.current ? (
        <div>
          <div className="text-2xl"><AwardBadge award={track.current} /></div>
          <p className="text-sm mt-2 text-(--color-fg-secondary)">
            Reached <CertificationDate value={track.current.reached_at} /> · {certificationTotal(track.current, metric)}
          </p>
        </div>
      ) : <p>No certification yet</p>}
      {next && (
        <>
          <p className="mt-4">Next: <strong>{certificationLabel(next)}</strong> at {certificationTotal(next, metric)}</p>
          <div role="progressbar" aria-label={`Progress toward ${certificationLabel(next)}`} aria-valuemin={start} aria-valuemax={target} aria-valuenow={value}
            className="h-2 bg-(--color-bg-tertiary) rounded-full overflow-hidden mt-3">
            <div className="h-full bg-(--color-primary) rounded-full" style={{ width: `${percent}%` }} />
          </div>
        </>
      )}
      <p className="text-sm text-(--color-fg-secondary) mt-2">
        {certificationTotal(track, metric)}
        {next && <> · {certificationTotal({ plays: target - value, seconds: target - value }, metric)} to go</>}
      </p>
      {isTime && track.plays > 0 && value === 0 && (
        <p className="text-sm mt-2">Track duration is unknown. These listens still count toward play awards.</p>
      )}
      {track.history.length > 0 && <AwardHistory history={track.history} metric={metric} />}
    </>
  );
}
