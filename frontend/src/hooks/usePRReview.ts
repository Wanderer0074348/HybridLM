import { useCallback, useRef, useState } from "react";
import type {
  IssuePayload,
  PRPayload,
  ReviewEvent,
  ReviewKind,
  ReviewMode,
  RunTotals,
  SubtaskEvent,
} from "@/types/review";

const BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

const ENDPOINT: Record<ReviewKind, string> = {
  pr: "/demo/pr-review",
  issue: "/demo/issue-review",
};

export function classifyURL(raw: string): ReviewKind | null {
  const t = raw.trim();
  if (/^https?:\/\/github\.com\/[^/]+\/[^/]+\/pull\/\d+/.test(t)) return "pr";
  if (/^https?:\/\/github\.com\/[^/]+\/[^/]+\/issues\/\d+/.test(t)) return "issue";
  return null;
}

export interface RunState {
  subtasks: Record<string, SubtaskEvent>;
  order: string[];
  totals: RunTotals | null;
  done: boolean;
}

const emptyRun = (): RunState => ({ subtasks: {}, order: [], totals: null, done: false });

export interface PRReviewState {
  status: "idle" | "fetching" | "running" | "done" | "error";
  kind: ReviewKind | null;
  pr: PRPayload | null;
  issue: IssuePayload | null;
  prep: RunState;
  hybrid: RunState;
  cloud: RunState;
  error: string | null;
}

const initial: PRReviewState = {
  status: "idle",
  kind: null,
  pr: null,
  issue: null,
  prep: emptyRun(),
  hybrid: emptyRun(),
  cloud: emptyRun(),
  error: null,
};

export function usePRReview() {
  const [state, setState] = useState<PRReviewState>(initial);
  const sourceRef = useRef<EventSource | null>(null);

  const stop = useCallback(() => {
    sourceRef.current?.close();
    sourceRef.current = null;
  }, []);

  const start = useCallback(
    (rawURL: string) => {
      const kind = classifyURL(rawURL);
      if (!kind) {
        setState({
          ...initial,
          status: "error",
          error: "URL must be a GitHub PR (/pull/N) or issue (/issues/N) link.",
        });
        return;
      }

      stop();
      setState({ ...initial, kind, status: "fetching" });

      const params = new URLSearchParams({ url: rawURL });
      const es = new EventSource(`${BASE_URL}${ENDPOINT[kind]}?${params}`, { withCredentials: true });
      sourceRef.current = es;

      let completedRuns = 0;

      es.onmessage = (msg) => {
        let event: ReviewEvent;
        try {
          event = JSON.parse(msg.data);
        } catch {
          return;
        }

        setState((prev) => applyEvent(prev, event));

        if (event.type === "complete") {
          completedRuns += 1;
          if (completedRuns >= 2) {
            es.close();
            sourceRef.current = null;
            setState((p) => ({ ...p, status: "done" }));
          }
        }
        if (event.type === "error") {
          es.close();
          sourceRef.current = null;
        }
      };

      es.onerror = () => {
        es.close();
        sourceRef.current = null;
        setState((p) =>
          p.status === "done"
            ? p
            : { ...p, status: "error", error: p.error || "Connection lost" }
        );
      };
    },
    [stop]
  );

  return { state, start, stop };
}

function applyEvent(prev: PRReviewState, e: ReviewEvent): PRReviewState {
  switch (e.type) {
    case "pr_fetched":
      if (e.pr) return { ...prev, pr: e.pr, status: "running" };
      if (e.issue) return { ...prev, issue: e.issue, status: "running" };
      return prev;
    case "error":
      return { ...prev, status: "error", error: e.error || "Unknown error" };
    case "subtask": {
      if (!e.mode || !e.subtask) return prev;
      return updateRun(prev, e.mode as ReviewMode, (run) => upsertSubtask(run, e.subtask!));
    }
    case "complete": {
      if (!e.mode || !e.totals) return prev;
      return updateRun(prev, e.mode as ReviewMode, (run) => ({ ...run, totals: e.totals!, done: true }));
    }
    default:
      return prev;
  }
}

function updateRun(state: PRReviewState, mode: ReviewMode, fn: (run: RunState) => RunState): PRReviewState {
  switch (mode) {
    case "hybrid":
      return { ...state, hybrid: fn(state.hybrid) };
    case "cloud":
      return { ...state, cloud: fn(state.cloud) };
    case "prep":
      return { ...state, prep: fn(state.prep) };
    default:
      return state;
  }
}

function upsertSubtask(run: RunState, ev: SubtaskEvent): RunState {
  const exists = run.order.includes(ev.id);
  return {
    ...run,
    subtasks: { ...run.subtasks, [ev.id]: { ...run.subtasks[ev.id], ...ev } },
    order: exists ? run.order : [...run.order, ev.id],
  };
}

export function modeLabel(mode: ReviewMode): string {
  switch (mode) {
    case "hybrid":
      return "HybridLM";
    case "cloud":
      return "Cloud LLM";
    case "prep":
      return "Prep";
  }
}
