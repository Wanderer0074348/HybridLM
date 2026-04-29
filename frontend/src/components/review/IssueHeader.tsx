import type { IssuePayload } from "@/types/review";

export function IssueHeader({ issue }: { issue: IssuePayload }) {
  const { metadata: m } = issue;
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
      <h2 className="mt-1 text-base font-semibold text-text-primary">
        <span className="mr-2 text-text-muted">#{m.number}</span>
        {m.title}
      </h2>
      <p className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-text-secondary">
        <span><span className="text-text-muted">by</span> {m.author}</span>
        <span className="rounded-md border border-border-subtle bg-bg-base px-1.5 py-0.5 font-mono text-[10.5px] text-text-muted">
          {m.state}
        </span>
        {m.labels.map((l) => (
          <span
            key={l}
            className="rounded-md border border-accent/30 bg-accent/5 px-1.5 py-0.5 text-[10.5px] text-accent"
          >
            {l}
          </span>
        ))}
        <span>{issue.comments.length} comments</span>
      </p>
    </div>
  );
}
