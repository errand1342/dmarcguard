package graph

import (
	"os"
	"strconv"
)

// Config holds Microsoft Graph API configuration
type Config struct {
	Enabled      bool   `json:"enabled"`
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	CertPath     string `json:"cert_path"`     // Optional: for certificate auth
	CertKeyPath  string `json:"cert_key_path"` // Optional: for certificate auth
	Mailbox      string `json:"mailbox"`       // e.g., dmarc-reports@example.com
	FolderPath   string `json:"folder_path"`   // e.g., INBOX or INBOX/DMARC
	MarkAsRead   bool   `json:"mark_as_read"`
}

// LoadFromEnv loads Graph configuration from environment variables
func (c *Config) LoadFromEnv() {
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

// Validate checks if required fields are set
func (c *Config) Validate() error {
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
			return NewConfigError(k)
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
