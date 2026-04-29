import type { RunState } from "@/hooks/usePRReview";
import type { ReviewMode } from "@/types/review";
import { SubtaskCard } from "./SubtaskCard";
import { formatCost, formatLatency } from "@/utils/format";
import { Icon } from "@/components/ui/Icon";

type Props = {
  mode: ReviewMode;
  run: RunState;
};

export function RunPanel({ mode, run }: Props) {
  const isHybrid = mode === "hybrid";

  return (
    <section className="flex h-full min-h-0 flex-col rounded-xl border border-border-subtle bg-bg-base">
      <header className="flex items-center justify-between border-b border-border-subtle px-4 py-3">
        <div className="flex items-center gap-2">
          <span
            className={`flex h-7 w-7 items-center justify-center rounded-md ${
              isHybrid ? "bg-accent text-white" : "bg-sky-500/15 text-sky-400"
            }`}
          >
            {isHybrid ? <Icon.Sparkle size={14} /> : <Icon.Cloud size={14} />}
          </span>
          <div>
            <h2 className="text-sm font-semibold text-text-primary">
              {isHybrid ? "HybridLM (auto-routed)" : "Cloud LLM only"}
            </h2>
            <p className="text-[11px] text-text-muted">
              {isHybrid ? "Edge SLM ⇄ Cloud LLM per subtask" : "GPT-5.5 for every subtask"}
            </p>
          </div>
        </div>
        {run.totals && <Totals totals={run.totals} />}
      </header>

      <div className="flex-1 space-y-2 overflow-y-auto p-4">
        {run.order.length === 0 && (
          <p className="px-1 py-2 text-xs text-text-muted">Waiting for subtasks…</p>
        )}
        {run.order.map((id) => (
          <SubtaskCard key={id} task={run.subtasks[id]} />
        ))}
      </div>
    </section>
  );
}

function Totals({ totals }: { totals: NonNullable<RunState["totals"]> }) {
  return (
    <div className="flex items-center gap-3 text-[11px]">
      <Stat label="cost" value={formatCost(totals.total_cost)} />
      <Stat label="latency" value={formatLatency(totals.total_latency_ms * 1_000_000)} />
      <Stat label="SLM/LLM" value={`${totals.slm_count}/${totals.llm_count}`} />
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col items-end">
      <span className="font-mono text-xs font-semibold text-text-primary">{value}</span>
      <span className="text-[10px] uppercase tracking-wider text-text-muted">{label}</span>
    </div>
  );
}
