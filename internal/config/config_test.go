// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package config

import (
	"os"
	"strings"
	"testing"
)

// TestLoad_Defaults verifies that Load returns sensible development defaults
// when no environment variables are set.
func TestLoad_Defaults(t *testing.T) {
	// Clear all environment variables that Load reads so we get pure defaults.
	envVars := []string{
		"APP_HOST", "APP_PORT", "APP_ENV",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
		"VALKEY_HOST", "VALKEY_PORT", "VALKEY_PASSWORD",
		"AI_PROVIDER",
		"OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_BASE_URL",
		"GEMINI_API_KEY", "GEMINI_MODEL", "GEMINI_BASE_URL",
		"CLAUDE_API_KEY", "CLAUDE_MODEL", "CLAUDE_BASE_URL",
		"MISTRAL_API_KEY", "MISTRAL_MODEL", "MISTRAL_BASE_URL",
		"S3_ENDPOINT", "S3_REGION", "S3_ACCESS_KEY", "S3_SECRET_KEY",
		"S3_BUCKET_PUBLIC", "S3_BUCKET_PRIVATE", "S3_PUBLIC_URL",
	}
	for _, key := range envVars {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
	// Re-set them with t.Setenv so they are restored after the test.
	// Actually, t.Setenv already handles cleanup. We need to truly unset them.
	// t.Setenv sets the var; to unset we need a different approach.
	// The cleanest way: set them to empty so envOrDefault falls through to defaults.
	for _, key := range envVars {
		t.Setenv(key, "")
	}
	// But envOrDefault checks v != "", so empty string means "use default". Good.

	// However, t.Setenv sets the value — we need them to be genuinely empty.
	// os.Unsetenv won't be reverted by t.Setenv. Let's just set to "" which
	// envOrDefault treats the same as unset.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	defaults := map[string]string{
		"Host":           "0.0.0.0",
		"Port":           "8080",
		"Env":            "development",
		"DBHost":         "localhost",
		"DBPort":         "5432",
		"DBUser":         "yaaicms",
		"DBPassword":     "changeme",
		"DBName":         "yaaicms",
		"ValkeyHost":     "localhost",
		"ValkeyPort":     "6379",
		"ValkeyPassword": "",
		"AIProvider":     "gemini",
		"OpenAIModel":    "gpt-4o",
		"OpenAIBaseURL":  "https://api.openai.com/v1",
		"GeminiModel":    "gemini-3.1-pro-preview",
		"GeminiBaseURL":  "https://generativelanguage.googleapis.com",
		"ClaudeModel":    "claude-sonnet-4-6",
		"ClaudeBaseURL":  "https://api.anthropic.com",
		"MistralModel":   "mistral-large-latest",
		"MistralBaseURL": "https://api.mistral.ai/v1",
		"S3Region":       "fsn1",
		"S3BucketPublic": "yaaicms-public",
		"S3BucketPrivate":"yaaicms-private",
	}

	// Use a helper to avoid massive repetition.
	check := func(field, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}

	check("Host", cfg.Host, defaults["Host"])
	check("Port", cfg.Port, defaults["Port"])
	check("Env", cfg.Env, defaults["Env"])
	check("DBHost", cfg.DBHost, defaults["DBHost"])
	check("DBPort", cfg.DBPort, defaults["DBPort"])
	check("DBUser", cfg.DBUser, defaults["DBUser"])
	check("DBPassword", cfg.DBPassword, defaults["DBPassword"])
	check("DBName", cfg.DBName, defaults["DBName"])
	check("ValkeyHost", cfg.ValkeyHost, defaults["ValkeyHost"])
	check("ValkeyPort", cfg.ValkeyPort, defaults["ValkeyPort"])
	check("ValkeyPassword", cfg.ValkeyPassword, defaults["ValkeyPassword"])
	check("AIProvider", cfg.AIProvider, defaults["AIProvider"])
	check("OpenAIModel", cfg.OpenAIModel, defaults["OpenAIModel"])
	check("OpenAIBaseURL", cfg.OpenAIBaseURL, defaults["OpenAIBaseURL"])
	check("GeminiModel", cfg.GeminiModel, defaults["GeminiModel"])
	check("GeminiBaseURL", cfg.GeminiBaseURL, defaults["GeminiBaseURL"])
	check("ClaudeModel", cfg.ClaudeModel, defaults["ClaudeModel"])
	check("ClaudeBaseURL", cfg.ClaudeBaseURL, defaults["ClaudeBaseURL"])
	check("MistralModel", cfg.MistralModel, defaults["MistralModel"])
	check("MistralBaseURL", cfg.MistralBaseURL, defaults["MistralBaseURL"])
	check("S3Region", cfg.S3Region, defaults["S3Region"])
	check("S3BucketPublic", cfg.S3BucketPublic, defaults["S3BucketPublic"])
	check("S3BucketPrivate", cfg.S3BucketPrivate, defaults["S3BucketPrivate"])
}

