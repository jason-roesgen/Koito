import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { apiFetch } from "api/api";
import { certificationLabel, type Certification, type TrackCertifications, type ArtistCertifications } from "api/certifications";
import CardHeader from "./primitives/CardHeader";

const colors = { gold: "#e5b85e", platinum: "#c8d6df", diamond: "#83e6eb" };
const awardIcons = { gold: "📀", platinum: "💿", diamond: "💎" };

export function CertificationDate({ value }: { value: string }) {
  return <time dateTime={value} title={new Date(value).toLocaleString()}>{new Date(value).toLocaleString()}</time>;
}

function AwardBadge({ award }: { award: Certification }) {
  return (
    <span className="inline-flex items-center gap-2 font-semibold" style={{ color: colors[award.tier] }}>
      <span aria-hidden="true" className="inline-flex w-7 shrink-0 items-center justify-center text-2xl leading-none">{awardIcons[award.tier]}</span>
      {certificationLabel(award)}
    </span>
  );
}

function AwardHistory({ history }: { history: Certification[] }) {
  return (
    <details className="mt-4">
      <summary className="cursor-pointer text-sm">Certification history ({history.length})</summary>
      <ol className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4">
        {history.map(award => (
          <li key={award.plays} className="border border-(--color-bg-tertiary) rounded-(--border-radius) p-3">
            <AwardBadge award={award} />
            <p className="text-sm mt-2">{award.plays.toLocaleString()} plays</p>
            <p className="text-xs text-(--color-fg-secondary) mt-1"><CertificationDate value={award.reached_at} /></p>
          </li>
        ))}
      </ol>
    </details>
  );
}

export default function CertificationBoard({ type, id }: { type: "track" | "artist"; id: number }) {
  const { data, isPending, isError, error } = useQuery({
    queryKey: ["certifications", type, id],
    queryFn: () => apiFetch<TrackCertifications | ArtistCertifications>(`/apis/web/v1/${type}/${id}/certifications`),
  });
  return (
    <section className="w-full max-w-[1000px] min-w-0" aria-label="Song certifications">
      <CardHeader isOffset>{type === "track" ? "Song certifications" : "Artist certification board"}</CardHeader>
      <div className="card p-5 sm:p-6">
        <p className="text-sm text-(--color-fg-secondary) mb-5">
          Personal listening awards · Gold: 200 plays · Platinum: 400 · Diamond: 1,000
        </p>
        {isPending ? <p role="status">Loading certifications…</p> : isError ? (
          <p role="alert" className="error">Error: {error.message}</p>
        ) : data && "track_id" in data ? (
          <TrackBoard track={data} />
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
            {data.tracks.length === 0 ? <p>No certified songs yet. A song earns its first Gold at 200 plays.</p> : (
              <ul className="flex flex-col gap-4">
                {[...data.tracks].sort((a, b) => (b.current?.plays ?? 0) - (a.current?.plays ?? 0) || a.track_id - b.track_id).map(track => (
                  <li key={track.track_id} className="border-t border-(--color-bg-tertiary) pt-4">
                    <Link to={`/track/${track.track_id}`} className="font-semibold hover:underline">{track.title}</Link>
                    {track.current && <div className="mt-2"><AwardBadge award={track.current} /></div>}
                    <p className="text-sm mt-2 text-(--color-fg-secondary)">
                      {track.plays.toLocaleString()} plays{track.current && <> · Reached <CertificationDate value={track.current.reached_at} /></>}
                    </p>
                    <AwardHistory history={track.history} />
                  </li>
                ))}
              </ul>
            )}
          </>
        ) : null}
        <p className="text-xs text-(--color-fg-secondary) mt-5">
          Calculated from recorded history, including imports. Each 200-play step gets one certificate; Diamond wins at shared thresholds. Dates use your local timezone.
        </p>
      </div>
    </section>
  );
}

function TrackBoard({ track }: { track: TrackCertifications }) {
  const percent = Math.min(100, track.plays / track.next.plays * 100);
  return (
    <>
      {track.current ? (
        <div>
          <div className="text-2xl"><AwardBadge award={track.current} /></div>
          <p className="text-sm mt-2 text-(--color-fg-secondary)">Reached <CertificationDate value={track.current.reached_at} /> · {track.current.plays.toLocaleString()} plays</p>
        </div>
      ) : <p>No certification yet</p>}
      <p className="mt-4">Next: <strong>{certificationLabel(track.next)}</strong> at {track.next.plays.toLocaleString()} plays</p>
      <div role="progressbar" aria-label={`Progress toward ${certificationLabel(track.next)}`} aria-valuemin={0} aria-valuemax={track.next.plays} aria-valuenow={track.plays}
        className="h-2 bg-(--color-bg-tertiary) rounded-full overflow-hidden mt-3">
        <div className="h-full bg-(--color-primary) rounded-full" style={{ width: `${percent}%` }} />
      </div>
      <p className="text-sm text-(--color-fg-secondary) mt-2">{track.plays.toLocaleString()} plays · {(track.next.plays - track.plays).toLocaleString()} to go</p>
      {track.history.length > 0 && <AwardHistory history={track.history} />}
    </>
  );
}
