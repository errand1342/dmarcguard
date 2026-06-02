package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/caarlos0/env/v11"
	"github.com/goccy/go-json"
)

var (
	// ErrMissingIMAPHost is returned when IMAP host is not configured
	ErrMissingIMAPHost = errors.New("IMAP_HOST is required: set via environment variable or config file")
	// ErrMissingIMAPUsername is returned when IMAP username is not configured
	ErrMissingIMAPUsername = errors.New("IMAP_USERNAME is required: set via environment variable or config file")
	// ErrMissingIMAPPassword is returned when IMAP password is not configured
	ErrMissingIMAPPassword = errors.New("IMAP_PASSWORD is required: set via environment variable or config file")
)

// Config holds the application configuration
type Config struct {
	LogLevel    string         `json:"log_level" env:"LOG_LEVEL" envDefault:"info"`
	ColoredLogs bool           `json:"colored_logs" env:"COLORED_LOGS" envDefault:"false"`
	IMAP        IMAPConfig     `json:"imap"`
	Graph       GraphConfig    `json:"graph"`
	Database    DatabaseConfig `json:"database"`
	Server      ServerConfig   `json:"server"`
}

// GraphConfig holds Microsoft Graph API configuration
type GraphConfig struct {
	Enabled      bool   `json:"enabled" env:"GRAPH_ENABLED" envDefault:"false"`
	TenantID     string `json:"tenant_id" env:"GRAPH_TENANT_ID"`
	ClientID     string `json:"client_id" env:"GRAPH_CLIENT_ID"`
	ClientSecret string `json:"client_secret" env:"GRAPH_CLIENT_SECRET"`
	CertPath     string `json:"cert_path" env:"GRAPH_CERT_PATH"`
	CertKeyPath  string `json:"cert_key_path" env:"GRAPH_CERT_KEY_PATH"`
	Mailbox      string `json:"mailbox" env:"GRAPH_MAILBOX"`
	FolderPath   string `json:"folder_path" env:"GRAPH_FOLDER_PATH" envDefault:"INBOX"`
	MarkAsRead   bool   `json:"mark_as_read" env:"GRAPH_MARK_AS_READ" envDefault:"true"`
}

// LoadFromEnv loads Graph configuration from environment variables
func (c *GraphConfig) LoadFromEnv() {
	c.Enabled = parseBool(os.Getenv("GRAPH_ENABLED"), false)
	c.TenantID = os.Getenv("GRAPH_TENANT_ID")
	c.ClientID = os.Getenv("GRAPH_CLIENT_ID")
	c.ClientSecret = os.Getenv("GRAPH_CLIENT_SECRET")
	c.CertPath = os.Getenv("GRAPH_CERT_PATH")
	c.CertKeyPath = os.Getenv("GRAPH_CERT_KEY_PATH")
	c.Mailbox = os.Getenv("GRAPH_MAILBOX")
	c.FolderPath = os.Getenv("GRAPH_FOLDER_PATH")
	if c.FolderPath == "" {
		c.FolderPath = "INBOX"
	}
	c.MarkAsRead = parseBool(os.Getenv("GRAPH_MARK_AS_READ"), true)
}

// ValidateGraph checks if Graph configuration is valid
func (c *GraphConfig) ValidateGraph() error {
	if !c.Enabled {
		return nil
	}

	required := map[string]string{
		"tenant_id": c.TenantID,
		"client_id": c.ClientID,
		"mailbox":   c.Mailbox,
	}

	// Check for authentication method
	hasSecret := c.ClientSecret != ""
	hasCert := c.CertPath != "" && c.CertKeyPath != ""

	if !hasSecret && !hasCert {
		required["client_secret_or_cert"] = "" // This will trigger the error
	}

	for k, v := range required {
		if v == "" {
			return fmt.Errorf("GRAPH_CONFIG_ERROR: missing or invalid field '%s'", k)
		}
	}

	return nil
}

func parseBool(s string, defaultVal bool) bool {
	if s == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return defaultVal
	}
	return b
}

