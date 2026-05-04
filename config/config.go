package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig is the root configuration structure
type AppConfig struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Tushare   TushareConfig   `yaml:"tushare"`
	Jiuyan    JiuyanConfig    `yaml:"jiuyan"`
	LLM       LLMConfig       `yaml:"llm"`
	Monitor   MonitorConfig   `yaml:"monitor"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Retry     RetryConfig     `yaml:"retry"`
	Log       LogConfig       `yaml:"log"`
}

// ServerConfig is HTTP server configuration
type ServerConfig struct {
	Port   int    `yaml:"port"`
	Mode   string `yaml:"mode"`
	APIKey string `yaml:"api_key"`
}

// DatabaseConfig is PostgreSQL configuration
type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"dbname"`
	SSLMode         string `yaml:"sslmode"`
	Timezone        string `yaml:"timezone"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// TushareConfig is Tushare API configuration
type TushareConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
	Timeout int    `yaml:"timeout"`
}

// JiuyanConfig is Jiuyan Gongshe API configuration
type JiuyanConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	Timeout        int    `yaml:"timeout"`
	CrawlStartDate string `yaml:"crawl_start_date"`
}

// LLMConfig is LLM configuration
type LLMConfig struct {
	Provider            string  `yaml:"provider" mapstructure:"provider"`
	Model               string  `yaml:"model" mapstructure:"model"`
	APIKey              string  `yaml:"api_key" mapstructure:"api_key"`
	APIURL              string  `yaml:"api_url" mapstructure:"api_url"`
	MaxTokens           int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold" mapstructure:"confidence_threshold"`
	MaxConcurrent       int     `yaml:"max_concurrent" mapstructure:"max_concurrent"`
	Timeout             int     `yaml:"timeout" mapstructure:"timeout"`
}

// CircuitBreakerConfig is circuit breaker configuration
type CircuitBreakerConfig struct {
	ConsecutiveFailures int `yaml:"consecutive_failures"`
	CooldownSec         int `yaml:"cooldown_sec"`
	HalfOpenRequests    int `yaml:"half_open_requests"`
}

// MonitorConfig is monitor configuration
type MonitorConfig struct {
	IntervalSec       int                  `yaml:"interval_sec" mapstructure:"interval_sec"`
	TradingStart      string               `yaml:"trading_start" mapstructure:"trading_start"`
	TradingEnd        string               `yaml:"trading_end" mapstructure:"trading_end"`
	StrongThreshold   float64              `yaml:"strong_threshold" mapstructure:"strong_threshold"`
	ErrorThresholdPct int                  `yaml:"error_threshold_pct" mapstructure:"error_threshold_pct"`
	CircuitBreaker    CircuitBreakerConfig `yaml:"circuit_breaker" mapstructure:"circuit_breaker"`
}

// WebSocketConfig is WebSocket configuration
type WebSocketConfig struct {
	PingInterval    int `yaml:"ping_interval"`
	PongTimeout     int `yaml:"pong_timeout"`
	WriteBufferSize int `yaml:"write_buffer_size"`
	MaxConnections  int `yaml:"max_connections"`
	WriteWait       int `yaml:"write_wait"`
	MaxMessageSize  int `yaml:"max_message_size"`
}

// SchedulerConfig is scheduler configuration
type SchedulerConfig struct {
	PreMarketInit      string `yaml:"pre_market_init" mapstructure:"pre_market_init"`
	RealtimeCollect    string `yaml:"realtime_collect" mapstructure:"realtime_collect"`
	JiuyanSync         string `yaml:"jiuyan_sync" mapstructure:"jiuyan_sync"`
	ConceptSync        string `yaml:"concept_sync" mapstructure:"concept_sync"`
	ClosingSnapshot    string `yaml:"closing_snapshot" mapstructure:"closing_snapshot"`
	HistoryCleanup     string `yaml:"history_cleanup" mapstructure:"history_cleanup"`
	CacheWarmup        string `yaml:"cache_warmup" mapstructure:"cache_warmup"`
	LLMBatch           string `yaml:"llm_batch" mapstructure:"llm_batch"`
	LimitDetailRefresh string `yaml:"limit_detail_refresh" mapstructure:"limit_detail_refresh"`
}

