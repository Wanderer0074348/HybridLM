import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { SubtaskEvent } from "@/types/review";
import { Spinner } from "@/components/ui/Spinner";
import { Icon } from "@/components/ui/Icon";
import { formatCost, formatLatency, modelLabel } from "@/utils/format";

type Props = { task: SubtaskEvent };

export function SubtaskCard({ task }: Props) {
  const [expanded, setExpanded] = useState(false);
  const isEdge = task.model_used?.toLowerCase().includes("llama") || task.model_used?.toLowerCase().includes("slm") || task.model_used?.toLowerCase().includes("scout") || task.model_used?.toLowerCase().includes("oss");

  const tone =
    task.status === "failed"
      ? "border-red-500/30 bg-red-500/5"
      : task.status === "running"
        ? "border-border-default bg-bg-elevated"
        : "border-border-subtle bg-bg-elevated";

  return (
    <div className={`rounded-lg border ${tone} transition-colors`}>
      <button
        type="button"
        onClick={() => task.result && setExpanded((v) => !v)}
        className="flex w-full items-center gap-3 px-3 py-2.5 text-left"
      >
        <span className="flex h-5 w-5 flex-shrink-0 items-center justify-center">
          {task.status === "running" && <Spinner size={12} />}
          {task.status === "done" && (
            <span className="h-2 w-2 rounded-full bg-emerald-400" />
          )}
          {task.status === "failed" && (
            <span className="h-2 w-2 rounded-full bg-red-400" />
          )}
        </span>

        <span className="flex-1 truncate text-sm font-medium text-text-primary">{task.label}</span>

        {task.status === "done" && (
          <div className="flex items-center gap-1.5">
            {task.model_used && (
              <span
                className={`inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10.5px] font-medium ${
                  isEdge
                    ? "border-emerald-400/20 bg-emerald-400/5 text-emerald-400/90"
                    : "border-sky-400/20 bg-sky-400/5 text-sky-400/90"
                }`}
                title={task.reason}
              >
                {isEdge ? <Icon.Bolt size={11} /> : <Icon.Cloud size={11} />}
                {modelLabel(isEdge ? "edge-slm" : "cloud-llm")}
              </span>
            )}
            <span className="rounded-md border border-border-subtle bg-bg-base px-1.5 py-0.5 text-[10.5px] font-medium text-text-muted">
              {formatLatency((task.latency_ms ?? 0) * 1_000_000)}
            </span>
            <span className="rounded-md border border-border-subtle bg-bg-base px-1.5 py-0.5 text-[10.5px] font-medium text-text-muted">
              {formatCost(task.cost ?? 0)}
            </span>
          </div>
        )}
      </button>

      {expanded && task.result && (
        <div className="markdown border-t border-border-subtle px-4 py-3 text-[13px] text-text-secondary">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{task.result}</ReactMarkdown>
        </div>
      )}

      {task.status === "failed" && task.error && (
        <div className="border-t border-red-500/20 px-4 py-2 text-xs text-red-400">{task.error}</div>
      )}
    </div>
  );
}
