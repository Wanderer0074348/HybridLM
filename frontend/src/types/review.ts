export type SubtaskStatus = "running" | "done" | "failed";

export interface SubtaskEvent {
  id: string;
  label: string;
  status: SubtaskStatus;
  model_used?: string;
  reason?: string;
  latency_ms?: number;
  cost?: number;
  tokens?: number;
  result?: string;
  error?: string;
}

export interface RunTotals {
  total_cost: number;
  total_latency_ms: number;
  total_tokens: number;
  subtasks: number;
  slm_count: number;
  llm_count: number;
}

export interface PRMetadata {
  url: string;
  title: string;
  body: string;
  author: string;
  base_branch: string;
  head_branch: string;
  state: string;
  changed_files: number;
  additions: number;
  deletions: number;
}

export interface PRFile {
  filename: string;
  status: string;
  additions: number;
  deletions: number;
}

export interface PRPayload {
  metadata: PRMetadata;
  files: PRFile[];
}

export interface IssueMetadata {
  url: string;
  title: string;
  body: string;
  author: string;
  state: string;
  labels: string[];
  number: number;
}

export interface IssueComment {
  author: string;
  body: string;
}

export interface IssuePayload {
  metadata: IssueMetadata;
  comments: IssueComment[];
  keywords?: string[];
  files?: { path: string; bytes: number }[];
}

export type ReviewMode = "hybrid" | "cloud" | "prep";
export type ReviewKind = "pr" | "issue";

export interface ReviewEvent {
  type: "start" | "pr_fetched" | "subtask" | "complete" | "error";
  mode?: ReviewMode | "";
  timestamp: string;
  subtask?: SubtaskEvent;
  pr?: PRPayload;
  issue?: IssuePayload;
  totals?: RunTotals;
  error?: string;
}
