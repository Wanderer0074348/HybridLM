import { useState } from "react";
import { Link } from "react-router-dom";
import { classifyURL, usePRReview } from "@/hooks/usePRReview";
import { RunPanel } from "./RunPanel";
import { PRHeader } from "./PRHeader";
import { IssueHeader } from "./IssueHeader";
import { PrepPanel } from "./PrepPanel";
import { ComparisonBar } from "./ComparisonBar";
import { Icon } from "@/components/ui/Icon";
import { Spinner } from "@/components/ui/Spinner";

const SAMPLE_PR = "https://github.com/Wanderer0074348/HybridLM/pull/1";
const SAMPLE_ISSUE = "https://github.com/Wanderer0074348/HybridLM/issues/1";

export function ReviewPage() {
  const { state, start, stop } = usePRReview();
  const [url, setUrl] = useState("");

  const running = state.status === "fetching" || state.status === "running";

  const [validationError, setValidationError] = useState<string | null>(null);

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = url.trim();
    if (!trimmed || running) return;
    if (!classifyURL(trimmed)) {
      setValidationError(
        "URL must be a GitHub PR (/pull/N) or issue (/issues/N) link."
      );
      return;
    }
    setValidationError(null);
    start(trimmed);
  };

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-bg-base">
      <header className="flex items-center justify-between border-b border-border-subtle px-6 py-3">
        <div className="flex items-center gap-3">
          <Link to="/" className="icon-btn" aria-label="Back to chat">
            ←
          </Link>
          <h1 className="font-serif text-lg text-text-primary">PR &amp; Issue Review Showcase</h1>
          <span className="rounded-md border border-accent/30 bg-accent/5 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-accent">
            demo
          </span>
        </div>
      </header>

      <form onSubmit={onSubmit} className="flex items-center gap-2 border-b border-border-subtle px-6 py-3">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={`${SAMPLE_PR}  •  ${SAMPLE_ISSUE}`}
          disabled={running}
          className="flex-1 rounded-lg border border-border-default bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus-ring"
        />
        <button
          type="submit"
          disabled={running || !url.trim()}
          className="btn-primary"
        >
          {running ? <Spinner size={14} /> : <Icon.Sparkle size={14} />}
          {running ? "Reviewing…" : "Run review"}
        </button>
        {running && (
          <button type="button" onClick={stop} className="btn-ghost">
            Stop
          </button>
        )}
      </form>

      <div className="flex flex-1 flex-col gap-3 overflow-y-auto p-6">
        {(validationError || state.error) && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/5 px-4 py-2 text-sm text-red-400">
            {validationError || state.error}
          </div>
        )}

        {state.pr && <PRHeader pr={state.pr} />}
        {state.issue && <IssueHeader issue={state.issue} />}

        {state.hybrid.totals && state.cloud.totals && (
          <ComparisonBar hybrid={state.hybrid.totals} cloud={state.cloud.totals} />
        )}

        {state.kind === "issue" && <PrepPanel prep={state.prep} />}

        {state.status === "idle" && !state.pr && !state.issue && <EmptyHint />}

        {(state.pr || state.issue || running) && (
          <div className="grid min-h-[400px] flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
            <RunPanel mode="hybrid" run={state.hybrid} />
            <RunPanel mode="cloud" run={state.cloud} />
          </div>
        )}
      </div>
    </div>
  );
}

function EmptyHint() {
  return (
    <div className="flex flex-1 items-center justify-center">
      <div className="max-w-md text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-accent text-white">
          <Icon.Sparkle size={20} />
        </div>
        <h2 className="font-serif text-xl text-text-primary">Paste a GitHub PR or issue URL</h2>
        <p className="mt-2 text-sm text-text-secondary">
          Two agents review the target in parallel — one routed by HybridLM, one locked
          to GPT-5.5. For issues, the agent first searches the repo and reads the most
          relevant code, then proposes solutions.
        </p>
      </div>
    </div>
  );
}
