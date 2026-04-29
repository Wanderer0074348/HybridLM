package prreview

import "time"

type EventType string

const (
	EventStart      EventType = "start"
	EventFetched    EventType = "pr_fetched"
	EventSubtask    EventType = "subtask"
	EventComplete   EventType = "complete"
	EventError      EventType = "error"
)

type SubtaskStatus string

const (
	StatusRunning SubtaskStatus = "running"
	StatusDone    SubtaskStatus = "done"
	StatusFailed  SubtaskStatus = "failed"
)

type Event struct {
	Type      EventType     `json:"type"`
	Mode      string        `json:"mode,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Subtask   *SubtaskEvent `json:"subtask,omitempty"`
	PR        *PRPayload    `json:"pr,omitempty"`
	Issue     *IssuePayload `json:"issue,omitempty"`
	Totals    *RunTotals    `json:"totals,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type SubtaskEvent struct {
	ID         SubtaskID     `json:"id"`
	Label      string        `json:"label"`
	Status     SubtaskStatus `json:"status"`
	ModelUsed  string        `json:"model_used,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	LatencyMs  int64         `json:"latency_ms,omitempty"`
	Cost       float64       `json:"cost,omitempty"`
	Tokens     int           `json:"tokens,omitempty"`
	Result     string        `json:"result,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type RunTotals struct {
	TotalCost      float64 `json:"total_cost"`
	TotalLatencyMs int64   `json:"total_latency_ms"`
	TotalTokens    int     `json:"total_tokens"`
	Subtasks       int     `json:"subtasks"`
	SLMCount       int     `json:"slm_count"`
	LLMCount       int     `json:"llm_count"`
}

func newEvent(t EventType, mode string) Event {
	return Event{Type: t, Mode: mode, Timestamp: time.Now()}
}
