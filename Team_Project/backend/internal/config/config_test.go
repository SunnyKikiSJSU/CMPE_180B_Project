package config

import (
	"strings"
	"testing"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { v, ok := values[key]; return v, ok }
}

func TestLoadDatabaseSources(t *testing.T) {
	base := map[string]string{"JWT_SECRET": "0123456789abcdef0123456789abcdef"}
	with := func(extra map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	cfg, err := Load(env(with(map[string]string{"DB_PASSWORD": "p@ss word", "DB_HOST": "db", "DB_NAME": "appdb", "DB_USER": "u"})))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseSource != SourceEnv || cfg.DatabaseURL != "postgres://u:p%40ss%20word@db:5432/appdb?sslmode=prefer" {
		t.Fatalf("env source: %s %s", cfg.DatabaseSource, cfg.DatabaseURL)
	}

	arn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:cmpe180b-team-project/db-credentials-AbCdEf"
	cfg, err = Load(env(with(map[string]string{"DB_SECRET_ARN": arn, "DB_PASSWORD": "ignored"})))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseSource != SourceSecret || cfg.AWSRegion != "us-west-2" || cfg.DBSSLMode != "require" || cfg.DatabaseURL != "" {
		t.Fatalf("secret source: %+v", cfg)
	}

	cfg, err = Load(env(with(map[string]string{"DATABASE_URL": "postgres://x@y/z", "DB_SECRET_ARN": arn})))
	if err != nil || cfg.DatabaseSource != SourceURL {
		t.Fatalf("DATABASE_URL must win: %v %v", err, cfg)
	}
}

func TestLoadRejects(t *testing.T) {
	cases := map[string]map[string]string{
		"no JWT secret":                    {"DB_PASSWORD": "x"},
		"placeholder secret outside local": {"APP_ENV": "rds", "JWT_SECRET": "local-development-secret", "DB_PASSWORD": "x"},
		"no database credentials":          {"JWT_SECRET": "s"},
		"bad port":                         {"JWT_SECRET": "s", "DB_PASSWORD": "x", "PORT": "70000"},
		"bad sslmode":                      {"JWT_SECRET": "s", "DB_PASSWORD": "x", "DB_SSLMODE": "on"},
		"bad secret ARN":                   {"JWT_SECRET": "s", "DB_SECRET_ARN": "my-secret"},
		"bad cookie flag":                  {"JWT_SECRET": "s", "DB_PASSWORD": "x", "COOKIE_SECURE": "yes please"},
		"bad trust proxy flag":             {"JWT_SECRET": "s", "DB_PASSWORD": "x", "TRUST_PROXY": "maybe"},
		"bad Open Library URL":             {"JWT_SECRET": "s", "DB_PASSWORD": "x", "OPENLIBRARY_URL": "ftp://openlibrary.org"},
	}
	for name, values := range cases {
		if _, err := Load(env(values)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	// The placeholder is allowed for local development (Compose default).
	if _, err := Load(env(map[string]string{"JWT_SECRET": "local-development-secret", "DB_PASSWORD": "x"})); err != nil {
		t.Errorf("local placeholder: %v", err)
	}
}

func TestErrorsNeverContainSecrets(t *testing.T) {
	_, err := Load(env(map[string]string{"JWT_SECRET": "", "DB_PASSWORD": "hunter2-secret", "PORT": "x"}))
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaks or is missing: %v", err)
	}
	cfg, _ := Load(env(map[string]string{"JWT_SECRET": "jwt-secret-value-xyz", "DB_PASSWORD": "hunter2"}))
	if s := cfg.String(); strings.Contains(s, "hunter2") || strings.Contains(s, "jwt-secret") {
		t.Fatalf("String leaks secrets: %s", s)
	}
}
