package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"www.github.com/Wanderer0074348/HybridLM/src/cache"
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/handlers"
	"www.github.com/Wanderer0074348/HybridLM/src/inference"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

func init() {

	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	} else {
		log.Println("✅ Loaded .env file")
	}
}

func main() {

	if os.Getenv("LLM_API_KEY") == "" {
		log.Fatal("❌ LLM_API_KEY not set in environment or .env file")
	}
	if os.Getenv("GROQ_API_KEY") == "" {
		log.Fatal("❌ GROQ_API_KEY not set in environment or .env file")
	}

	log.Println("✅ Environment variables loaded successfully")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("✓ Config loaded successfully")

	redisCache, err := cache.NewRedisCache(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisCache.Close()
	log.Printf("✓ Redis connected")

	slmEngine, err := inference.NewSLMEngine(&cfg.SLM)
	if err != nil {
		log.Fatalf("Failed to initialize SLM engine: %v", err)
	}
	defer slmEngine.Close()
	log.Printf("✓ SLM engine ready with %d models (%s strategy)", len(cfg.SLM.Models), cfg.SLM.Strategy)
	for _, model := range cfg.SLM.Models {
		log.Printf("  - %s (weight: %.1f)", model.Name, model.Weight)
	}

	llmClient, err := inference.NewLLMClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
	}
	log.Printf("✓ LLM client ready: %s", cfg.LLM.Model)

	queryRouter := router.NewQueryRouter(&cfg.Router)
	log.Printf("✓ Query router initialized")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	inferenceHandler := handlers.NewInferenceHandler(
		queryRouter,
		slmEngine,
		llmClient,
		redisCache,
	)

	if cfg.SemanticCache.Enabled {
		if cfg.SemanticCache.APIKey == "" {
			log.Println("⚠️  Semantic cache enabled but SEMANTIC_CACHE_API_KEY not set, using standard cache only")
		} else {
			semanticCache, err := cache.NewSemanticCache(&cfg.Redis, &cfg.SemanticCache)
			if err != nil {
				log.Printf("⚠️  Failed to initialize semantic cache: %v, falling back to standard cache", err)
			} else {
				inferenceHandler.SetSemanticCache(semanticCache, cfg.SemanticCache.SimilarityThreshold)
				log.Printf("✓ Semantic cache enabled (threshold: %.2f)", cfg.SemanticCache.SimilarityThreshold)
			}
		}
	} else {
		log.Println("ℹ️  Semantic cache disabled, using standard exact-match cache")
	}

	v1 := r.Group("/api/v1")
	{
		v1.POST("/inference", inferenceHandler.HandleInference)
		v1.GET("/health", inferenceHandler.HealthCheck)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	log.Printf("🚀 HybridLM Engine running on port %s", cfg.Server.Port)
	log.Printf("📊 Complexity threshold: %.2f", cfg.Router.ComplexityThreshold)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	// Get allowed origins from environment variable
	// Default to localhost for development if not set
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string

	if allowedOriginsEnv != "" {
		// Split by comma for multiple origins
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
		// Trim whitespace from each origin
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	} else {
		// Default for local development
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:3001",
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow requests without Origin header (e.g., health checks, curl, Postman, Render health checks)
		if origin == "" {
			c.Next()
			return
		}

		// Check if the origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If origin is not allowed, don't set CORS headers
		if !allowed {
			c.AbortWithStatus(403)
			return
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
