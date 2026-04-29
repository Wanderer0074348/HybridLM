package prreview

import (
	"context"
	"fmt"
	"strings"
)

const (
	maxSearchResults = 8
	maxFetchedFiles  = 5
	fetchByteCap     = 8000
)

type PrepResult struct {
	Keywords []string
	Files    []FileContent
}

func PrepareIssueContext(
	ctx context.Context,
	gh *GitHubClient,
	issue *IssuePayload,
	infer Inferencer,
	sink EventSink,
) (*PrepResult, error) {
	emitPrep(sink, "extract_keywords", "")

	keywords, kwOut, err := extractKeywords(ctx, issue, infer)
	if err != nil {
		emitPrep(sink, "extract_keywords", err.Error())
		return nil, fmt.Errorf("keyword extraction: %w", err)
	}
	emitPrepResult(sink, "extract_keywords", "Extract search keywords", kwOut, strings.Join(keywords, ", "))

	emitPrep(sink, "search_repo", "")
	hits, err := gh.SearchCode(ctx, issue.Ref, keywords, maxSearchResults)
	if err != nil {
		emitPrep(sink, "search_repo", err.Error())
		return nil, fmt.Errorf("code search: %w", err)
	}
	pathList := make([]string, 0, len(hits))
	for _, h := range hits {
		pathList = append(pathList, h.Path)
	}
	emitPrepResult(sink, "search_repo", "Search repo", nil, fmt.Sprintf("Top %d matches:\n- %s", len(pathList), strings.Join(pathList, "\n- ")))

	emitPrep(sink, "fetch_files", "")
	files, err := fetchTopFiles(ctx, gh, issue.Ref, hits)
	if err != nil {
		emitPrep(sink, "fetch_files", err.Error())
		return nil, fmt.Errorf("file fetch: %w", err)
	}
	collected := make([]string, 0, len(files))
	for _, f := range files {
		collected = append(collected, fmt.Sprintf("%s (%d bytes)", f.Path, f.Bytes))
	}
	emitPrepResult(sink, "fetch_files", "Fetch relevant files", nil, fmt.Sprintf("Loaded %d files:\n- %s", len(files), strings.Join(collected, "\n- ")))

	return &PrepResult{Keywords: keywords, Files: files}, nil
}

func extractKeywords(ctx context.Context, issue *IssuePayload, infer Inferencer) ([]string, *InferenceOutput, error) {
	prompt := fmt.Sprintf(`From this GitHub issue, extract 3-5 SHORT search terms that would help find the most relevant code in the repository. Prefer:
- exact symbol names, function names, error message fragments
- specific file/module names mentioned

Output ONLY the terms, one per line. No numbering, no explanation.

%s`, formatIssueText(issue))

	out, err := infer.Run(ctx, prompt)
	if err != nil {
		return nil, nil, err
	}
	keywords := parseKeywords(out.Text)
	return keywords, out, nil
}

func parseKeywords(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		l = strings.TrimPrefix(l, "-")
		l = strings.TrimPrefix(l, "*")
		l = strings.TrimSpace(l)
		l = strings.Trim(l, "\"'`")
		if l == "" || len(l) > 80 {
			continue
		}
		out = append(out, l)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func fetchTopFiles(ctx context.Context, gh *GitHubClient, ref IssueRef, hits []CodeSearchResult) ([]FileContent, error) {
	out := make([]FileContent, 0, maxFetchedFiles)
	seen := map[string]bool{}
	for _, h := range hits {
		if len(out) >= maxFetchedFiles {
			break
		}
		if seen[h.Path] {
			continue
		}
		seen[h.Path] = true

		f, err := gh.FetchFile(ctx, ref.Owner, ref.Repo, h.Path, fetchByteCap)
		if err != nil {
			continue
		}
		out = append(out, *f)
	}
	return out, nil
}

func emitPrep(sink EventSink, id, errMsg string) {
	ev := newEvent(EventSubtask, "prep")
	se := &SubtaskEvent{
		ID:     SubtaskID(id),
		Label:  prepLabel(id),
		Status: StatusRunning,
	}
	if errMsg != "" {
		se.Status = StatusFailed
		se.Error = errMsg
	}
	ev.Subtask = se
	sink(ev)
}

func emitPrepResult(sink EventSink, id, label string, out *InferenceOutput, summary string) {
	ev := newEvent(EventSubtask, "prep")
	se := &SubtaskEvent{
		ID:     SubtaskID(id),
		Label:  label,
		Status: StatusDone,
		Result: summary,
	}
	if out != nil {
		se.ModelUsed = out.ModelUsed
		se.Reason = out.Reason
		se.LatencyMs = out.Latency.Milliseconds()
		se.Cost = out.Cost.TotalCost
		se.Tokens = out.Cost.TotalTokens
	}
	ev.Subtask = se
	sink(ev)
}

func prepLabel(id string) string {
	switch id {
	case "extract_keywords":
		return "Extract search keywords"
	case "search_repo":
		return "Search repo"
	case "fetch_files":
		return "Fetch relevant files"
	default:
		return id
	}
}
