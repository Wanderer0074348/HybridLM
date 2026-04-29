package prreview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

type Handler struct {
	github       *GitHubClient
	router       *router.QueryRouter
	slm          models.SLMInferencer
	llm          models.LLMInferencer
	llmModelName string
	slmModelName string
	orchestrator *Orchestrator
}

type Config struct {
	GitHubToken  string
	Router       *router.QueryRouter
	SLM          models.SLMInferencer
	LLM          models.LLMInferencer
	LLMModelName string
	SLMModelName string
}

func NewHandler(cfg Config) *Handler {
	return &Handler{
		github:       NewGitHubClient(cfg.GitHubToken),
		router:       cfg.Router,
		slm:          cfg.SLM,
		llm:          cfg.LLM,
		llmModelName: cfg.LLMModelName,
		slmModelName: cfg.SLMModelName,
		orchestrator: NewOrchestrator(),
	}
}

type ReviewRequest struct {
	URL string `json:"url" form:"url"`
}

func (h *Handler) Stream(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		var req ReviewRequest
		_ = c.ShouldBind(&req)
		url = req.URL
	}
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing url"})
		return
	}

	ref, err := ParsePRURL(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flusher, ok := setupSSE(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	emitter := newSSEEmitter(c.Writer, flusher)

	pr, err := h.github.Fetch(ctx, ref)
	if err != nil {
		emitter.send(Event{Type: EventError, Error: fmt.Sprintf("github fetch failed: %v", err)})
		return
	}

	fetched := newEvent(EventFetched, "")
	fetched.PR = &PRPayload{Metadata: pr.Metadata, Files: pr.Files}
	emitter.send(fetched)

	plan := PRPlan(pr)
	h.runDual(ctx, plan, emitter)
}

func (h *Handler) StreamIssue(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing url"})
		return
	}

	ref, err := ParseIssueURL(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flusher, ok := setupSSE(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	emitter := newSSEEmitter(c.Writer, flusher)

	issue, err := h.github.FetchIssue(ctx, ref)
	if err != nil {
		emitter.send(Event{Type: EventError, Error: fmt.Sprintf("github fetch failed: %v", err)})
		return
	}

	fetched := newEvent(EventFetched, "")
	fetched.Issue = &IssuePayload{Metadata: issue.Metadata, Comments: issue.Comments}
	emitter.send(fetched)

	prepInfer := NewHybridInferencer(h.router, h.slm, h.llm, h.llmModelName, h.slmModelName)
	prep, err := PrepareIssueContext(ctx, h.github, issue, prepInfer, emitter.send)
	if err != nil {
		emitter.send(Event{Type: EventError, Error: fmt.Sprintf("prep failed: %v", err)})
		return
	}

	issue.Keywords = prep.Keywords
	issue.Files = prep.Files

	plan := IssuePlan(issue)
	h.runDual(ctx, plan, emitter)
}

func (h *Handler) runDual(ctx context.Context, plan []Subtask, emitter *sseEmitter) {
	hybrid := NewHybridInferencer(h.router, h.slm, h.llm, h.llmModelName, h.slmModelName)
	cloud := NewCloudInferencer(h.llm, h.llmModelName)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		h.orchestrator.Run(ctx, plan, hybrid, emitter.send)
	}()
	go func() {
		defer wg.Done()
		h.orchestrator.Run(ctx, plan, cloud, emitter.send)
	}()
	wg.Wait()
}

func setupSSE(c *gin.Context) (http.Flusher, bool) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return nil, false
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})

	return flusher, true
}

type sseEmitter struct {
	mu      sync.Mutex
	w       io.Writer
	flusher http.Flusher
}

func newSSEEmitter(w io.Writer, f http.Flusher) *sseEmitter {
	return &sseEmitter{w: w, flusher: f}
}

func (e *sseEmitter) send(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	payload, err := json.Marshal(ev)
	if err != nil {
		log.Printf("sse marshal failed: %v", err)
		return
	}
	if _, err := fmt.Fprintf(e.w, "data: %s\n\n", payload); err != nil {
		return
	}
	e.flusher.Flush()
}
