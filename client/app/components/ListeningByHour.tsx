import { useQuery } from "@tanstack/react-query";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { apiFetch } from "api/api";
import CardHeader from "./primitives/CardHeader";

type Hour = { hour: number; plays: number; seconds: number };
type HourlyListening = { timezone: string; hours: Hour[] };
const hourLabel = (hour: number) => `${String(hour).padStart(2, "0")}:00`;

export default function ListeningByHour({ period, metric }: { period: string; metric: "plays" | "time" }) {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const { data, isPending, isError, error } = useQuery({
    queryKey: ["listening-by-hour", period, timezone],
    queryFn: () => apiFetch<HourlyListening>("/apis/web/v1/listening-by-hour", { period, tz: timezone }),
  });
  const minutes = metric === "time";
  const unit = minutes ? "minutes" : "plays";
  const hours = data?.hours.map(hour => ({ ...hour, value: minutes ? hour.seconds / 60 : hour.plays })) ?? [];
  const totalPlays = data?.hours.reduce((sum, hour) => sum + hour.plays, 0) ?? 0;
  const totalValue = hours.reduce((sum, hour) => sum + hour.value, 0);

  return (
    <section className="w-full max-w-[900px] min-w-0" aria-label="Listening by hour">
      <CardHeader isOffset>Listening by hour</CardHeader>
      <div className="card p-4 sm:p-6">
        {isPending ? <p role="status">Loading hourly listening…</p> : isError ? (
          <p role="alert" className="error">Error: {error.message}</p>
        ) : (
          <>
            <p className="text-sm text-(--color-fg-secondary) mb-4">
              {data.timezone} · {minutes ? "Minutes" : "Plays"} by the hour each listen started
            </p>
            {totalPlays === 0 ? <p>No listens in this period.</p> : minutes && totalValue === 0 ? (
              <p>No known track durations in this period. Switch to Plays to see hourly activity.</p>
            ) : (
              <div className="w-full h-[240px] sm:h-[280px]">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={hours} margin={{ top: 10, right: 4, left: 0, bottom: 5 }} accessibilityLayer>
                    <CartesianGrid vertical={false} stroke="var(--color-bg-tertiary)" />
                    <XAxis dataKey="hour" ticks={[0, 4, 8, 12, 16, 20, 23]} tickFormatter={hourLabel} tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }} />
                    <YAxis width={48} allowDecimals={minutes} tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }} />
                    <Tooltip
                      labelFormatter={hour => `${hourLabel(Number(hour))}–${hourLabel((Number(hour) + 1) % 24)}`}
                      formatter={value => [Number(value).toLocaleString(undefined, { maximumFractionDigits: 1 }), unit]}
                      contentStyle={{ background: "var(--color-bg-secondary)", borderColor: "var(--color-bg-tertiary)", color: "var(--color-fg)" }}
                      cursor={{ fill: "var(--color-bg-tertiary)" }}
                    />
                    <Bar dataKey="value" name={unit} fill="var(--color-primary)" radius={[3, 3, 0, 0]} isAnimationActive={false} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
            {minutes && totalPlays > 0 && (
              <p className="text-xs text-(--color-fg-secondary) mt-3">
                Estimated from full track durations, assigned to the start hour. Unknown durations count as zero.
              </p>
            )}
            <details className="mt-3 text-sm text-(--color-fg-secondary)">
              <summary className="cursor-pointer">Hourly values ({unit})</summary>
              <table className="w-full mt-2 text-left">
                <caption className="sr-only">Hourly listening in {data.timezone}</caption>
                <thead><tr><th scope="col">Hour</th><th scope="col">Plays</th><th scope="col">Minutes</th></tr></thead>
                <tbody>{hours.map(hour => (
                  <tr key={hour.hour}>
                    <th scope="row">{hourLabel(hour.hour)}</th>
                    <td>{hour.plays.toLocaleString()}</td>
                    <td>{(hour.seconds / 60).toLocaleString(undefined, { maximumFractionDigits: 1 })}</td>
                  </tr>
                ))}</tbody>
              </table>
            </details>
          </>
        )}
      </div>
    </section>
  );
}