// RetryConfig is retry configuration
type RetryConfig struct {
	MaxRetries     int     `yaml:"max_retries"`
	InitialDelayMs int     `yaml:"initial_delay_ms"`
	Multiplier     float64 `yaml:"multiplier"`
	MaxDelayMs     int     `yaml:"max_delay_ms"`
}

func (c *RetryConfig) InitialDelay() time.Duration {
	return time.Duration(c.InitialDelayMs) * time.Millisecond
}

func (c *RetryConfig) MaxDelay() time.Duration {
	return time.Duration(c.MaxDelayMs) * time.Millisecond
}

// LogConfig is log configuration
type LogConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
	Output string `yaml:"output" mapstructure:"output"`
}

// DSN returns a pgx-compatible connection string.
func (c *DatabaseConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s TimeZone=%s search_path=public",
		c.Host, c.Port, c.User, c.DBName, c.SSLMode, c.Timezone,
	)
	if c.Password != "" {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s search_path=public",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Timezone,
		)
	}
	return dsn
}

var GlobalConfig *AppConfig

// Init loads configuration from config.yaml into GlobalConfig.
func Init(cfgPath string) error {
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	GlobalConfig = cfg
	return nil
}

func loadConfig(cfgPath string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(resolveConfigPath(cfgPath))
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	bindEnvOverrides(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &AppConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}

func resolveConfigPath(cfgPath string) string {
	if _, err := os.Stat(cfgPath); err == nil {
		return cfgPath
	}
	ext := filepath.Ext(cfgPath)
	if ext == "" {
		return cfgPath
	}
	base := strings.TrimSuffix(cfgPath, ext)
	fallback := base + ".example" + ext
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return cfgPath
}

func bindEnvOverrides(v *viper.Viper) {
	envMap := map[string]string{
		"server.port":              "SERVER_PORT",
		"server.mode":              "SERVER_MODE",
		"server.apiKey":            "SERVER_API_KEY",
		"database.host":            "DATABASE_HOST",
		"database.port":            "DATABASE_PORT",
		"database.user":            "DATABASE_USER",
		"database.password":        "DATABASE_PASSWORD",
		"database.dbname":          "DATABASE_DBNAME",
		"database.sslmode":         "DATABASE_SSLMODE",
		"database.timezone":        "DATABASE_TIMEZONE",
		"redis.addr":               "REDIS_ADDR",
		"redis.password":           "REDIS_PASSWORD",
		"redis.db":                 "REDIS_DB",
		"tushare.baseUrl":          "TUSHARE_BASE_URL",
		"tushare.token":            "TUSHARE_TOKEN",
		"jiuyan.baseUrl":           "JIUYAN_BASE_URL",
		"jiuyan.apiKey":            "JIUYAN_API_KEY",
		"llm.api_key":              "LLM_API_KEY",
		"llm.api_url":              "LLM_API_URL",
		"llm.provider":             "LLM_PROVIDER",
		"llm.model":                "LLM_MODEL",
		"llm.max_tokens":           "LLM_MAX_TOKENS",
		"llm.timeout":              "LLM_TIMEOUT",
		"llm.temperature":          "LLM_TEMPERATURE",
		"llm.confidence_threshold": "LLM_CONFIDENCE_THRESHOLD",
		"llm.max_concurrent":       "LLM_MAX_CONCURRENT",
		"log.level":                "LOG_LEVEL",
		"log.format":               "LOG_FORMAT",
		"log.output":               "LOG_OUTPUT",
	}
	for key, env := range envMap {
		_ = v.BindEnv(key, env)
	}
}
