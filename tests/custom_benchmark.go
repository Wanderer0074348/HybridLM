package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---------- data types ----------

type datasetEntry struct {
	ID            int    `json:"id"`
	Query         string `json:"query"`
	ExpectedRoute string `json:"expected_route"` // "slm" or "llm"
	Category      string `json:"category"`
	Difficulty    string `json:"difficulty"`
}

type inferenceRequest struct {
	Query string `json:"query"`
}

type costMetrics struct {
	TotalCost float64 `json:"total_cost"`
}

type inferenceResponse struct {
	Response      string       `json:"response"`
	ModelUsed     string       `json:"model_used"` // "edge-slm" or "cloud-llm"
	RoutingReason string       `json:"routing_reason"`
	Latency       int64        `json:"latency"` // nanoseconds
	CacheHit      bool         `json:"cache_hit"`
	CostMetrics   *costMetrics `json:"cost_metrics"`
}

// ---------- result types ----------

type singleResult struct {
	ID            int     `json:"id"`
	Query         string  `json:"query"`
	ExpectedRoute string  `json:"expected_route"`
	ActualRoute   string  `json:"actual_route"`
	Correct       bool    `json:"correct"`
	Category      string  `json:"category"`
	Difficulty    string  `json:"difficulty"`
	LatencySec    float64 `json:"latency_sec"`
	Cost          float64 `json:"cost"`
	CacheHit      bool    `json:"cache_hit"`
	RoutingReason string  `json:"routing_reason"`
	Error         string  `json:"error,omitempty"`
}

type categoryStats struct {
	Total   int     `json:"total"`
	Correct int     `json:"correct"`
	Accuracy float64 `json:"accuracy_pct"`
}

type benchmarkSummary struct {
	TotalQueries        int                       `json:"total_queries"`
	CorrectRoutings     int                       `json:"correct_routings"`
	RoutingAccuracyPct  float64                   `json:"routing_accuracy_pct"`
	SimpleTotal         int                       `json:"simple_total"`
	SimpleCorrect       int                       `json:"simple_correct"`
	SimpleAccuracyPct   float64                   `json:"simple_accuracy_pct"`
	HardTotal           int                       `json:"hard_total"`
	HardCorrect         int                       `json:"hard_correct"`
	HardAccuracyPct     float64                   `json:"hard_accuracy_pct"`
	TotalCost           float64                   `json:"total_cost"`
	AvgLatencySec       float64                   `json:"avg_latency_sec"`
	TruePositives       int                       `json:"true_positives"`
	TrueNegatives       int                       `json:"true_negatives"`
	FalsePositives      int                       `json:"false_positives"`
	FalseNegatives      int                       `json:"false_negatives"`
	Precision           float64                   `json:"precision"`
	Recall              float64                   `json:"recall"`
	F1Score             float64                   `json:"f1_score"`
	CategoryBreakdown   map[string]*categoryStats `json:"category_breakdown"`
	Results             []singleResult            `json:"results"`
}

// ---------- helpers ----------

func routeMatches(expected, actual string) bool {
	isSlm := actual == "edge-slm" ||
		strings.Contains(actual, "llama") ||
		strings.Contains(actual, "mixtral") ||
		strings.Contains(actual, "gemma") ||
		strings.Contains(actual, "groq")
	isLlm := actual == "cloud-llm" ||
		strings.Contains(actual, "gpt") ||
		strings.Contains(actual, "claude") ||
		strings.Contains(actual, "gemini")
	switch expected {
	case "slm":
		return isSlm
	case "llm":
		return isLlm
	}
	return false
}

func truncateQuery(q string, max int) string {
	q = strings.ReplaceAll(q, "\n", " ")
	if len(q) > max {
		return q[:max] + "..."
	}
	return q
}

