package config

import (
	"os"
	"strings"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "no variables",
			input:    "hello world",
			envVars:  nil,
			expected: "hello world",
		},
		{
			name:     "simple variable",
			input:    "host: ${MY_HOST}",
			envVars:  map[string]string{"MY_HOST": "localhost"},
			expected: "host: localhost",
		},
		{
			name:     "variable with default - env set",
			input:    "port: ${MY_PORT:-5432}",
			envVars:  map[string]string{"MY_PORT": "3306"},
			expected: "port: 3306",
		},
		{
			name:     "variable with default - env not set",
			input:    "port: ${MY_PORT:-5432}",
			envVars:  nil,
			expected: "port: 5432",
		},
		{
			name:     "variable without default - env not set",
			input:    "password: ${MY_PASSWORD}",
			envVars:  nil,
			expected: "password: ",
		},
		{
			name:     "multiple variables",
			input:    "host: ${HOST:-localhost}, port: ${PORT:-5432}",
			envVars:  map[string]string{"HOST": "db.example.com"},
			expected: "host: db.example.com, port: 5432",
		},
		{
			name:     "empty default value",
			input:    "value: ${EMPTY:-}",
			envVars:  nil,
			expected: "value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set env vars
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// Check database defaults
	if cfg.Database.Host != "127.0.0.1" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "127.0.0.1")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}
	if cfg.Database.User != "powa_readonly" {
		t.Errorf("Database.User = %q, want %q", cfg.Database.User, "powa_readonly")
	}
	if cfg.Database.DBName != "powa" {
		t.Errorf("Database.DBName = %q, want %q", cfg.Database.DBName, "powa")
	}

	// Check rules defaults
	if cfg.Rules.SlowSQL.TopN != 10 {
		t.Errorf("Rules.SlowSQL.TopN = %d, want %d", cfg.Rules.SlowSQL.TopN, 10)
	}
	if cfg.Rules.SlowSQL.RankBy != "total_time" {
		t.Errorf("Rules.SlowSQL.RankBy = %q, want %q", cfg.Rules.SlowSQL.RankBy, "total_time")
	}
	if cfg.Rules.Regression.ThresholdPercent != 50 {
		t.Errorf("Rules.Regression.ThresholdPercent = %f, want %f", cfg.Rules.Regression.ThresholdPercent, 50.0)
	}

	// Check notifier defaults
	if cfg.Notifier.Type != "console" {
		t.Errorf("Notifier.Type = %q, want %q", cfg.Notifier.Type, "console")
	}
	if cfg.Notifier.Retries != 3 {
		t.Errorf("Notifier.Retries = %d, want %d", cfg.Notifier.Retries, 3)
	}

	// Check server defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config with console notifier",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules: RulesConfig{
					SlowSQL:         SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"},
					Regression:      RegressionRuleConfig{ThresholdPercent: 50},
					IndexSuggestion: IndexSuggestionRuleConfig{MinImprovementPercent: 30},
				},
				Notifier: NotifierConfig{Type: "console", Retries: 3, RetryDelay: "1s"},
				Server:   ServerConfig{Port: 8080},
			},
			wantErr: false,
		},
		{
			name: "valid config with wecom notifier",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules: RulesConfig{
					SlowSQL:         SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"},
					Regression:      RegressionRuleConfig{ThresholdPercent: 50},
					IndexSuggestion: IndexSuggestionRuleConfig{MinImprovementPercent: 30},
				},
				Notifier: NotifierConfig{Type: "wecom", WebhookURL: "https://example.com/webhook", Retries: 3, RetryDelay: "1s"},
				Server:   ServerConfig{Port: 8080},
			},
			wantErr: false,
		},
		{
			name: "missing database host",
			cfg: Config{
				Database: DatabaseConfig{Host: "", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "invalid notifier type",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "invalid", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "wecom without webhook URL",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "wecom", WebhookURL: "", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "invalid duration",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "invalid", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "invalid TopN",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 0, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "invalid rank_by",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "invalid"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "valid expected_extensions",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432, ExpectedExtensions: []string{"pg_stat_kcache", "pg_qualstats"}},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: false,
		},
		{
			name: "invalid expected_extensions",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432, ExpectedExtensions: []string{"powa", "pg_qualstats"}},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
		{
			name: "invalid expected_extensions deduplicated",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Port: 5432, ExpectedExtensions: []string{"powa", "powa", "other"}},
				Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
				Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
				Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_ExpectedExtensions_Deduplicated(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{Host: "localhost", Port: 5432, ExpectedExtensions: []string{"powa", "powa", "other"}},
		Analysis: AnalysisConfig{WindowDuration: "24h", ComparisonOffset: "168h"},
		Rules:    RulesConfig{SlowSQL: SlowSQLRuleConfig{TopN: 10, RankBy: "total_time"}},
		Notifier: NotifierConfig{Type: "console", RetryDelay: "1s"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid expected_extensions")
	}
	msg := err.Error()
	// Each invalid extension should be reported only once (powa once, other once)
	if strings.Count(msg, `"powa"`) != 1 {
		t.Errorf("expected exactly one error line for powa, got %d mentions in: %s", strings.Count(msg, `"powa"`), msg)
	}
	if strings.Count(msg, `"other"`) != 1 {
		t.Errorf("expected exactly one error line for other, got %d mentions in: %s", strings.Count(msg, `"other"`), msg)
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn := cfg.DSN(); dsn != expected {
		t.Errorf("DSN() = %q, want %q", dsn, expected)
	}
}
