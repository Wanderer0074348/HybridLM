// MMLU-Lite Benchmark for HybridLM (Experiment 1)
//
// Fetches questions from HuggingFace datasets API and runs them through
// the live HybridLM server, collecting accuracy, cost, latency, and F1.
//
// Usage:
//
//	export HYBRIDLM_SESSION=<session_id from browser after Google OAuth>
//	go run ./tests/mmlu_lite.go
//
// Flags:
//
//	-url    HybridLM base URL        (default: http://localhost:8080)
//	-n      Number of questions      (default: 500)
//	-delay  Ms between requests      (default: 100)
//	-out    Output JSON file         (default: tests/results_mmlu.json)
//	-hf     HuggingFace API token    (optional, for higher rate limits)
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// ── HuggingFace datasets-server types ──────────────────────────────────────

type hfRow struct {
	RowIdx int `json:"row_idx"`
	Row    struct {
		Question string `json:"question"`
		OptionA  string `json:"option_a"`
		OptionB  string `json:"option_b"`
		OptionC  string `json:"option_c"`
		OptionD  string `json:"option_d"`
		Answer   string `json:"answer"` // "A", "B", "C", or "D"
		Subject  string `json:"subject"`
	} `json:"row"`
}

type hfResponse struct {
	Rows     []hfRow `json:"rows"`
	NumTotal int     `json:"num_rows_total"`
}

// ── HybridLM API types ─────────────────────────────────────────────────────

type inferenceRequest struct {
	Query string `json:"query"`
}

type costMetrics struct {
	TotalCost float64 `json:"total_cost"`
}

type inferenceResponse struct {
	Response      string       `json:"response"`
	ModelUsed     string       `json:"model_used"`
	RoutingReason string       `json:"routing_reason"`
	Latency       int64        `json:"latency"` // nanoseconds (Go time.Duration)
	CacheHit      bool         `json:"cache_hit"`
	CostMetrics   *costMetrics `json:"cost_metrics"`
}

// ── Per-question result ────────────────────────────────────────────────────

type questionResult struct {
	Subject         string  `json:"subject"`
	CorrectAnswer   string  `json:"correct_answer"`
	PredictedAnswer string  `json:"predicted_answer"`
	IsCorrect       bool    `json:"is_correct"`
	ModelUsed       string  `json:"model_used"`
	RoutingReason   string  `json:"routing_reason"`
	LatencyS        float64 `json:"latency_s"`
	CostUSD         float64 `json:"cost_usd"`
	CacheHit        bool    `json:"cache_hit"`
	Error           string  `json:"error,omitempty"`
}

// ── Summary metrics ────────────────────────────────────────────────────────

type benchmarkMetrics struct {
	AccuracyPct     float64            `json:"accuracy_pct"`
	TotalCostUSD    float64            `json:"total_cost_usd"`
	AvgLatencyS     float64            `json:"avg_latency_s"`
	F1Score         float64            `json:"f1_score"`
	TotalQuestions  int                `json:"total_questions"`
	CorrectCount    int                `json:"correct_count"`
	SLMRouted       int                `json:"slm_routed"`
	LLMRouted       int                `json:"llm_routed"`
	CacheHits       int                `json:"cache_hits"`
	Errors          int                `json:"errors"`
	SubjectAccuracy map[string]float64 `json:"subject_accuracy"`
}

// ── HuggingFace fetcher ────────────────────────────────────────────────────

