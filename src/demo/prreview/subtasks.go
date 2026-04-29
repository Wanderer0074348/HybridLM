package prreview

import (
	"fmt"
	"strings"
)

type SubtaskID string

const (
	TaskClassify   SubtaskID = "classify_files"
	TaskMetadata   SubtaskID = "extract_metadata"
	TaskSurface    SubtaskID = "surface_observations"
	TaskSemantic   SubtaskID = "semantic_review"
	TaskSecurity   SubtaskID = "security_scan"
	TaskArchitect  SubtaskID = "architectural_review"
	TaskRiskScore  SubtaskID = "risk_score"
	TaskSummary    SubtaskID = "executive_summary"
)

type Subtask struct {
	ID          SubtaskID
	Label       string
	Description string
	DependsOn   []SubtaskID
	BuildPrompt func(results map[SubtaskID]string) string
}

func resultOr(results map[SubtaskID]string, id SubtaskID) string {
	if v, ok := results[id]; ok {
		return v
	}
	return ""
}

func PRPlan(pr *PRPayload) []Subtask {
	diff := DiffSlice(pr.Diff, maxDiffChars)
	files := pr.Files
	meta := pr.Metadata

	return []Subtask{
		{
			ID:          TaskMetadata,
			Label:       "Extract PR metadata",
			Description: "Pull title, intent, branches into a structured snapshot.",
			BuildPrompt: func(_ map[SubtaskID]string) string {
				return fmt.Sprintf(`Extract a one-line intent statement and a list of stated goals from this PR.

Title: %s
Author: %s
Base → Head: %s → %s
Body:
%s

Respond with:
INTENT: <one sentence>
GOALS:
- <goal 1>
- <goal 2>
...`, meta.Title, meta.Author, meta.BaseBranch, meta.HeadBranch, truncate(meta.Body, 1500))
			},
		},
		{
			ID:          TaskClassify,
			Label:       "Classify changed files",
			Description: "Bucket each file: feature / refactor / test / config / docs / generated.",
			BuildPrompt: func(_ map[SubtaskID]string) string {
				lines := []string{}
				for _, f := range files {
					lines = append(lines, fmt.Sprintf("- %s (+%d/-%d, %s)", f.Filename, f.Additions, f.Deletions, f.Status))
				}
				return fmt.Sprintf(`Classify each changed file into ONE bucket:
feature, refactor, test, config, docs, generated, other.

Files:
%s

Respond as a markdown table with columns | File | Bucket | One-line reason |.`, strings.Join(lines, "\n"))
			},
		},
		{
			ID:          TaskSurface,
			Label:       "Surface observations",
			Description: "Naming, dead code, obvious style issues across the diff.",
			DependsOn:   []SubtaskID{TaskClassify},
			BuildPrompt: func(_ map[SubtaskID]string) string {
				return fmt.Sprintf(`Scan this diff for surface-level issues only: bad naming, dead code, debug prints, TODOs left in, obvious style violations. Skip semantic concerns.

Diff:
%s

Respond as a bulleted list. If nothing, say "No surface issues found."`, diff)
			},
		},
		{
			ID:          TaskSemantic,
			Label:       "Semantic review",
			Description: "What the change actually does, and whether it's coherent.",
			DependsOn:   []SubtaskID{TaskMetadata, TaskClassify},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Review this PR diff for correctness and coherence with its stated intent.

Stated intent:
%s

File classification:
%s

Diff:
%s

Identify: logic bugs, missing error handling, off-by-ones, incorrect API use, inconsistent invariants. Be specific — cite file and approximate line. Limit to the top 6 issues by impact.`,
					resultOr(r, TaskMetadata),
					truncate(resultOr(r, TaskClassify), 1200),
					diff)
			},
		},
		{
			ID:          TaskSecurity,
			Label:       "Security scan",
			Description: "Credentials, injection, unsafe deserialization, auth gaps.",
			BuildPrompt: func(_ map[SubtaskID]string) string {
				return fmt.Sprintf(`Look for security issues in this diff: hardcoded secrets, SQL/command injection, path traversal, missing auth checks, unsafe deserialization, weak crypto, sensitive data in logs.

Diff:
%s

Respond as a bulleted list of CONCRETE findings with file path. If clean, say "No security issues found."`, diff)
			},
		},
		{
			ID:          TaskArchitect,
			Label:       "Architectural review",
			Description: "Cross-file concerns, abstraction fit, breaking changes.",
			DependsOn:   []SubtaskID{TaskClassify, TaskSemantic},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Reason about cross-file architectural concerns in this PR.

File classification:
%s

Semantic findings:
%s

Diff:
%s

Identify: breaking changes to callers, abstraction leaks, layering violations, test coverage gaps for the new behavior, missing migration steps. Limit to 4 highest-leverage concerns.`,
					truncate(resultOr(r, TaskClassify), 1200),
					truncate(resultOr(r, TaskSemantic), 1500),
					diff)
			},
		},
		{
			ID:          TaskRiskScore,
			Label:       "Risk score",
			Description: "LOW / MEDIUM / HIGH per category.",
			DependsOn:   []SubtaskID{TaskSurface, TaskSemantic, TaskSecurity, TaskArchitect},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Given the findings below, assign a single risk level (LOW / MEDIUM / HIGH) to each category, with a one-line justification.

Surface:
%s

Semantic:
%s

Security:
%s

Architectural:
%s

Respond as a markdown table | Category | Risk | Why |.`,
					truncate(resultOr(r, TaskSurface), 800),
					truncate(resultOr(r, TaskSemantic), 1200),
					truncate(resultOr(r, TaskSecurity), 800),
					truncate(resultOr(r, TaskArchitect), 1200))
			},
		},
		{
			ID:          TaskSummary,
			Label:       "Executive summary",
			Description: "What a reviewer should do next, ranked.",
			DependsOn:   []SubtaskID{TaskRiskScore, TaskSemantic, TaskArchitect},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Write a tight code-review summary for this PR. Two paragraphs max plus a "Recommended actions" list of 3-5 concrete next steps, ordered by priority.

Intent: %s

Risk table:
%s

Semantic findings:
%s

Architectural findings:
%s`,
					resultOr(r, TaskMetadata),
					resultOr(r, TaskRiskScore),
					truncate(resultOr(r, TaskSemantic), 1500),
					truncate(resultOr(r, TaskArchitect), 1200))
			},
		},
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…[truncated]"
}

func DiffSlice(diff string, max int) string {
	if len(diff) <= max {
		return diff
	}
	return diff[:max] + "\n…[diff truncated for length]"
}
