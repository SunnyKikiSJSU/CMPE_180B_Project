// Package config reads and validates the API's environment variables
// (BACKEND_PLAN §6.1). Secret values are never logged.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// DatabaseSource says where the database credentials come from.
type DatabaseSource string

const (
	// SourceURL: DATABASE_URL (wins when set).
	SourceURL DatabaseSource = "DATABASE_URL"
	// SourceSecret: the Terraform Secrets Manager secret named by DB_SECRET_ARN,
	// read at startup with the instance role (conflict.md #4).
	SourceSecret DatabaseSource = "DB_SECRET_ARN"
	// SourceEnv: DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE (.env).
	SourceEnv DatabaseSource = "DB_*"
)

// Config is the validated runtime configuration.
type Config struct {
	AppEnv string
	Port   int

	DatabaseSource DatabaseSource
	// DatabaseURL is set for SourceURL and SourceEnv; for SourceSecret it is
	// built at startup from the secret (database.ResolveURL).
	DatabaseURL string
	DBSecretARN string
	DBSSLMode   string
	AWSRegion   string

	JWTSecret    []byte
	CookieSecure bool
	// TrustProxy reads the client address from X-Forwarded-For (for rate
	// limits). Set it only when the API is reachable through the web tier
	// alone, never directly from the internet.
	TrustProxy bool

	// OpenLibraryURL is the Open Library base URL (tests point it at a fake).
	OpenLibraryURL string
	// OpenLibraryUserAgent identifies the app to Open Library, which asks
	// frequent API users to name the app and a contact.
	OpenLibraryUserAgent string
}

// DefaultOpenLibraryUserAgent is sent when OPENLIBRARY_USER_AGENT is unset.
const DefaultOpenLibraryUserAgent = "BookMetaversePortal/1.0 (SJSU CMPE 180B course project)"

// placeholderSecrets are example values from .env.example and the Compose
// defaults. Signing sessions with a published value would let anyone forge
// them, so the API refuses to start with one outside APP_ENV=local.
var placeholderSecrets = map[string]bool{
	"replace_with_a_long_random_value": true,
	"local-development-secret":         true,
}

var (
	sslModes = map[string]bool{"disable": true, "allow": true, "prefer": true, "require": true, "verify-ca": true, "verify-full": true}
)

// Load reads the environment (lookup is os.LookupEnv in production).
func Load(lookup func(string) (string, bool)) (*Config, error) {
	get := func(key, fallback string) string {
		if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		return fallback
	}
	var problems []string

	cfg := &Config{
		AppEnv:               get("APP_ENV", "local"),
		OpenLibraryURL:       strings.TrimRight(get("OPENLIBRARY_URL", "https://openlibrary.org"), "/"),
		OpenLibraryUserAgent: get("OPENLIBRARY_USER_AGENT", DefaultOpenLibraryUserAgent),
	}
	if u, err := url.Parse(cfg.OpenLibraryURL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		problems = append(problems, "OPENLIBRARY_URL must be an http(s) URL such as https://openlibrary.org")
	}

	port, err := strconv.Atoi(get("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		problems = append(problems, "PORT must be a number from 1 to 65535")
	}
	cfg.Port = port

	cookieSecure, err := strconv.ParseBool(get("COOKIE_SECURE", "false"))
	if err != nil {
		problems = append(problems, "COOKIE_SECURE must be true or false")
	}
	cfg.CookieSecure = cookieSecure

	trustProxy, err := strconv.ParseBool(get("TRUST_PROXY", "false"))
	if err != nil {
		problems = append(problems, "TRUST_PROXY must be true or false")
	}
	cfg.TrustProxy = trustProxy

	secret, _ := lookup("JWT_SECRET")
	switch {
	case secret == "":
		problems = append(problems, "JWT_SECRET must be set")
	case placeholderSecrets[secret] && cfg.AppEnv != "local":
		problems = append(problems, "JWT_SECRET is an example value; set a long random value (for example: openssl rand -hex 32)")
	}
	cfg.JWTSecret = []byte(secret)

	problems = append(problems, loadDatabase(cfg, get)...)

	if len(problems) > 0 {
		return nil, errors.New("invalid configuration: " + strings.Join(problems, "; "))
	}
	return cfg, nil
}

func loadDatabase(cfg *Config, get func(string, string) string) []string {
	cfg.DBSSLMode = get("DB_SSLMODE", "")
	if cfg.DBSSLMode != "" && !sslModes[cfg.DBSSLMode] {
		return []string{"DB_SSLMODE must be one of disable, allow, prefer, require, verify-ca, verify-full"}
	}

	if databaseURL := get("DATABASE_URL", ""); databaseURL != "" {
		cfg.DatabaseSource = SourceURL
		cfg.DatabaseURL = databaseURL
		if _, err := url.Parse(databaseURL); err != nil {
			return []string{"DATABASE_URL is not a valid URL"}
		}
		return nil
	}

	if arn := get("DB_SECRET_ARN", ""); arn != "" {
		cfg.DatabaseSource = SourceSecret
		cfg.DBSecretARN = arn
		// Secrets Manager ARNs look like arn:aws:secretsmanager:<region>:<account>:secret:<name>.
		parts := strings.Split(arn, ":")
		if len(parts) < 7 || parts[0] != "arn" || parts[2] != "secretsmanager" {
			return []string{"DB_SECRET_ARN must be a Secrets Manager secret ARN"}
		}
		cfg.AWSRegion = get("AWS_REGION", parts[3])
		if cfg.DBSSLMode == "" {
			cfg.DBSSLMode = "require"
		}
		return nil
	}

	cfg.DatabaseSource = SourceEnv
	if cfg.DBSSLMode == "" {
		cfg.DBSSLMode = "prefer"
	}
	password := get("DB_PASSWORD", "")
	if password == "" {
		return []string{"set DATABASE_URL, DB_SECRET_ARN, or DB_PASSWORD (with DB_HOST, DB_NAME, DB_USER)"}
	}
	cfg.DatabaseURL = BuildURL(get("DB_HOST", "localhost"), get("DB_PORT", "5432"), get("DB_NAME", "book_metaverse"),
		get("DB_USER", "book_app"), password, cfg.DBSSLMode)
	return nil
}

// BuildURL makes a postgres:// URL; url.UserPassword escapes reserved characters.
func BuildURL(host, port, name, user, password, sslmode string) string {
	connection := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: url.Values{"sslmode": {sslmode}}.Encode(),
	}
	return connection.String()
}

// LogSummary logs non-secret settings at startup.
func (c *Config) LogSummary() {
	slog.Info("configuration loaded",
		"app_env", c.AppEnv,
		"port", c.Port,
		"database_source", string(c.DatabaseSource),
		"db_sslmode", c.DBSSLMode,
		"cookie_secure", c.CookieSecure,
		"trust_proxy", c.TrustProxy,
		"openlibrary_url", c.OpenLibraryURL,
		"openlibrary_user_agent_set", c.OpenLibraryUserAgent != DefaultOpenLibraryUserAgent,
	)
	if len(c.JWTSecret) < 32 {
		slog.Warn("JWT_SECRET is shorter than 32 bytes; use a long random value outside local development")
	}
}

// FromEnv loads the process environment.
func FromEnv() (*Config, error) { return Load(os.LookupEnv) }

// String hides secrets if a Config is ever printed.
func (c *Config) String() string {
	return fmt.Sprintf("Config{AppEnv:%s Port:%d DatabaseSource:%s}", c.AppEnv, c.Port, c.DatabaseSource)
}