// TestLoad_EnvOverrides verifies that every environment variable properly
// overrides the default value.
func TestLoad_EnvOverrides(t *testing.T) {
	overrides := map[string]string{
		"APP_HOST":           "127.0.0.1",
		"APP_PORT":           "9090",
		"APP_ENV":            "testing",
		"POSTGRES_HOST":      "db.example.com",
		"POSTGRES_PORT":      "5433",
		"POSTGRES_USER":      "testuser",
		"POSTGRES_PASSWORD":  "testpass",
		"POSTGRES_DB":        "testdb",
		"VALKEY_HOST":        "cache.example.com",
		"VALKEY_PORT":        "6380",
		"VALKEY_PASSWORD":    "cachepass",
		"AI_PROVIDER":        "openai",
		"OPENAI_API_KEY":     "sk-test-key",
		"OPENAI_MODEL":       "gpt-4-turbo",
		"OPENAI_BASE_URL":    "https://custom.openai.example.com",
		"GEMINI_API_KEY":     "gemini-test-key",
		"GEMINI_MODEL":       "gemini-pro",
		"GEMINI_BASE_URL":    "https://custom.gemini.example.com",
		"CLAUDE_API_KEY":     "claude-test-key",
		"CLAUDE_MODEL":       "claude-3-opus",
		"CLAUDE_BASE_URL":    "https://custom.claude.example.com",
		"MISTRAL_API_KEY":    "mistral-test-key",
		"MISTRAL_MODEL":      "mistral-medium",
		"MISTRAL_BASE_URL":   "https://custom.mistral.example.com",
		"S3_ENDPOINT":        "https://s3.example.com",
		"S3_REGION":          "eu-central-1",
		"S3_ACCESS_KEY":      "AKIATEST",
		"S3_SECRET_KEY":      "secrettest",
		"S3_BUCKET_PUBLIC":   "my-public",
		"S3_BUCKET_PRIVATE":  "my-private",
		"S3_PUBLIC_URL":      "https://cdn.example.com",
	}

	for key, val := range overrides {
		t.Setenv(key, val)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	check := func(field, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}

	check("Host", cfg.Host, "127.0.0.1")
	check("Port", cfg.Port, "9090")
	check("Env", cfg.Env, "testing")
	check("DBHost", cfg.DBHost, "db.example.com")
	check("DBPort", cfg.DBPort, "5433")
	check("DBUser", cfg.DBUser, "testuser")
	check("DBPassword", cfg.DBPassword, "testpass")
	check("DBName", cfg.DBName, "testdb")
	check("ValkeyHost", cfg.ValkeyHost, "cache.example.com")
	check("ValkeyPort", cfg.ValkeyPort, "6380")
	check("ValkeyPassword", cfg.ValkeyPassword, "cachepass")
	check("AIProvider", cfg.AIProvider, "openai")
	check("OpenAIKey", cfg.OpenAIKey, "sk-test-key")
	check("OpenAIModel", cfg.OpenAIModel, "gpt-4-turbo")
	check("OpenAIBaseURL", cfg.OpenAIBaseURL, "https://custom.openai.example.com")
	check("GeminiKey", cfg.GeminiKey, "gemini-test-key")
	check("GeminiModel", cfg.GeminiModel, "gemini-pro")
	check("GeminiBaseURL", cfg.GeminiBaseURL, "https://custom.gemini.example.com")
	check("ClaudeKey", cfg.ClaudeKey, "claude-test-key")
	check("ClaudeModel", cfg.ClaudeModel, "claude-3-opus")
	check("ClaudeBaseURL", cfg.ClaudeBaseURL, "https://custom.claude.example.com")
	check("MistralKey", cfg.MistralKey, "mistral-test-key")
	check("MistralModel", cfg.MistralModel, "mistral-medium")
	check("MistralBaseURL", cfg.MistralBaseURL, "https://custom.mistral.example.com")
	check("S3Endpoint", cfg.S3Endpoint, "https://s3.example.com")
	check("S3Region", cfg.S3Region, "eu-central-1")
	check("S3AccessKey", cfg.S3AccessKey, "AKIATEST")
	check("S3SecretKey", cfg.S3SecretKey, "secrettest")
	check("S3BucketPublic", cfg.S3BucketPublic, "my-public")
	check("S3BucketPrivate", cfg.S3BucketPrivate, "my-private")
	check("S3PublicURL", cfg.S3PublicURL, "https://cdn.example.com")
}

// TestLoad_ProductionRequiresPassword verifies that production mode rejects
// the default "changeme" password and accepts a real one.
func TestLoad_ProductionRequiresPassword(t *testing.T) {
	t.Run("rejects default password", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		// Do not set POSTGRES_PASSWORD — it will default to "changeme".
		t.Setenv("POSTGRES_PASSWORD", "")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() should return an error when production uses default password")
		}
		if !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
			t.Errorf("error should mention POSTGRES_PASSWORD, got: %v", err)
		}
	})

	t.Run("rejects explicit changeme", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("POSTGRES_PASSWORD", "changeme")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() should return an error when production uses 'changeme'")
		}
	})

	t.Run("accepts real password", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("POSTGRES_PASSWORD", "s3cur3-pr0d-p@ssw0rd")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned unexpected error: %v", err)
		}
		if cfg.DBPassword != "s3cur3-pr0d-p@ssw0rd" {
			t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "s3cur3-pr0d-p@ssw0rd")
		}
	})
}