func fetchMMLULite(n int, hfToken string) ([]hfRow, error) {
	const (
		dataset  = "CohereLabs/Global-MMLU-Lite"
		config   = "en"
		split    = "test"
		pageSize = 100
	)

	client := &http.Client{Timeout: 30 * time.Second}
	var all []hfRow

	for offset := 0; len(all) < n; offset += pageSize {
		limit := pageSize
		if remaining := n - len(all); remaining < pageSize {
			limit = remaining
		}

		url := fmt.Sprintf(
			"https://datasets-server.huggingface.co/rows?dataset=%s&config=%s&split=%s&offset=%d&limit=%d",
			dataset, config, split, offset, limit,
		)

		req, _ := http.NewRequest("GET", url, nil)
		if hfToken != "" {
			req.Header.Set("Authorization", "Bearer "+hfToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HuggingFace request failed: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HuggingFace API %d: %s", resp.StatusCode, string(body))
		}

		var hfResp hfResponse
		if err := json.Unmarshal(body, &hfResp); err != nil {
			return nil, fmt.Errorf("failed to parse HF response: %w", err)
		}

		all = append(all, hfResp.Rows...)

		if len(hfResp.Rows) == 0 || offset+len(hfResp.Rows) >= hfResp.NumTotal {
			break
		}
	}

	if len(all) > n {
		all = all[:n]
	}
	return all, nil
}

// ── Prompt formatting ──────────────────────────────────────────────────────

func formatPrompt(row hfRow) string {
	return fmt.Sprintf(
		"You are answering a multiple choice question. Reply with ONLY a single letter: A, B, C, or D. No explanation.\n\nQuestion: %s\n\nA. %s\nB. %s\nC. %s\nD. %s\n\nAnswer:",
		row.Row.Question,
		row.Row.OptionA, row.Row.OptionB, row.Row.OptionC, row.Row.OptionD,
	)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ── Answer extraction ──────────────────────────────────────────────────────

var (
	reAnswerIs  = regexp.MustCompile(`(?i)the answer is\s*[:\-]?\s*([A-D])`)
	reAnswerTag = regexp.MustCompile(`(?i)answer\s*[:\-]\s*([A-D])`)
	reStartLine = regexp.MustCompile(`(?m)^\s*([A-D])[.):] `)
	reBare      = regexp.MustCompile(`\b([A-D])\b`)
)

func extractAnswer(text string) string {
	for _, re := range []*regexp.Regexp{reAnswerIs, reAnswerTag, reStartLine, reBare} {
		if m := re.FindStringSubmatch(text); m != nil {
			return strings.ToUpper(m[1])
		}
	}
	return ""
}

// ── HybridLM inference ─────────────────────────────────────────────────────

func callInference(baseURL, session, query string, noCache bool) (*inferenceResponse, float64, error) {
	payload, _ := json.Marshal(inferenceRequest{Query: query})

	req, _ := http.NewRequest("POST", baseURL+"/api/v1/inference", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if noCache {
		req.Header.Set("X-No-Cache", "true")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	wallTime := time.Since(start).Seconds()
	if err != nil {
		return nil, wallTime, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, wallTime, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result inferenceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, wallTime, fmt.Errorf("parse error: %w", err)
	}
	return &result, wallTime, nil
}

// ── Metrics computation ────────────────────────────────────────────────────

func computeMetrics(results []questionResult) benchmarkMetrics {
	m := benchmarkMetrics{
		TotalQuestions:  len(results),
		SubjectAccuracy: make(map[string]float64),
	}

	subjectCorrect := make(map[string]int)
	subjectTotal := make(map[string]int)
	var totalLatency, totalCost float64
	var tp, fn int

	for _, r := range results {
		if r.Error != "" {
			m.Errors++
		}
		if r.IsCorrect {
			m.CorrectCount++
			tp++
		} else {
			fn++
		}
		totalLatency += r.LatencyS
		totalCost += r.CostUSD
		if r.CacheHit {
			m.CacheHits++
		}
		switch r.ModelUsed {
		case "edge-slm":
			m.SLMRouted++
		case "cloud-llm":
			m.LLMRouted++
		}
		subjectTotal[r.Subject]++
		if r.IsCorrect {
			subjectCorrect[r.Subject]++
		}
	}

	if m.TotalQuestions > 0 {
		m.AccuracyPct = math.Round(float64(m.CorrectCount)/float64(m.TotalQuestions)*10000) / 100
		m.AvgLatencyS = math.Round(totalLatency/float64(m.TotalQuestions)*100) / 100
	}
	m.TotalCostUSD = math.Round(totalCost*10000) / 10000

	// F1: precision=1 (no false positives), recall=tp/(tp+fn)
	if tp+fn > 0 {
		m.F1Score = math.Round(float64(tp)/float64(tp+fn)*100) / 100
	}

	for subj, total := range subjectTotal {
		m.SubjectAccuracy[subj] = math.Round(float64(subjectCorrect[subj])/float64(total)*1000) / 10
	}

	return m
}

// ── Output ─────────────────────────────────────────────────────────────────

func printSummary(m benchmarkMetrics) {
	sep := strings.Repeat("=", 55)
	fmt.Println("\n" + sep)
	fmt.Println("  MMLU-Lite Benchmark — HybridLM (Experiment 1)")
	fmt.Println(sep)
	fmt.Printf("  Questions:     %d\n", m.TotalQuestions)
	fmt.Printf("  Accuracy:      %.1f%%  (%d/%d)\n", m.AccuracyPct, m.CorrectCount, m.TotalQuestions)
	fmt.Printf("  Total Cost:    $%.4f\n", m.TotalCostUSD)
	fmt.Printf("  Avg Latency:   %.2fs\n", m.AvgLatencyS)
	fmt.Printf("  F1-Score:      %.2f\n", m.F1Score)
	fmt.Printf("  SLM routed:    %d\n", m.SLMRouted)
	fmt.Printf("  LLM routed:    %d\n", m.LLMRouted)
	fmt.Printf("  Cache hits:    %d\n", m.CacheHits)
	if m.Errors > 0 {
		fmt.Printf("  Errors:        %d\n", m.Errors)
	}
	fmt.Println("\n  Per-subject accuracy:")
	for subj, acc := range m.SubjectAccuracy {
		fmt.Printf("    %-42s %.1f%%\n", subj, acc)
	}
	fmt.Println(sep)
}

// ── Main ───────────────────────────────────────────────────────────────────

func main() {
	baseURL  := flag.String("url", "http://localhost:8080", "HybridLM base URL")
	n        := flag.Int("n", 500, "Number of questions to run")
	delayMs  := flag.Int("delay", 100, "Delay between requests (ms)")
	outFile  := flag.String("out", "tests/results_mmlu.json", "Output JSON file")
	hfToken  := flag.String("hf", os.Getenv("HF_TOKEN"), "HuggingFace API token (or set HF_TOKEN)")
	noCache  := flag.Bool("no-cache", false, "Bypass cache for every request (recommended for benchmarks)")
	flag.Parse()

	session := os.Getenv("HYBRIDLM_SESSION") // optional: /api/v1/inference is public

	// 1. Load questions
	log.Printf("Fetching %d questions from answerdotai/mmlu-lite on HuggingFace...", *n)
	rows, err := fetchMMLULite(*n, *hfToken)
	if err != nil {
		log.Fatalf("Failed to load MMLU-Lite: %v", err)
	}
	log.Printf("Loaded %d questions. Starting benchmark...", len(rows))

	// 2. Run questions
	results := make([]questionResult, 0, len(rows))
	delay := time.Duration(*delayMs) * time.Millisecond

	for i, row := range rows {
		prompt := formatPrompt(row)
		correctLetter := strings.ToUpper(row.Row.Answer)

		resp, wallTime, err := callInference(*baseURL, session, prompt, *noCache)

		qr := questionResult{
			Subject:       row.Row.Subject,
			CorrectAnswer: correctLetter,
		}

		if err != nil {
			qr.Error = err.Error()
			qr.LatencyS = wallTime
			log.Printf("[%d/%d] ERROR: %v", i+1, len(rows), err)
		} else {
			predicted := extractAnswer(resp.Response)
			qr.PredictedAnswer = predicted
			qr.IsCorrect = predicted == correctLetter
			qr.ModelUsed = resp.ModelUsed
			qr.RoutingReason = resp.RoutingReason
			qr.CacheHit = resp.CacheHit
			qr.LatencyS = float64(resp.Latency) / 1e9
			if qr.LatencyS == 0 {
				qr.LatencyS = wallTime
			}
			if resp.CostMetrics != nil {
				qr.CostUSD = resp.CostMetrics.TotalCost
			}

			mark := "✗"
			if qr.IsCorrect {
				mark = "✓"
			}
			log.Printf("[%d/%d] %s  got=%-2s want=%-2s  %-10s  %.2fs  $%.5f",
				i+1, len(rows), mark, predicted, correctLetter,
				resp.ModelUsed, qr.LatencyS, qr.CostUSD)
			if !qr.IsCorrect {
				log.Printf("         raw response: %q", truncate(resp.Response, 200))
			}
		}

		results = append(results, qr)

		if delay > 0 && i < len(rows)-1 {
			time.Sleep(delay)
		}
	}

	// 3. Compute and display metrics
	m := computeMetrics(results)
	printSummary(m)

	// 4. Save to JSON
	output := map[string]any{
		"run_at":  time.Now().Format(time.RFC3339),
		"config":  map[string]any{"url": *baseURL, "n": *n},
		"metrics": m,
		"results": results,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	if err := os.WriteFile(*outFile, data, 0644); err != nil {
		log.Printf("Warning: could not write output: %v", err)
	} else {
		log.Printf("Full results saved to %s", *outFile)
	}
}
