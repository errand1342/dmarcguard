package graph

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Message represents a Microsoft Graph mail message
type Message struct {
	ID               string       `json:"id"`
	Subject          string       `json:"subject"`
	From             From         `json:"from"`
	ReceivedDateTime time.Time    `json:"receivedDateTime"`
	IsRead           bool         `json:"isRead"`
	HasAttachments   bool         `json:"hasAttachments"`
	BodyPreview      string       `json:"bodyPreview"`
	Body             ItemBody     `json:"body"`
	Attachments      []Attachment `json:"attachments,omitempty"`
}

type From struct {
	EmailAddress EmailAddress `json:"emailAddress"`
}

type EmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type ItemBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// Attachment represents an email attachment
type Attachment struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	IsInline     bool   `json:"isInline"`
	ContentBytes string `json:"contentBytes"`
}

// MessagePage represents paginated messages from Graph API
type MessagePage struct {
	Value    []Message `json:"value"`
	NextLink string    `json:"@odata.nextLink"`
}

// Client implements email fetching via Microsoft Graph API
type Client struct {
	cfg  *Config
	auth *Authenticator
	log  *zerolog.Logger
	mu   sync.Mutex
}

// NewClient creates a new Microsoft Graph client
func NewClient(cfg *Config, log *zerolog.Logger) (*Client, error) {
	if !cfg.Enabled {
		return nil, NewClientError("Graph API is not enabled")
	}

	if err := cfg.ValidateGraph(); err != nil {
		return nil, err
	}

	return &Client{
		cfg:  cfg,
		auth: NewAuthenticator(cfg),
		log:  log,
	}, nil
}