// IMAPConfig holds IMAP server configuration
type IMAPConfig struct {
	Host     string `json:"host" env:"IMAP_HOST"`
	Port     int    `json:"port" env:"IMAP_PORT" envDefault:"993"`
	Username string `json:"username" env:"IMAP_USERNAME"`
	Password string `json:"password" env:"IMAP_PASSWORD"`
	Mailbox  string `json:"mailbox" env:"IMAP_MAILBOX" envDefault:"INBOX"`
	UseTLS   bool   `json:"use_tls" env:"IMAP_USE_TLS" envDefault:"true"`

	MarkAsSeen       bool   `json:"mark_as_seen" env:"IMAP_MARK_AS_SEEN" envDefault:"true"`
	ProcessedMailbox string `json:"processed_mailbox" env:"IMAP_PROCESSED_MAILBOX"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string `json:"path" env:"DATABASE_PATH"`
}

// ServerConfig holds web server configuration
type ServerConfig struct {
	Port int    `json:"port" env:"SERVER_PORT" envDefault:"8080"`
	Host string `json:"host" env:"SERVER_HOST" envDefault:""`
}

func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot determine home directory")
	}
	return filepath.Join(home, ".parse-dmarc/db.sqlite"), nil
}

func fallbackDBPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.New("cannot determine home directory or current working directory")
	}
	return filepath.Join(cwd, ".parse-dmarc/db.sqlite"), nil
}

func ensureDBPathExists(dbPath string) error {
	parent := filepath.Dir(dbPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return errors.New("failed to create database directory at " + parent + ": " + err.Error() + " - ensure the path is writable or set DATABASE_PATH environment variable")
	}
	return nil
}

// Load loads configuration from a JSON file
func Load(path string) (*Config, error) {
	var cfg Config
	var err error

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file %s: %w", path, err)
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config file %s: %w", path, err)
		}
	}

	if cfg.IMAP.Port == 0 {
		cfg.IMAP.Port = 993
	}
	if cfg.IMAP.Mailbox == "" {
		cfg.IMAP.Mailbox = "INBOX"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path, err = defaultDBPath()
		if err != nil || ensureDBPathExists(cfg.Database.Path) != nil {
			cfg.Database.Path, err = fallbackDBPath()
			if err != nil {
				return nil, fmt.Errorf("resolve database path: %w", err)
			}
			err = ensureDBPathExists(cfg.Database.Path)
			if err != nil {
				return nil, fmt.Errorf("ensure database path: %w", err)
			}
		}
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// Auto-enable Graph if sufficient Graph configuration is provided.
	// This makes the application prefer Graph over IMAP when Graph creds
	// are present in env or the config file, without requiring the
	// explicit "enabled" flag.
	if cfg.Graph.TenantID != "" && cfg.Graph.ClientID != "" {
		hasAuth := cfg.Graph.ClientSecret != "" || (cfg.Graph.CertPath != "" && cfg.Graph.CertKeyPath != "")
		if hasAuth {
			cfg.Graph.Enabled = true
		}
	}

	return &cfg, nil
}

// Validate checks that all required configuration values are set.
// Either IMAP (host, username, password) or Graph (tenant_id, client_id, client_secret/cert, mailbox) must be configured.
// Returns nil if valid, or an error describing the missing configuration.
func (c *Config) Validate() error {
	// Check if Graph is enabled
	if c.Graph.Enabled {
		if c.Graph.TenantID == "" {
			return errors.New("GRAPH_TENANT_ID is required when Graph is enabled")
		}
		if c.Graph.ClientID == "" {
			return errors.New("GRAPH_CLIENT_ID is required when Graph is enabled")
		}
		if c.Graph.ClientSecret == "" && (c.Graph.CertPath == "" || c.Graph.CertKeyPath == "") {
			return errors.New("GRAPH_CLIENT_SECRET or GRAPH_CERT_PATH/GRAPH_CERT_KEY_PATH is required when Graph is enabled")
		}
		if c.Graph.Mailbox == "" {
			return errors.New("GRAPH_MAILBOX is required when Graph is enabled")
		}
		return nil
	}

	// Otherwise, IMAP must be configured
	if c.IMAP.Host == "" {
		return ErrMissingIMAPHost
	}
	if c.IMAP.Username == "" {
		return ErrMissingIMAPUsername
	}
	if c.IMAP.Password == "" {
		return ErrMissingIMAPPassword
	}
	return nil
}

// GenerateSample creates a sample configuration file
func GenerateSample(path string) error {
	dbPath, err := defaultDBPath()
	if err != nil {
		return fmt.Errorf("resolve default database path: %w", err)
	}
	sample := Config{
		LogLevel: "info",
		IMAP: IMAPConfig{
			Host:     "imap.example.com",
			Port:     993,
			Username: "your-email@example.com",
			Password: "your-password",
			Mailbox:  "INBOX",
			UseTLS:   true,

			MarkAsSeen:       true,
			ProcessedMailbox: "",
		},
		Database: DatabaseConfig{
			Path: dbPath,
		},
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
	}

	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sample config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}

	return nil
}
