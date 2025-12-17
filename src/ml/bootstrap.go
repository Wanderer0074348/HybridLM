package ml

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/features"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type BootstrapDataGenerator struct {
	routingRepo      *repository.RoutingRepository
	featureExtractor *features.FeatureExtractor
}

func NewBootstrapDataGenerator(
	routingRepo *repository.RoutingRepository,
	featureExtractor *features.FeatureExtractor,
) *BootstrapDataGenerator {
	return &BootstrapDataGenerator{
		routingRepo:      routingRepo,
		featureExtractor: featureExtractor,
	}
}

func (b *BootstrapDataGenerator) GenerateBootstrapData(ctx context.Context) error {
	examples := []struct {
		query      string
		shouldUseLLM bool
		rating     int
	}{
		{"Why does the sky appear blue?", true, 5},
		{"Explain quantum entanglement", true, 5},
		{"How does photosynthesis work?", true, 5},
		{"Compare democracy and autocracy", true, 5},
		{"Prove that the square root of 2 is irrational", true, 5},
		{"Debug this Python code: def foo(): return bar", true, 5},
		{"What is the capital of France?", false, 4},
		{"Hi", false, 3},
		{"Define photosynthesis", false, 4},
		{"What time is it?", false, 3},
		{"Who is the president?", false, 4},
		{"Thanks", false, 3},
		{"Calculate 15 + 27", true, 5},
		{"Solve for x: 2x + 5 = 15", true, 5},
		{"Derive the quadratic formula", true, 5},
		{"What is 2+2?", false, 4},
		{"Analyze the themes in Shakespeare's Hamlet", true, 5},
		{"Critique the argument that AI will replace humans", true, 5},
		{"Synthesize information about climate change causes", true, 5},
		{"Optimize this algorithm for better performance", true, 5},
		{"Implement a binary search tree in Python", true, 5},
		{"Refactor this code to use design patterns", true, 5},
		{"What is a variable?", false, 4},
		{"Define recursion", false, 4},
		{"List Python data types", false, 4},
		{"Evaluate the pros and cons of renewable energy", true, 5},
		{"Justify your answer with reasoning", true, 5},
		{"Demonstrate how neural networks learn", true, 5},
		{"What is machine learning?", false, 4},
		{"Define artificial intelligence", false, 4},
		{"How do I install Python?", false, 4},
		{"Explain step-by-step how to build a REST API", true, 5},
		{"What's the difference between supervised and unsupervised learning?", true, 5},
		{"Contrast procedural and object-oriented programming", true, 5},
		{"Deduce the next number in the sequence: 2, 4, 8, 16", true, 5},
		{"What is the Fibonacci sequence?", false, 4},
		{"Argue for or against universal basic income", true, 5},
		{"Discuss the ethical implications of genetic engineering", true, 5},
		{"Assess the impact of social media on mental health", true, 5},
		{"What is social media?", false, 3},
		{"Elaborate on the Big Bang theory", true, 5},
		{"Describe in detail how vaccines work", true, 5},
		{"What is a vaccine?", false, 4},
		{"If A implies B and B implies C, what can we conclude?", true, 5},
		{"Given that all humans are mortal and Socrates is human, is Socrates mortal?", true, 5},
		{"What is logic?", false, 4},
		{"Fix this bug: undefined variable error", true, 5},
		{"What is a bug?", false, 3},
		{"Compute the derivative of x^2 + 3x + 2", true, 5},
		{"What is a derivative?", false, 4},
		{"Integrate sin(x) from 0 to pi", true, 5},
		{"What is integration?", false, 4},
		{"Prove Pythagoras' theorem", true, 5},
		{"What is Pythagoras' theorem?", false, 4},
		{"Explain with examples how recursion works", true, 5},
		{"What does API stand for?", false, 3},
		{"How does a compiler differ from an interpreter?", true, 5},
		{"What is a compiler?", false, 4},
		{"Analyze the time complexity of bubble sort", true, 5},
		{"What is time complexity?", false, 4},
		{"Design a database schema for an e-commerce system", true, 5},
		{"What is a database?", false, 3},
		{"Evaluate whether this code follows SOLID principles", true, 5},
		{"What are SOLID principles?", false, 4},
		{"Reason through this problem: You have 3 boxes", true, 5},
		{"What is reasoning?", false, 3},
		{"Compare TCP and UDP protocols", true, 5},
		{"What is TCP?", false, 4},
		{"Describe the MVC architecture pattern", true, 5},
		{"What is MVC?", false, 3},
		{"Explain how blockchain ensures security", true, 5},
		{"What is blockchain?", false, 4},
		{"Analyze the differences between NoSQL and SQL databases", true, 5},
		{"What is NoSQL?", false, 4},
		{"Develop a strategy for optimizing database queries", true, 5},
		{"What is a query?", false, 3},
		{"Investigate why this function returns unexpected results", true, 5},
		{"What is a function?", false, 3},
		{"Synthesize the key concepts of object-oriented programming", true, 5},
		{"List OOP concepts", false, 4},
		{"Examine the trade-offs between microservices and monoliths", true, 5},
		{"What is a microservice?", false, 4},
		{"Break down how JWT authentication works", true, 5},
		{"What is JWT?", false, 3},
		{"Consider the implications of using global variables", true, 5},
		{"What is a global variable?", false, 3},
		{"Determine the best approach for caching strategies", true, 5},
		{"What is caching?", false, 3},
		{"Interpret this error message and suggest fixes", true, 5},
		{"What is an error?", false, 3},
		{"Formulate a hypothesis about why latency increased", true, 5},
		{"What is latency?", false, 3},
		{"Construct a proof by induction", true, 5},
		{"What is induction?", false, 4},
		{"Validate this regular expression", true, 5},
		{"What is regex?", false, 3},
		{"Transform this callback hell into async/await", true, 5},
		{"What is async/await?", false, 4},
		{"Clarify the difference between abstract classes and interfaces", true, 5},
		{"What is an interface?", false, 4},
		{"Illustrate how garbage collection works", true, 5},
		{"What is garbage collection?", false, 4},
		{"Survey the landscape of modern frontend frameworks", true, 5},
		{"Name a frontend framework", false, 3},
		{"Review this code for security vulnerabilities", true, 5},
		{"What is security?", false, 3},
	}

	for _, ex := range examples {
		features := b.featureExtractor.Extract(ex.query, "")

		hash := md5.Sum([]byte(ex.query))
		queryHash := hex.EncodeToString(hash[:])

		decision := "slm"
		if ex.shouldUseLLM {
			decision = "llm"
		}

		record := &models.RoutingDecisionRecord{
			Query:     ex.query,
			QueryHash: queryHash,
			Features:  features,
			Routing: models.RoutingMetadata{
				Decision:    decision,
				Reason:      "Bootstrap training data",
				Confidence:  1.0,
				Strategy:    "bootstrap",
				ABTestGroup: "bootstrap",
			},
			Performance: models.PerformanceMetrics{
				LatencyMs: 100,
				CostUSD:   0.0001,
				CacheHit:  false,
				ModelUsed: "bootstrap",
			},
			Feedback: &models.UserFeedback{
				Rating:  &ex.rating,
				Thumbs:  "up",
				Comment: "Bootstrap data",
			},
			UserID:    "system",
			SessionID: "bootstrap",
			Timestamp: time.Now(),
		}

		err := b.routingRepo.SaveDecision(ctx, record)
		if err != nil {
			return err
		}
	}

	return nil
}
