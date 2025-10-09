package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
	LLM    LLMConfig
	SLM    SLMConfig
	Router RouterConfig
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

type LLMConfig struct {
	Endpoint  string        `mapstructure:"endpoint"`
	APIKey    string        `mapstructure:"api_key"`
	Model     string        `mapstructure:"model"`
	MaxTokens int           `mapstructure:"max_tokens"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type SLMConfig struct {
	OllamaHost    string        `mapstructure:"ollama_host"`
	ModelName     string        `mapstructure:"model_name"`
	MaxConcurrent int           `mapstructure:"max_concurrent"`
	MaxTokens     int           `mapstructure:"max_tokens"`
	Timeout       time.Duration `mapstructure:"timeout"`
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
	viper.BindEnv("redis.password", "REDIS_PASSWORD")

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

	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		config.Redis.Password = redisPass
	}

	// Validate required fields
	if config.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY environment variable is required")
	}

	return &config, nil
}
