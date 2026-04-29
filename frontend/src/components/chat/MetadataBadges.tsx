import type { ReactNode } from "react";
import type { MessageMeta } from "@/types";
import { Icon } from "@/components/ui/Icon";
import { formatCost, formatLatency, modelLabel } from "@/utils/format";

type Props = { meta: MessageMeta };

type Badge = { icon: ReactNode; label: string; tone?: string; title?: string };

export function MetadataBadges({ meta }: Props) {
  const isEdge = meta.model_used?.toLowerCase().includes("slm") || meta.model_used?.includes("edge");

  const badges: Badge[] = [
    {
      icon: isEdge ? <Icon.Bolt size={12} /> : <Icon.Cloud size={12} />,
      label: modelLabel(meta.model_used || ""),
      tone: isEdge ? "text-emerald-400/90 bg-emerald-400/5 border-emerald-400/20" : "text-sky-400/90 bg-sky-400/5 border-sky-400/20",
      title: meta.routing_reason,
    },
    {
      icon: <span className="font-mono text-[10px]">⏱</span>,
      label: formatLatency(meta.latency || 0),
    },
  ];

  if (meta.cache_hit) {
    badges.push({
      icon: <Icon.Cache size={12} />,
      label: "cached",
      tone: "text-amber-300/90 bg-amber-300/5 border-amber-300/20",
    });
  }

  if (meta.cost_metrics) {
    badges.push({
      icon: <Icon.Coin size={12} />,
      label: formatCost(meta.cost_metrics.total_cost),
      title: `Tokens: ${meta.cost_metrics.total_tokens} • Saved: ${formatCost(meta.cost_metrics.estimated_savings)}`,
    });
  }

  return (
    <div className="mt-2 flex flex-wrap items-center gap-1.5">
      {badges.map((b, i) => (
        <span
          key={i}
          title={b.title}
          className={`inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10.5px] font-medium ${
            b.tone || "border-border-subtle bg-bg-elevated text-text-muted"
          }`}
        >
          {b.icon}
          {b.label}
        </span>
      ))}
    </div>
  );
}
