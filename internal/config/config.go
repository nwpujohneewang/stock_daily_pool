package config

import (
	"fmt"
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

// RedisConfig is Redis configuration
type RedisConfig struct {
	Addr         string `yaml:"addr"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	DialTimeout  int    `yaml:"dial_timeout"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// TushareConfig is Tushare API configuration
type TushareConfig struct {
	BaseURL         string `yaml:"base_url"`
	Token           string `yaml:"token"`
	RateLimitPerMin int    `yaml:"rate_limit_per_min"`
	Timeout         int    `yaml:"timeout"`
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
	Provider            string  `yaml:"provider"`
	Model               string  `yaml:"model"`
	APIKey              string  `yaml:"api_key"`
	APIURL              string  `yaml:"api_url"`
	Temperature         float64 `yaml:"temperature"`
	MaxTokens           int     `yaml:"max_tokens"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
	MaxConcurrent       int     `yaml:"max_concurrent"`
	Timeout             int     `yaml:"timeout"`
}

// CircuitBreakerConfig is circuit breaker configuration
type CircuitBreakerConfig struct {
	ConsecutiveFailures int `yaml:"consecutive_failures"`
	CooldownSec         int `yaml:"cooldown_sec"`
	HalfOpenRequests    int `yaml:"half_open_requests"`
}

// MonitorConfig is monitor configuration
type MonitorConfig struct {
	IntervalSec       int                  `yaml:"interval_sec"`
	TradingStart      string               `yaml:"trading_start"`
	TradingEnd        string               `yaml:"trading_end"`
	StrongThreshold   float64              `yaml:"strong_threshold"`
	ShardCount        int                  `yaml:"shard_count"`
	ErrorThresholdPct int                  `yaml:"error_threshold_pct"`
	CircuitBreaker    CircuitBreakerConfig `yaml:"circuit_breaker"`
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
	PreMarketInit   string `yaml:"pre_market_init"`
	RealtimeCollect string `yaml:"realtime_collect"`
	JiuyanSync      string `yaml:"jiuyan_sync"`
	ConceptSync     string `yaml:"concept_sync"`
	ClosingSnapshot string `yaml:"closing_snapshot"`
	HistoryCleanup  string `yaml:"history_cleanup"`
	CacheWarmup     string `yaml:"cache_warmup"`
	LLMBatch        string `yaml:"llm_batch"`
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
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// DSN returns a pgx-compatible connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Timezone,
	)
}

var GlobalConfig *AppConfig

// Init loads configuration from config.yaml into GlobalConfig.
func Init(cfgPath string) error {
	viper.SetConfigFile(cfgPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	GlobalConfig = &AppConfig{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	return nil
}
