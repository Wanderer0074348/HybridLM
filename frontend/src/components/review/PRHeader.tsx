import type { PRPayload } from "@/types/review";

export function PRHeader({ pr }: { pr: PRPayload }) {
  const { metadata: m } = pr;
  return (
    <div className="rounded-xl border border-border-subtle bg-bg-elevated px-5 py-4">
      <a
        href={m.url}
        target="_blank"
        rel="noopener noreferrer"
        className="text-xs text-text-muted hover:text-accent"
      >
        {m.url}
      </a>
      <h2 className="mt-1 text-base font-semibold text-text-primary">{m.title}</h2>
      <p className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-text-secondary">
        <span><span className="text-text-muted">by</span> {m.author}</span>
        <span className="font-mono">{m.base_branch} ← {m.head_branch}</span>
        <span>{m.changed_files} files</span>
        <span className="text-emerald-400">+{m.additions}</span>
        <span className="text-red-400">−{m.deletions}</span>
      </p>
    </div>
  );
}
