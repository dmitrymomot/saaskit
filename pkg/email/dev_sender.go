package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DevSender implements EmailSender for local development.
// It saves emails as HTML and JSON files to a specified directory
// instead of sending them through an email service.
type DevSender struct {
	dir string
}

// NewDevSender creates a development email sender that saves emails to disk.
// The directory will be created if it doesn't exist.
func NewDevSender(dir string) EmailSender {
	return &DevSender{dir: dir}
}

// emailMetadata contains the email data saved to JSON (excluding HTML content).
type emailMetadata struct {
	Timestamp string `json:"timestamp"`
	SendTo    string `json:"send_to"`
	Subject   string `json:"subject"`
	Tag       string `json:"tag,omitempty"`
}

// SendEmail saves the email as HTML and metadata as JSON to the configured directory.
func (d *DevSender) SendEmail(ctx context.Context, params SendEmailParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(d.dir, 0755); err != nil {
		return fmt.Errorf("%w: failed to create directory: %v", ErrFailedToSendEmail, err)
	}

	// Generate timestamp and base filename
	now := time.Now()
	timestamp := now.Format("2006_01_02_150405")

	// Use tag if available, otherwise use subject
	identifier := params.Tag
	if identifier == "" {
		identifier = params.Subject
	}

	// Sanitize identifier for filesystem
	safeIdentifier := sanitizeFilename(identifier)
	baseFilename := fmt.Sprintf("%s_%s", timestamp, safeIdentifier)

	// Write HTML file
	htmlPath := filepath.Join(d.dir, baseFilename+".html")
	if err := os.WriteFile(htmlPath, []byte(params.BodyHTML), 0644); err != nil {
		return fmt.Errorf("%w: failed to write HTML file: %v", ErrFailedToSendEmail, err)
	}

	// Prepare metadata
	metadata := emailMetadata{
		Timestamp: now.Format(time.RFC3339),
		SendTo:    params.SendTo,
		Subject:   params.Subject,
		Tag:       params.Tag,
	}

	// Write JSON metadata file
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: failed to marshal metadata: %v", ErrFailedToSendEmail, err)
	}

	jsonPath := filepath.Join(d.dir, baseFilename+".json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("%w: failed to write JSON file: %v", ErrFailedToSendEmail, err)
	}

	return nil
}

// sanitizeRegex matches characters that are not alphanumeric, dash, underscore, or dot
var sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)

// sanitizeFilename converts a string into a safe filename.
// It replaces spaces with underscores, removes special characters,
// and truncates to a reasonable length.
func sanitizeFilename(s string) string {
	// Replace spaces with underscores
	s = strings.ReplaceAll(s, " ", "_")

	// Remove unsafe characters
	s = sanitizeRegex.ReplaceAllString(s, "")

	// Truncate if too long (keep it reasonable for filesystems)
	const maxLength = 100
	if len(s) > maxLength {
		s = s[:maxLength]
	}

	// Handle empty result
	if s == "" {
		s = "email"
	}

	return strings.ToLower(s)
}
