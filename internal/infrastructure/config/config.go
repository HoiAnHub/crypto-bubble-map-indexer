package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App     AppConfig     `mapstructure:"app"`
	NATS    NATSConfig    `mapstructure:"nats"`
	Neo4J   Neo4JConfig   `mapstructure:"neo4j"`
	Health  HealthConfig  `mapstructure:"health"`
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// AppConfig represents application-specific configuration
type AppConfig struct {
	Env            string `mapstructure:"env"`
	LogLevel       string `mapstructure:"log_level"`
	HTTPPort       int    `mapstructure:"http_port"`
	WorkerPoolSize int    `mapstructure:"worker_pool_size"`
	BatchSize      int    `mapstructure:"batch_size"`
}

// NATSConfig represents NATS configuration
type NATSConfig struct {
	URL                string        `mapstructure:"url"`
	StreamName         string        `mapstructure:"stream_name"`
	SubjectPrefix      string        `mapstructure:"subject_prefix"`
	ConsumerGroup      string        `mapstructure:"consumer_group"`
	ConnectTimeout     time.Duration `mapstructure:"connect_timeout"`
	ReconnectAttempts  int           `mapstructure:"reconnect_attempts"`
	ReconnectDelay     time.Duration `mapstructure:"reconnect_delay"`
	MaxPendingMessages int           `mapstructure:"max_pending_messages"`
	Enabled            bool          `mapstructure:"enabled"`
}

// Neo4JConfig represents Neo4J configuration
type Neo4JConfig struct {
	URI                          string        `mapstructure:"uri"`
	Username                     string        `mapstructure:"username"`
	Password                     string        `mapstructure:"password"`
	Database                     string        `mapstructure:"database"`
	ConnectTimeout               time.Duration `mapstructure:"connect_timeout"`
	MaxConnectionPoolSize        int           `mapstructure:"max_connection_pool_size"`
	ConnectionAcquisitionTimeout time.Duration `mapstructure:"connection_acquisition_timeout"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Interval time.Duration `mapstructure:"interval"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// Load loads configuration from environment variables and files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/crypto-bubble-map-indexer")

	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	// Map environment variables to nested config keys
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Default values
	setDefaults()

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// App defaults
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.http_port", 8080)
	viper.SetDefault("app.worker_pool_size", 10)
	viper.SetDefault("app.batch_size", 100)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://ethereum-nats:4222")
	viper.SetDefault("nats.stream_name", "TRANSACTIONS")
	viper.SetDefault("nats.subject_prefix", "transactions")
	viper.SetDefault("nats.consumer_group", "bubble-map-indexer")
	viper.SetDefault("nats.connect_timeout", "10s")
	viper.SetDefault("nats.reconnect_attempts", 5)
	viper.SetDefault("nats.reconnect_delay", "2s")
	viper.SetDefault("nats.max_pending_messages", 10000)
	viper.SetDefault("nats.enabled", true)

	// Neo4J defaults
	viper.SetDefault("neo4j.uri", "neo4j://localhost:7687")
	viper.SetDefault("neo4j.username", "neo4j")
	viper.SetDefault("neo4j.password", "password")
	viper.SetDefault("neo4j.database", "neo4j")
	viper.SetDefault("neo4j.connect_timeout", "10s")
	viper.SetDefault("neo4j.max_connection_pool_size", 50)
	viper.SetDefault("neo4j.connection_acquisition_timeout", "60s")

	// Health defaults
	viper.SetDefault("health.interval", "30s")
	viper.SetDefault("health.timeout", "5s")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)

	// Bind env for NATS URL
	viper.BindEnv("nats.url", "NATS_URL")
}
