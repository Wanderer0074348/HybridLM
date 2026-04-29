import type { RunTotals } from "@/types/review";
import { formatCost, formatLatency } from "@/utils/format";

type Props = {
  hybrid: RunTotals | null;
  cloud: RunTotals | null;
};

export function ComparisonBar({ hybrid, cloud }: Props) {
  if (!hybrid || !cloud) return null;

  const costSaved = cloud.total_cost - hybrid.total_cost;
  const costPct = cloud.total_cost > 0 ? (costSaved / cloud.total_cost) * 100 : 0;
  const latencyDelta = cloud.total_latency_ms - hybrid.total_latency_ms;

  return (
    <div className="flex items-center justify-between gap-6 rounded-xl border border-accent/30 bg-accent/5 px-5 py-3">
      <div>
        <p className="text-[11px] uppercase tracking-wider text-accent/80">HybridLM savings</p>
        <p className="mt-0.5 font-mono text-lg font-semibold text-text-primary">
          {formatCost(costSaved)} <span className="text-sm text-text-secondary">({costPct.toFixed(0)}%)</span>
        </p>
      </div>
      <div>
        <p className="text-[11px] uppercase tracking-wider text-text-muted">Latency delta</p>
        <p className="mt-0.5 font-mono text-sm text-text-secondary">
          {latencyDelta >= 0 ? "−" : "+"}
          {formatLatency(Math.abs(latencyDelta) * 1_000_000)}
        </p>
      </div>
      <div>
        <p className="text-[11px] uppercase tracking-wider text-text-muted">Routing mix</p>
        <p className="mt-0.5 font-mono text-sm text-text-secondary">
          {hybrid.slm_count} SLM • {hybrid.llm_count} LLM
        </p>
      </div>
    </div>
  );
}