func safeDiv(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

// ---------- main ----------

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Base URL of the inference server")
	delay := flag.Duration("delay", 200*time.Millisecond, "Delay between requests")
	outFile := flag.String("out", "tests/results_custom.json", "Path to write JSON results")
	dataFile := flag.String("data", "tests/custom_dataset.json", "Path to dataset JSON file")
	flag.Parse()

	inferenceURL := *baseURL + "/api/v1/inference"

	// Load dataset
	raw, err := os.ReadFile(*dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot read dataset %s: %v\n", *dataFile, err)
		os.Exit(1)
	}

	var dataset []datasetEntry
	if err := json.Unmarshal(raw, &dataset); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot parse dataset: %v\n", err)
		os.Exit(1)
	}

	// Shuffle dataset so order doesn't bias the run
	rand.Shuffle(len(dataset), func(i, j int) { dataset[i], dataset[j] = dataset[j], dataset[i] })

	total := len(dataset)
	fmt.Printf("Loaded %d queries from %s (shuffled)\n", total, *dataFile)
	fmt.Printf("Target: %s\n", inferenceURL)
	fmt.Println(strings.Repeat("-", 90))

	client := &http.Client{Timeout: 60 * time.Second}

	results := make([]singleResult, 0, total)
	categoryMap := make(map[string]*categoryStats)

	var (
		correct       int
		simpleTotal   int
		simpleCorrect int
		hardTotal     int
		hardCorrect   int
		totalCost     float64
		totalLatency  float64
		tp, tn, fp, fn int
	)

	for i, entry := range dataset {
		// Ensure category bucket exists
		if _, ok := categoryMap[entry.Category]; !ok {
			categoryMap[entry.Category] = &categoryStats{}
		}
		categoryMap[entry.Category].Total++

		// Build request
		reqBody, _ := json.Marshal(inferenceRequest{Query: entry.Query})

		res := singleResult{
			ID:            entry.ID,
			Query:         entry.Query,
			ExpectedRoute: entry.ExpectedRoute,
			Category:      entry.Category,
			Difficulty:    entry.Difficulty,
		}

		// Retry loop with backoff for rate limiting
		var ir inferenceResponse
		var callErr error
		for attempt := 1; attempt <= 8; attempt++ {
			wallStart := time.Now()
			resp, httpErr := client.Post(inferenceURL, "application/json", bytes.NewReader(reqBody))
			res.LatencySec = time.Since(wallStart).Seconds()

			if httpErr != nil {
				callErr = httpErr
				wait := time.Duration(attempt*attempt) * time.Second
				fmt.Printf("[%d/%d] attempt %d: connection error, retrying in %v: %v\n",
					i+1, total, attempt, wait, httpErr)
				time.Sleep(wait)
				reqBody, _ = json.Marshal(inferenceRequest{Query: entry.Query}) // fresh body
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Rate limit or server overload — back off and retry
			if resp.StatusCode == 429 || resp.StatusCode == 503 || resp.StatusCode == 502 {
				wait := time.Duration(attempt*attempt) * time.Second
				fmt.Printf("[%d/%d] attempt %d: HTTP %d (rate limit/overload), waiting %v...\n",
					i+1, total, attempt, resp.StatusCode, wait)
				time.Sleep(wait)
				reqBody, _ = json.Marshal(inferenceRequest{Query: entry.Query})
				continue
			}

			if resp.StatusCode != 200 {
				callErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
				break
			}

			if jsonErr := json.Unmarshal(body, &ir); jsonErr != nil {
				callErr = fmt.Errorf("json decode: %v", jsonErr)
				break
			}

			callErr = nil
			break
		}

		if callErr != nil {
			res.Error = callErr.Error()
			res.ActualRoute = "error"
			fmt.Printf("[%d/%d] ERROR  expected=%-3s  %.2fs  %q\n   err: %v\n",
				i+1, total, entry.ExpectedRoute, res.LatencySec, truncateQuery(entry.Query, 60), callErr)
		} else {
			res.ActualRoute = ir.ModelUsed
			res.CacheHit = ir.CacheHit
			res.RoutingReason = ir.RoutingReason

			if ir.Latency > 0 {
				res.LatencySec = float64(ir.Latency) / float64(time.Second)
			}
			if ir.CostMetrics != nil {
				res.Cost = ir.CostMetrics.TotalCost
			}
			res.Correct = routeMatches(entry.ExpectedRoute, ir.ModelUsed)
		}

		// Confusion matrix (error counts as wrong route)
		expectedLLM := entry.ExpectedRoute == "llm"
		gotLLM := res.ActualRoute == "cloud-llm" ||
			strings.Contains(res.ActualRoute, "gpt") ||
			strings.Contains(res.ActualRoute, "claude") ||
			strings.Contains(res.ActualRoute, "gemini")
		switch {
		case expectedLLM && gotLLM:
			tp++
		case !expectedLLM && !gotLLM:
			tn++
		case !expectedLLM && gotLLM:
			fp++
		case expectedLLM && !gotLLM:
			fn++
		}

		mark := "✗"
		if res.Correct {
			mark = "✓"
			correct++
			categoryMap[entry.Category].Correct++
		}

		// Per-difficulty counters
		if entry.Difficulty == "simple" {
			simpleTotal++
			if res.Correct {
				simpleCorrect++
			}
		} else {
			hardTotal++
			if res.Correct {
				hardCorrect++
			}
		}

		totalCost += res.Cost
		totalLatency += res.LatencySec

		fmt.Printf("[%d/%d] %s  route=%-10s expected=%-3s  %.2fs  $%.5f  %q\n",
			i+1, total, mark,
			res.ActualRoute, entry.ExpectedRoute,
			res.LatencySec, res.Cost,
			truncateQuery(entry.Query, 60))

		results = append(results, res)

		if i < total-1 && *delay > 0 {
			time.Sleep(*delay)
		}
	}

	// Compute derived stats
	precision := safeDiv(float64(tp), float64(tp+fp))
	recall := safeDiv(float64(tp), float64(tp+fn))
	f1 := safeDiv(2*precision*recall, precision+recall)
	if math.IsNaN(f1) {
		f1 = 0
	}

	for _, cs := range categoryMap {
		cs.Accuracy = safeDiv(float64(cs.Correct), float64(cs.Total)) * 100
	}

	summary := benchmarkSummary{
		TotalQueries:       total,
		CorrectRoutings:    correct,
		RoutingAccuracyPct: safeDiv(float64(correct), float64(total)) * 100,
		SimpleTotal:        simpleTotal,
		SimpleCorrect:      simpleCorrect,
		SimpleAccuracyPct:  safeDiv(float64(simpleCorrect), float64(simpleTotal)) * 100,
		HardTotal:          hardTotal,
		HardCorrect:        hardCorrect,
		HardAccuracyPct:    safeDiv(float64(hardCorrect), float64(hardTotal)) * 100,
		TotalCost:          totalCost,
		AvgLatencySec:      safeDiv(totalLatency, float64(total)),
		TruePositives:      tp,
		TrueNegatives:      tn,
		FalsePositives:     fp,
		FalseNegatives:     fn,
		Precision:          precision,
		Recall:             recall,
		F1Score:            f1,
		CategoryBreakdown:  categoryMap,
		Results:            results,
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 90))
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 90))
	fmt.Printf("Total queries        : %d\n", total)
	fmt.Printf("Routing accuracy     : %.2f%%  (%d/%d correct)\n",
		summary.RoutingAccuracyPct, correct, total)
	fmt.Printf("Simple query acc.    : %.2f%%  (%d/%d SLM queries → edge-slm)\n",
		summary.SimpleAccuracyPct, simpleCorrect, simpleTotal)
	fmt.Printf("Hard query acc.      : %.2f%%  (%d/%d LLM queries → cloud-llm)\n",
		summary.HardAccuracyPct, hardCorrect, hardTotal)
	fmt.Printf("Total cost           : $%.5f\n", totalCost)
	fmt.Printf("Avg latency          : %.3fs\n", summary.AvgLatencySec)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Confusion matrix     : TP=%d  TN=%d  FP=%d  FN=%d\n", tp, tn, fp, fn)
	fmt.Printf("Precision            : %.4f\n", precision)
	fmt.Printf("Recall               : %.4f\n", recall)
	fmt.Printf("F1 Score             : %.4f\n", f1)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Category breakdown:")
	for cat, cs := range categoryMap {
		fmt.Printf("  %-20s  %d/%d  (%.1f%%)\n", cat, cs.Correct, cs.Total, cs.Accuracy)
	}
	fmt.Println(strings.Repeat("=", 90))

	// Save results
	outJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot marshal results: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outFile, outJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot write %s: %v\n", *outFile, err)
		os.Exit(1)
	}

	fmt.Printf("\nResults saved to %s\n", *outFile)
}
