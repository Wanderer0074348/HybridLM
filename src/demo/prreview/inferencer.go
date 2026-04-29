package prreview

import (
	"context"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
	"www.github.com/Wanderer0074348/HybridLM/src/utils"
)

const slmCooldown = 1800 * time.Millisecond

var slmGate = make(chan struct{}, 1)

func acquireSLM() {
	slmGate <- struct{}{}
}

func releaseSLM() {
	time.Sleep(slmCooldown)
	<-slmGate
}

type InferenceOutput struct {
	Text      string
	ModelUsed string
	Reason    string
	Latency   time.Duration
	Cost      *models.CostMetrics
}

type Inferencer interface {
	Mode() string
	Run(ctx context.Context, prompt string) (*InferenceOutput, error)
}

type HybridInferencer struct {
	router    *router.QueryRouter
	slm       models.SLMInferencer
	llm       models.LLMInferencer
	llmModel  string
	slmModel  string
}

func NewHybridInferencer(r *router.QueryRouter, slm models.SLMInferencer, llm models.LLMInferencer, llmModel, slmModel string) *HybridInferencer {
	return &HybridInferencer{router: r, slm: slm, llm: llm, llmModel: llmModel, slmModel: slmModel}
}

func (h *HybridInferencer) Mode() string { return "hybrid" }

func (h *HybridInferencer) Run(ctx context.Context, prompt string) (*InferenceOutput, error) {
	req := &models.InferenceRequest{Query: prompt}
	start := time.Now()

	decision, err := h.router.Route(ctx, req)
	if err != nil {
		return nil, err
	}

	var text, modelUsed, specific string
	if decision.UseLLM {
		text, err = h.llm.Infer(ctx, req)
		modelUsed, specific = "cloud-llm", h.llmModel
	} else {
		acquireSLM()
		defer releaseSLM()
		text, err = h.slm.Infer(ctx, req)
		modelUsed, specific = "edge-slm", h.slmModel
	}
	if err != nil {
		return nil, err
	}

	latency := time.Since(start)
	cost := utils.CalculateCostMetrics(prompt, text, modelUsed, specific, false, false)

	return &InferenceOutput{
		Text:      text,
		ModelUsed: specific,
		Reason:    decision.Reason,
		Latency:   latency,
		Cost:      cost,
	}, nil
}

type CloudInferencer struct {
	llm      models.LLMInferencer
	llmModel string
}

func NewCloudInferencer(llm models.LLMInferencer, llmModel string) *CloudInferencer {
	return &CloudInferencer{llm: llm, llmModel: llmModel}
}

func (c *CloudInferencer) Mode() string { return "cloud" }

func (c *CloudInferencer) Run(ctx context.Context, prompt string) (*InferenceOutput, error) {
	req := &models.InferenceRequest{Query: prompt}
	start := time.Now()

	text, err := c.llm.Infer(ctx, req)
	if err != nil {
		return nil, err
	}

	latency := time.Since(start)
	cost := utils.CalculateCostMetrics(prompt, text, "cloud-llm", c.llmModel, false, false)

	return &InferenceOutput{
		Text:      text,
		ModelUsed: c.llmModel,
		Reason:    "cloud-only baseline",
		Latency:   latency,
		Cost:      cost,
	}, nil
}
