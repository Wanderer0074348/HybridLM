package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Redis         RedisConfig         `mapstructure:"redis"`
	SemanticCache SemanticCacheConfig `mapstructure:"semantic_cache"`
	LLM           LLMConfig           `mapstructure:"llm"`
	SLM           SLMConfig           `mapstructure:"slm"`
	Router        RouterConfig        `mapstructure:"router"`
}

type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type RedisConfig struct {
	Address  string        `mapstructure:"address"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

type SemanticCacheConfig struct {
	Enabled             bool    `mapstructure:"enabled"`
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
	APIKey              string  `mapstructure:"api_key"`
}

type LLMConfig struct {
	Endpoint  string        `mapstructure:"endpoint"`
	APIKey    string        `mapstructure:"api_key"`
	Model     string        `mapstructure:"model"`
	MaxTokens int           `mapstructure:"max_tokens"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type SLMModelConfig struct {
	Name     string  `mapstructure:"name"`
	Endpoint string  `mapstructure:"endpoint"`
	APIKey   string  `mapstructure:"api_key"`
	Weight   float64 `mapstructure:"weight"` // For weighted voting in parallel mode
}

type SLMConfig struct {
	Models         []SLMModelConfig `mapstructure:"models"`
	Strategy       string           `mapstructure:"strategy"` // "parallel", "series", "hybrid"
	MaxConcurrent  int              `mapstructure:"max_concurrent"`
	MaxTokens      int              `mapstructure:"max_tokens"`
	Timeout        time.Duration    `mapstructure:"timeout"`
	AggregationFn  string           `mapstructure:"aggregation_fn"` // "voting", "longest", "weighted"
	ChainThreshold float64          `mapstructure:"chain_threshold"` // Confidence threshold for chaining
}

type RouterConfig struct {
	ComplexityThreshold float64 `mapstructure:"complexity_threshold"`
	LatencyBudgetMs     int     `mapstructure:"latency_budget_ms"`
	CostThresholdUSD    float64 `mapstructure:"cost_threshold_usd"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Enable environment variable override
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("llm.api_key", "LLM_API_KEY")
	viper.BindEnv("semantic_cache.api_key", "SEMANTIC_CACHE_API_KEY")

	// Read config file (optional if not present)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Override with environment variables explicitly
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}

	// Parse REDIS_URL if provided (Render/Heroku format)
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		if err := parseRedisURL(redisURL, &config.Redis); err != nil {
			return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
	}

	// Individual Redis env vars override REDIS_URL
	if redisAddr := os.Getenv("REDIS_ADDRESS"); redisAddr != "" {
		config.Redis.Address = redisAddr
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		config.Redis.Password = redisPass
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			config.Redis.DB = db
		}
	}

	// Override API keys for all SLM models from GROQ_API_KEY
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		for i := range config.SLM.Models {
			config.SLM.Models[i].APIKey = groqKey
		}
	}

	// Override semantic cache API key from environment
	// If not set, default to using the same key as LLM
	if semanticCacheKey := os.Getenv("SEMANTIC_CACHE_API_KEY"); semanticCacheKey != "" {
		config.SemanticCache.APIKey = semanticCacheKey
	} else {
		config.SemanticCache.APIKey = config.LLM.APIKey
	}

	// Validate required fields
	if config.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY environment variable is required")
	}

	return &config, nil
}

// parseRedisURL parses a Redis connection URL (redis://user:password@host:port/db)
// and populates the RedisConfig struct
func parseRedisURL(redisURL string, cfg *RedisConfig) error {
	u, err := url.Parse(redisURL)
	if err != nil {
		return fmt.Errorf("invalid Redis URL format: %w", err)
	}

	// Extract host and port
	cfg.Address = u.Host

	// Extract password from URL
	if u.User != nil {
		if password, ok := u.User.Password(); ok {
			cfg.Password = password
		}
	}

	// Extract database number from path (e.g., /0, /1)
	if u.Path != "" && u.Path != "/" {
		dbStr := u.Path[1:] // Remove leading slash
		if db, err := strconv.Atoi(dbStr); err == nil {
			cfg.DB = db
		}
	}

	return nil
}
