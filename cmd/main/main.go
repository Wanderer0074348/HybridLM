package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("✓ Config loaded successfully")

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisCache.Close()
	log.Printf("✓ Redis connected")

	// Initialize SLM engine (Ollama with langchaingo)
	slmEngine, err := inference.NewSLMEngine(&cfg.SLM)
	if err != nil {
		log.Fatalf("Failed to initialize SLM engine: %v", err)
	}
	defer slmEngine.Close()
	log.Printf("✓ Ollama SLM engine ready: %s", cfg.SLM.ModelName)

	// Initialize LLM client (OpenAI with langchaingo)
	llmClient, err := inference.NewLLMClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
	}
	log.Printf("✓ OpenAI LLM client ready: %s", cfg.LLM.Model)

	// Initialize query router
	queryRouter := router.NewQueryRouter(&cfg.Router)
	log.Printf("✓ Query router initialized")

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Initialize handlers
	inferenceHandler := handlers.NewInferenceHandler(
		queryRouter,
		slmEngine,
		llmClient,
		redisCache,
	)

	// Routes
	v1 := r.Group("/api/v1")
	{
		v1.POST("/inference", inferenceHandler.HandleInference)
		v1.GET("/health", inferenceHandler.HealthCheck)
	}

	// Server setup
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
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
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
