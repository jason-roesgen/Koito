import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "api/api";
import { certificationTotal, type Certification, type CertificationMetric, type TrackCertifications } from "api/certifications";
import { AwardBadge, CertificationDate } from "./CertificationBoard";

export default function HallSongAwardHistory({
    trackId,
    defaultOpen,
    metric,
}: {
    trackId: number;
    defaultOpen: boolean;
    metric: CertificationMetric;
}) {
    const [open, setOpen] = useState(defaultOpen);

    return (
        <details
            className="w-full border-t border-(--color-bg-tertiary) pt-3"
            open={open}
            onToggle={event => setOpen(event.currentTarget.open)}
        >
            <summary className="cursor-pointer text-sm font-semibold">Certification history</summary>
            {open && <History trackId={trackId} metric={metric} />}
        </details>
    );
}

function History({ trackId, metric }: { trackId: number; metric: CertificationMetric }) {
    const { data, isPending, isError, refetch, isFetching } = useQuery({
        queryKey: ["certifications", "track", trackId, metric],
        queryFn: () => apiFetch<TrackCertifications>(`/apis/web/v1/track/${trackId}/certifications`, { metric }),
        retry: false,
    });

    if (isPending) return <p className="mt-3 text-sm" role="status">Loading certification history…</p>;
    if (isError) {
        return (
            <div className="flex items-center gap-3 mt-2 text-sm">
                <p role="alert">Couldn’t load certification history.</p>
                <button className="default" disabled={isFetching} onClick={() => void refetch()}>
                    Try again
                </button>
            </div>
        );
    }

    const [first] = data.history;
    if (!first || !data.current) return <p className="mt-3 text-sm">No certifications recorded.</p>;

    return (
        <div className="mt-4">
            <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <AwardSummary label="First certification" award={first} />
                <AwardSummary label="Current certification" award={data.current} />
            </dl>
            <ol aria-label="Certifications earned, newest first" className="mt-4 ml-2 border-l border-(--color-bg-tertiary)">
                {[...data.history].reverse().map(award => (
                    <li key={award.seconds ?? award.plays} className="relative py-2 pl-4">
                        <span aria-hidden="true" className="absolute -left-1 top-5 size-2 rounded-full bg-(--color-primary)" />
                        <AwardBadge award={award} />
                        <p className="text-sm text-(--color-fg-secondary)">
                            {certificationTotal(award, metric)} · <CertificationDate value={award.reached_at} />
                        </p>
                    </li>
                ))}
            </ol>
        </div>
    );
}

function AwardSummary({ label, award }: { label: string; award: Certification }) {
    return (
        <div className="rounded-(--border-radius) border border-(--color-bg-tertiary) p-3">
            <dt className="mb-2 text-sm text-(--color-fg-secondary)">{label}</dt>
            <dd>
                <AwardBadge award={award} />
                <p className="mt-1 text-sm"><CertificationDate value={award.reached_at} /></p>
            </dd>
        </div>
    );
}
