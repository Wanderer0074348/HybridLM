package prreview

import (
	"context"
	"sync"
)

const maxDiffChars = 12000

type EventSink func(Event)

type Orchestrator struct{}

func NewOrchestrator() *Orchestrator { return &Orchestrator{} }

func (o *Orchestrator) Run(ctx context.Context, plan []Subtask, infer Inferencer, sink EventSink) {
	mode := infer.Mode()
	sink(newEvent(EventStart, mode))

	results := map[SubtaskID]string{}
	totals := &RunTotals{Subtasks: len(plan)}
	executed := map[SubtaskID]bool{}

	for {
		ready := readyTasks(plan, executed)
		if len(ready) == 0 {
			break
		}

		out := o.runParallel(ctx, ready, results, infer, sink, mode)
		for id, r := range out {
			results[id] = r.Text
			executed[id] = true
			totals.TotalCost += r.Cost.TotalCost
			totals.TotalLatencyMs += r.Latency.Milliseconds()
			totals.TotalTokens += r.Cost.TotalTokens
			if isLLM(r.ModelUsed) {
				totals.LLMCount++
			} else {
				totals.SLMCount++
			}
		}

		if len(out) != len(ready) {
			break
		}
	}

	final := newEvent(EventComplete, mode)
	final.Totals = totals
	sink(final)
}

func readyTasks(plan []Subtask, executed map[SubtaskID]bool) []Subtask {
	ready := []Subtask{}
	for _, t := range plan {
		if executed[t.ID] {
			continue
		}
		if depsSatisfied(t, executed) {
			ready = append(ready, t)
		}
	}
	return ready
}

func depsSatisfied(t Subtask, executed map[SubtaskID]bool) bool {
	for _, d := range t.DependsOn {
		if !executed[d] {
			return false
		}
	}
	return true
}

func (o *Orchestrator) runParallel(
	ctx context.Context,
	tasks []Subtask,
	results map[SubtaskID]string,
	infer Inferencer,
	sink EventSink,
	mode string,
) map[SubtaskID]*InferenceOutput {
	out := map[SubtaskID]*InferenceOutput{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	resultsSnapshot := map[SubtaskID]string{}
	for k, v := range results {
		resultsSnapshot[k] = v
	}

	for _, t := range tasks {
		wg.Add(1)
		go func(task Subtask) {
			defer wg.Done()

			running := newEvent(EventSubtask, mode)
			running.Subtask = &SubtaskEvent{ID: task.ID, Label: task.Label, Status: StatusRunning}
			sink(running)

			prompt := task.BuildPrompt(resultsSnapshot)
			result, err := infer.Run(ctx, prompt)

			done := newEvent(EventSubtask, mode)
			if err != nil {
				done.Subtask = &SubtaskEvent{
					ID: task.ID, Label: task.Label,
					Status: StatusFailed, Error: err.Error(),
				}
				sink(done)
				return
			}

			done.Subtask = &SubtaskEvent{
				ID:        task.ID,
				Label:     task.Label,
				Status:    StatusDone,
				ModelUsed: result.ModelUsed,
				Reason:    result.Reason,
				LatencyMs: result.Latency.Milliseconds(),
				Cost:      result.Cost.TotalCost,
				Tokens:    result.Cost.TotalTokens,
				Result:    result.Text,
			}
			sink(done)

			mu.Lock()
			out[task.ID] = result
			mu.Unlock()
		}(t)
	}
	wg.Wait()
	return out
}

func isLLM(model string) bool {
	for _, p := range []string{"gpt-", "o1", "o3", "claude-"} {
		if hasPrefix(model, p) {
			return true
		}
	}
	return false
}

func hasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
