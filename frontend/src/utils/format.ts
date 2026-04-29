export function formatLatency(ns: number): string {
  if (!ns || ns <= 0) return "—";
  const ms = ns / 1_000_000;
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function formatCost(usd: number | undefined): string {
  if (usd === undefined || usd === null) return "—";
  if (usd === 0) return "$0";
  if (usd < 0.001) return `$${(usd * 1000).toFixed(3)}m`;
  return `$${usd.toFixed(4)}`;
}

export function formatRelative(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diff = Date.now() - then;
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return new Date(iso).toLocaleDateString();
}

export function truncate(s: string, n: number): string {
  return s.length <= n ? s : `${s.slice(0, n)}…`;
}

export function modelLabel(model: string): string {
  if (!model) return "auto";
  if (model.includes("edge") || model.toLowerCase().includes("slm")) return "Edge SLM";
  if (model.includes("cloud") || model.toLowerCase().includes("llm")) return "Cloud LLM";
  return model;
}