// FetchReports fetches DMARC reports from the mailbox
// Returns a list of email bodies/attachments containing DMARC reports
func (c *Client) FetchReports() ([][]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get folder ID for the specified folder path
	folderID, err := c.getFolderID(c.cfg.FolderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %w", err)
	}

	// Query for unread messages with attachments (potential DMARC reports)
	params := url.Values{}
	params.Set("$filter", "hasAttachments eq true")
	params.Set("$top", "50")

	query := fmt.Sprintf("/users/%s/mailFolders/%s/messages?%s",
		url.PathEscape(c.cfg.Mailbox), url.PathEscape(folderID), params.Encode())

	respBody, err := c.auth.MakeRequest("GET", query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var page MessagePage
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	var reports [][]byte

	for _, msg := range page.Value {
		// Skip already read messages to avoid reprocessing
		if msg.IsRead {
			continue
		}

		c.log.Debug().
			Str("msg_id", msg.ID).
			Str("subject", msg.Subject).
			Msg("processing message")

		// Fetch message with full details and attachments
		fullMsg, err := c.getMessageWithAttachments(msg.ID)
		if err != nil {
			c.log.Error().Err(err).Str("msg_id", msg.ID).Msg("failed to get message attachments")
			continue
		}

		// Extract DMARC report data (attachments or body)
		reportData, err := c.extractDMARCData(fullMsg)
		if err != nil {
			c.log.Debug().Err(err).Str("msg_id", msg.ID).Msg("no DMARC data found in message")
			continue
		}

		if reportData != nil {
			reports = append(reports, reportData)

			// Mark as read if configured
			if c.cfg.MarkAsRead {
				if err := c.markMessageAsRead(msg.ID); err != nil {
					c.log.Error().Err(err).Str("msg_id", msg.ID).Msg("failed to mark message as read")
				}
			}
		}
	}

	return reports, nil
}

// getFolderID retrieves the folder ID for a given folder path
func (c *Client) getFolderID(folderPath string) (string, error) {
	// Common folder aliases
	aliases := map[string]string{
		"INBOX":  "inbox",
		"DRAFTS": "drafts",
		"SENT":   "sentitems",
		"TRASH":  "deleteditems",
		"SPAM":   "junkemail",
	}

	// Check if it's an alias
	if folderID, ok := aliases[folderPath]; ok {
		return folderID, nil
	}

	// Otherwise, search for the folder by name
	params := url.Values{}
	params.Set("$filter", fmt.Sprintf("displayName eq '%s'", folderPath))

	query := fmt.Sprintf("/users/%s/mailFolders?%s",
		url.PathEscape(c.cfg.Mailbox), params.Encode())

	respBody, err := c.auth.MakeRequest("GET", query, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if len(result.Value) == 0 {
		return "", fmt.Errorf("folder not found: %s", folderPath)
	}

	return result.Value[0].ID, nil
}

// getMessageWithAttachments fetches a message with all details and attachments
func (c *Client) getMessageWithAttachments(messageID string) (*Message, error) {
	query := fmt.Sprintf("/users/%s/messages/%s", url.PathEscape(c.cfg.Mailbox), url.PathEscape(messageID))

	respBody, err := c.auth.MakeRequest("GET", query, nil)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(respBody, &msg); err != nil {
		return nil, err
	}

	// If it has attachments, fetch them
	if msg.HasAttachments {
		attachmentQuery := fmt.Sprintf("/users/%s/messages/%s/attachments", url.PathEscape(c.cfg.Mailbox), url.PathEscape(messageID))
		attachResp, err := c.auth.MakeRequest("GET", attachmentQuery, nil)
		if err != nil {
			c.log.Error().Err(err).Str("msg_id", messageID).Msg("failed to fetch attachments")
		} else {
			var attachPage struct {
				Value []Attachment `json:"value"`
			}
			if err := json.Unmarshal(attachResp, &attachPage); err == nil {
				msg.Attachments = attachPage.Value
			}
		}
	}

	return &msg, nil
}

// extractDMARCData extracts DMARC report content from a message
func (c *Client) extractDMARCData(msg *Message) ([]byte, error) {
	// Check attachments first (common for DMARC reports)
	for _, attach := range msg.Attachments {
		if isDMARCFile(attach.Name, attach.ContentType) {
			// Decode base64 content
			data, err := base64.StdEncoding.DecodeString(attach.ContentBytes)
			if err != nil {
				c.log.Error().Err(err).Str("attachment", attach.Name).Msg("failed to decode attachment")
				continue
			}
			return data, nil
		}
	}

	// Check email body as fallback (some senders embed XML in body)
	if strings.Contains(msg.Subject, "DMARC") || strings.Contains(msg.Subject, "dmarc") {
		if msg.Body.ContentType == "text/html" || msg.Body.ContentType == "text/plain" {
			// The body might contain embedded XML or links, but typically DMARC uses attachments
			// Return the body as-is if it looks like XML
			if strings.Contains(msg.Body.Content, "<?xml") {
				return []byte(msg.Body.Content), nil
			}
		}
	}

	return nil, fmt.Errorf("no DMARC data found")
}

// markMessageAsRead marks a message as read
func (c *Client) markMessageAsRead(messageID string) error {
	query := fmt.Sprintf("/users/%s/messages/%s", c.cfg.Mailbox, messageID)

	payload := map[string]bool{
		"isRead": true,
	}

	_, err := c.auth.MakeRequest("PATCH", query, payload)
	return err
}

// Helper to check if file is likely a DMARC report
func isDMARCFile(name, contentType string) bool {
	name = strings.ToLower(name)

	// Check filename
	if strings.Contains(name, "dmarc") || strings.Contains(name, "report") {
		// Check if it's a common DMARC format
		return strings.HasSuffix(name, ".xml") ||
			strings.HasSuffix(name, ".xml.gz") ||
			strings.HasSuffix(name, ".gz") ||
			strings.HasSuffix(name, ".zip")
	}

	// Check content type
	if contentType == "application/xml" ||
		contentType == "text/xml" ||
		contentType == "application/gzip" ||
		contentType == "application/zip" {
		return true
	}

	return false
}
