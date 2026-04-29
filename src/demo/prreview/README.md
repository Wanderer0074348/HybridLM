# PR Review Demo

Showcases HybridLM's routing on a real agentic workload: a code-review agent decomposes a PR into 8 subtasks and runs the same plan twice in parallel — once routed by HybridLM (edge SLM ⇄ cloud LLM per subtask) and once locked to GPT-5.5 — streaming results to the UI for live comparison.

## Endpoint

```
GET /api/v1/demo/pr-review?url=https://github.com/<owner>/<repo>/pull/<n>
```

Server-Sent Events stream. Each event is JSON with `type` ∈ `{start, pr_fetched, subtask, complete, error}` and a `mode` field tagging which run (`hybrid` or `cloud`). Subtask events include `model_used`, `latency_ms`, `cost`, `tokens`, and the markdown `result`.

## Env

```
GITHUB_TOKEN=ghp_…   # personal access token, repo: read scope
```

Public PRs work without a token but you'll hit GitHub's 60 req/hr unauth limit. With a token: 5000/hr.

## Subtask plan

| # | Subtask | Depends on | Expected route |
|---|---|---|---|
| 1 | extract_metadata | – | SLM |
| 2 | classify_files | – | SLM |
| 3 | surface_observations | classify | SLM |
| 4 | semantic_review | metadata, classify | mixed |
| 5 | security_scan | – | mixed |
| 6 | architectural_review | classify, semantic | LLM |
| 7 | risk_score | surface, semantic, security, architect | SLM |
| 8 | executive_summary | risk, semantic, architect | LLM |

Subtasks at the same dependency level run in parallel within each run.

## Files

- `github.go` — PR URL parsing + GitHub API client (metadata + diff + files)
- `subtasks.go` — subtask definitions, prompts, dependency DAG
- `inferencer.go` — `Inferencer` interface, `HybridInferencer` (uses QueryRouter + engines), `CloudInferencer` (always LLM)
- `orchestrator.go` — runs the DAG against a given inferencer, emits events
- `events.go` — SSE event types
- `handler.go` — Gin SSE endpoint that fans out hybrid + cloud runs in parallel against a shared emitter
