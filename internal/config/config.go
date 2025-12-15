// Package config provides configuration loading and management for powa-sentinel.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Schedule ScheduleConfig `yaml:"schedule"`
	Analysis AnalysisConfig `yaml:"analysis"`
	Rules    RulesConfig    `yaml:"rules"`
	Notifier NotifierConfig `yaml:"notifier"`
	Server   ServerConfig   `yaml:"server"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// ScheduleConfig defines when analysis jobs run.
type ScheduleConfig struct {
	Cron string `yaml:"cron"`
}

// AnalysisConfig defines analysis time windows.
type AnalysisConfig struct {
	WindowDuration   string `yaml:"window_duration"`
	ComparisonOffset string `yaml:"comparison_offset"`
}

// WindowDurationParsed returns the parsed window duration.
func (a *AnalysisConfig) WindowDurationParsed() (time.Duration, error) {
	return time.ParseDuration(a.WindowDuration)
}

// ComparisonOffsetParsed returns the parsed comparison offset.
func (a *AnalysisConfig) ComparisonOffsetParsed() (time.Duration, error) {
	return time.ParseDuration(a.ComparisonOffset)
}

// RulesConfig contains all rule configurations.
type RulesConfig struct {
	SlowSQL         SlowSQLRuleConfig         `yaml:"slow_sql"`
	Regression      RegressionRuleConfig      `yaml:"regression"`
	IndexSuggestion IndexSuggestionRuleConfig `yaml:"index_suggestion"`
}

// SlowSQLRuleConfig defines slow SQL detection parameters.
type SlowSQLRuleConfig struct {
	TopN   int    `yaml:"top_n"`
	RankBy string `yaml:"rank_by"`
}

// RegressionRuleConfig defines regression detection parameters.
type RegressionRuleConfig struct {
	ThresholdPercent float64 `yaml:"threshold_percent"`
}

// IndexSuggestionRuleConfig defines index suggestion filtering.
type IndexSuggestionRuleConfig struct {
	MinImprovementPercent float64 `yaml:"min_improvement_percent"`
}

// NotifierConfig holds notification channel settings.
type NotifierConfig struct {
	Type       string `yaml:"type"`
	WebhookURL string `yaml:"webhook_url"`
	Retries    int    `yaml:"retries"`
	RetryDelay string `yaml:"retry_delay"`
}

// RetryDelayParsed returns the parsed retry delay duration.
func (n *NotifierConfig) RetryDelayParsed() (time.Duration, error) {
	return time.ParseDuration(n.RetryDelay)
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port      int  `yaml:"port"`
	DeepCheck bool `yaml:"deep_check"`
}

// Load reads and parses the configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// expandEnvVars expands ${VAR} and ${VAR:-default} patterns in the input string.
func expandEnvVars(input string) string {
	// Pattern: ${VAR:-default} or ${VAR}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = parts[2]
		}

		if val, exists := os.LookupEnv(varName); exists {
			return val
		}
		return defaultVal
	})
}

// applyDefaults sets default values for any unset configuration fields.
func applyDefaults(cfg *Config) {
	// Database defaults
	if cfg.Database.Host == "" {
		cfg.Database.Host = "127.0.0.1"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "powa_readonly"
	}
	if cfg.Database.DBName == "" {
		cfg.Database.DBName = "powa"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}

	// Schedule defaults (6-field cron with seconds)
	if cfg.Schedule.Cron == "" {
		cfg.Schedule.Cron = "0 0 9 * * 1" // Every Monday at 9:00 AM
	}

	// Analysis defaults
	if cfg.Analysis.WindowDuration == "" {
		cfg.Analysis.WindowDuration = "24h"
	}
	if cfg.Analysis.ComparisonOffset == "" {
		cfg.Analysis.ComparisonOffset = "168h"
	}

	// Rules defaults
	if cfg.Rules.SlowSQL.TopN == 0 {
		cfg.Rules.SlowSQL.TopN = 10
	}
	if cfg.Rules.SlowSQL.RankBy == "" {
		cfg.Rules.SlowSQL.RankBy = "total_time"
	}
	if cfg.Rules.Regression.ThresholdPercent == 0 {
		cfg.Rules.Regression.ThresholdPercent = 50
	}
	if cfg.Rules.IndexSuggestion.MinImprovementPercent == 0 {
		cfg.Rules.IndexSuggestion.MinImprovementPercent = 30
	}

	// Notifier defaults
	if cfg.Notifier.Type == "" {
		cfg.Notifier.Type = "console"
	}
	if cfg.Notifier.Retries == 0 {
		cfg.Notifier.Retries = 3
	}
	if cfg.Notifier.RetryDelay == "" {
		cfg.Notifier.RetryDelay = "1s"
	}

	// Server defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	var errs []string

	// Validate database
	if c.Database.Host == "" {
		errs = append(errs, "database.host is required")
	}

	// Validate notifier type
	validNotifierTypes := map[string]bool{"wecom": true, "console": true}
	if !validNotifierTypes[c.Notifier.Type] {
		errs = append(errs, "notifier.type must be one of: wecom, console")
	}

	// Validate notifier webhook URL
	if c.Notifier.Type == "wecom" && c.Notifier.WebhookURL == "" {
		errs = append(errs, "notifier.webhook_url is required when type is 'wecom'")
	}

	// Validate durations
	if _, err := c.Analysis.WindowDurationParsed(); err != nil {
		errs = append(errs, fmt.Sprintf("analysis.window_duration is invalid: %v", err))
	}
	if _, err := c.Analysis.ComparisonOffsetParsed(); err != nil {
		errs = append(errs, fmt.Sprintf("analysis.comparison_offset is invalid: %v", err))
	}
	if _, err := c.Notifier.RetryDelayParsed(); err != nil {
		errs = append(errs, fmt.Sprintf("notifier.retry_delay is invalid: %v", err))
	}

	// Validate rule values
	if c.Rules.SlowSQL.TopN < 1 {
		errs = append(errs, "rules.slow_sql.top_n must be at least 1")
	}
	validRankBy := map[string]bool{"total_time": true, "mean_time": true, "cpu_time": true, "io_time": true}
	if !validRankBy[c.Rules.SlowSQL.RankBy] {
		errs = append(errs, "rules.slow_sql.rank_by must be one of: total_time, mean_time, cpu_time, io_time")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
