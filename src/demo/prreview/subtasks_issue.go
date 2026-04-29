package prreview

import (
	"fmt"
	"strings"
)

const (
	IssueTaskSummary    SubtaskID = "issue_summary"
	IssueTaskClassify   SubtaskID = "issue_classify"
	IssueTaskMapCode    SubtaskID = "code_relevance_map"
	IssueTaskSuspect    SubtaskID = "identify_suspect_regions"
	IssueTaskRootCause  SubtaskID = "root_cause_hypothesis"
	IssueTaskSolutions  SubtaskID = "propose_solutions"
	IssueTaskRisk       SubtaskID = "solution_risk"
	IssueTaskReport     SubtaskID = "final_report"
)

const (
	maxFileChars     = 6000
	maxBundledFiles  = 5
	maxCommentChars  = 4000
)

func IssuePlan(issue *IssuePayload) []Subtask {
	issueText := formatIssueText(issue)
	codeBundle := formatFileBundle(issue.Files)

	return []Subtask{
		{
			ID:          IssueTaskSummary,
			Label:       "Summarize issue",
			Description: "Distill issue + comments into a tight bullet brief.",
			BuildPrompt: func(_ map[SubtaskID]string) string {
				return fmt.Sprintf(`Summarize this GitHub issue in 4-6 bullets covering: what the user reports, what they expected, environment/repro steps, and any error messages or stack traces. Be tight, no fluff.

%s`, issueText)
			},
		},
		{
			ID:          IssueTaskClassify,
			Label:       "Classify issue",
			Description: "Bug / feature / question / performance / docs / regression.",
			BuildPrompt: func(_ map[SubtaskID]string) string {
				return fmt.Sprintf(`Classify this issue into ONE category:
bug, feature_request, question, performance, docs, regression, other.

%s

Respond as:
CATEGORY: <one word>
REASON: <one sentence>`, issueText)
			},
		},
		{
			ID:          IssueTaskMapCode,
			Label:       "Map code relevance",
			Description: "For each fetched file, why it's likely relevant to the issue.",
			DependsOn:   []SubtaskID{IssueTaskSummary},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`For each file below, write ONE line: what this file does and why it might be relevant to the reported issue.

Issue summary:
%s

Files:
%s

Respond as a markdown table | File | Purpose | Relevance |.`,
					resultOr(r, IssueTaskSummary),
					codeBundle)
			},
		},
		{
			ID:          IssueTaskSuspect,
			Label:       "Identify suspect regions",
			Description: "Pinpoint specific functions / blocks most likely to be causing the issue.",
			DependsOn:   []SubtaskID{IssueTaskSummary, IssueTaskMapCode},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Pinpoint the most likely suspect lines, functions, or blocks across the provided code. Cite file paths and approximate line numbers. Be specific. Limit to top 5.

Issue summary:
%s

Code map:
%s

Code:
%s`,
					resultOr(r, IssueTaskSummary),
					truncate(resultOr(r, IssueTaskMapCode), 1200),
					codeBundle)
			},
		},
		{
			ID:          IssueTaskRootCause,
			Label:       "Root-cause hypothesis",
			Description: "Concrete theory of what is happening and why.",
			DependsOn:   []SubtaskID{IssueTaskSuspect},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Form a concrete root-cause hypothesis for this issue. State: what is happening, why, and which lines of code support the theory. If the evidence is insufficient, say so explicitly and list what additional info would close the gap.

Issue summary:
%s

Suspect regions:
%s

Code:
%s`,
					resultOr(r, IssueTaskSummary),
					resultOr(r, IssueTaskSuspect),
					codeBundle)
			},
		},
		{
			ID:          IssueTaskSolutions,
			Label:       "Propose solutions",
			Description: "2-3 distinct solution paths with code sketches.",
			DependsOn:   []SubtaskID{IssueTaskRootCause},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Propose 2-3 DISTINCT solution paths for this issue. For each:
- **Approach** (one sentence)
- **Code sketch** (a fenced code block with a minimal change, including file path)
- **Tradeoffs** (one sentence)

Root cause:
%s

Code:
%s`,
					resultOr(r, IssueTaskRootCause),
					codeBundle)
			},
		},
		{
			ID:          IssueTaskRisk,
			Label:       "Solution risk assessment",
			Description: "Complexity / blast radius / rollback risk per solution.",
			DependsOn:   []SubtaskID{IssueTaskSolutions},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Score each proposed solution on three axes:
- Complexity: LOW / MEDIUM / HIGH
- Blast radius: LOW / MEDIUM / HIGH
- Rollback risk: LOW / MEDIUM / HIGH

Add a single-sentence justification per solution.

Proposed solutions:
%s

Respond as a markdown table | # | Complexity | Blast | Rollback | Why |.`,
					resultOr(r, IssueTaskSolutions))
			},
		},
		{
			ID:          IssueTaskReport,
			Label:       "Final report",
			Description: "Publishable issue analysis with recommended next step.",
			DependsOn:   []SubtaskID{IssueTaskClassify, IssueTaskRootCause, IssueTaskSolutions, IssueTaskRisk},
			BuildPrompt: func(r map[SubtaskID]string) string {
				return fmt.Sprintf(`Write a publishable issue-analysis report with these sections:

## Summary
(1 paragraph: what the issue is and the proposed direction)

## Root cause
(based on the hypothesis below)

## Recommended next step
(one specific, actionable step)

## Solution paths
(brief table referencing the proposals)

Inputs:

Classification:
%s

Root cause:
%s

Solutions:
%s

Risk assessment:
%s`,
					resultOr(r, IssueTaskClassify),
					resultOr(r, IssueTaskRootCause),
					truncate(resultOr(r, IssueTaskSolutions), 2500),
					resultOr(r, IssueTaskRisk))
			},
		},
	}
}

func formatIssueText(issue *IssuePayload) string {
	m := issue.Metadata
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", m.Title)
	fmt.Fprintf(&b, "Author: %s\n", m.Author)
	fmt.Fprintf(&b, "State: %s\n", m.State)
	if len(m.Labels) > 0 {
		fmt.Fprintf(&b, "Labels: %s\n", strings.Join(m.Labels, ", "))
	}
	fmt.Fprintf(&b, "\nBody:\n%s\n", truncate(m.Body, 4000))

	if len(issue.Comments) > 0 {
		used := 0
		b.WriteString("\nComments:\n")
		for _, c := range issue.Comments {
			snippet := truncate(c.Body, 600)
			used += len(snippet)
			fmt.Fprintf(&b, "- @%s: %s\n", c.Author, snippet)
			if used > maxCommentChars {
				b.WriteString("- …[remaining comments truncated]\n")
				break
			}
		}
	}
	return b.String()
}

func formatFileBundle(files []FileContent) string {
	if len(files) == 0 {
		return "(no relevant code files were located)"
	}
	limit := len(files)
	if limit > maxBundledFiles {
		limit = maxBundledFiles
	}
	var b strings.Builder
	for i := 0; i < limit; i++ {
		f := files[i]
		content := f.Content
		if len(content) > maxFileChars {
			content = content[:maxFileChars] + "\n…[file truncated]"
		}
		fmt.Fprintf(&b, "\n--- File: %s (%d bytes) ---\n", f.Path, f.Bytes)
		b.WriteString(content)
		b.WriteString("\n")
	}
	return b.String()
}