// TestLoad_DevelopmentAllowsDefaultPassword ensures the default password
// does not cause an error outside of production.
func TestLoad_DevelopmentAllowsDefaultPassword(t *testing.T) {
	envs := []string{"development", "testing", ""}
	for _, env := range envs {
		t.Run("env="+env, func(t *testing.T) {
			if env != "" {
				t.Setenv("APP_ENV", env)
			} else {
				t.Setenv("APP_ENV", "")
			}
			t.Setenv("POSTGRES_PASSWORD", "")

			_, err := Load()
			if err != nil {
				t.Fatalf("Load() should not error in %q mode with default password, got: %v", env, err)
			}
		})
	}
}

// TestDSN verifies the PostgreSQL connection string format.
func TestDSN(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name: "default local config",
			cfg: Config{
				DBUser:     "yaaicms",
				DBPassword: "changeme",
				DBHost:     "localhost",
				DBPort:     "5432",
				DBName:     "yaaicms",
			},
			expected: "postgres://yaaicms:changeme@localhost:5432/yaaicms?sslmode=disable",
		},
		{
			name: "custom remote config",
			cfg: Config{
				DBUser:     "prod_user",
				DBPassword: "p@ss/w0rd",
				DBHost:     "db.prod.example.com",
				DBPort:     "5433",
				DBName:     "cms_production",
			},
			expected: "postgres://prod_user:p@ss/w0rd@db.prod.example.com:5433/cms_production?sslmode=disable",
		},
		{
			name: "password with special characters",
			cfg: Config{
				DBUser:     "admin",
				DBPassword: "h@ck&me!",
				DBHost:     "10.0.0.5",
				DBPort:     "5432",
				DBName:     "testdb",
			},
			expected: "postgres://admin:h@ck&me!@10.0.0.5:5432/testdb?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.DSN()
			if got != tt.expected {
				t.Errorf("DSN() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestAddr verifies the server listen address format.
func TestAddr(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		expected string
	}{
		{
			name:     "default",
			host:     "0.0.0.0",
			port:     "8080",
			expected: "0.0.0.0:8080",
		},
		{
			name:     "localhost with custom port",
			host:     "127.0.0.1",
			port:     "3000",
			expected: "127.0.0.1:3000",
		},
		{
			name:     "empty host",
			host:     "",
			port:     "8080",
			expected: ":8080",
		},
		{
			name:     "ipv6 host",
			host:     "::1",
			port:     "443",
			expected: "::1:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Host: tt.host, Port: tt.port}
			got := cfg.Addr()
			if got != tt.expected {
				t.Errorf("Addr() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestIsDev verifies the IsDev method for various environment modes.
func TestIsDev(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{name: "development mode", env: "development", expected: true},
		{name: "production mode", env: "production", expected: false},
		{name: "testing mode", env: "testing", expected: false},
		{name: "empty string", env: "", expected: false},
		{name: "uppercase DEVELOPMENT", env: "DEVELOPMENT", expected: false},
		{name: "mixed case Development", env: "Development", expected: false},
		{name: "dev shorthand", env: "dev", expected: false},
		{name: "staging", env: "staging", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Env: tt.env}
			got := cfg.IsDev()
			if got != tt.expected {
				t.Errorf("IsDev() = %v, want %v (env=%q)", got, tt.expected, tt.env)
			}
		})
	}
}

// TestEnvOrDefault verifies the unexported helper function indirectly
// through Load. This test confirms that an explicitly set env var wins
// over the default, and that an empty var falls through to the default.
func TestEnvOrDefault(t *testing.T) {
	t.Run("set value wins", func(t *testing.T) {
		t.Setenv("APP_PORT", "3000")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned unexpected error: %v", err)
		}
		if cfg.Port != "3000" {
			t.Errorf("Port = %q, want %q", cfg.Port, "3000")
		}
	})

	t.Run("empty value uses default", func(t *testing.T) {
		t.Setenv("APP_PORT", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned unexpected error: %v", err)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
		}
	})
}
