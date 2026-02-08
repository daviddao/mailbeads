// Package gmail provides native Go Gmail API operations for mailbeads.
//
// It replaces the Python scripts (search_emails.py, read_email.py) with
// direct Go API calls using google.golang.org/api/gmail/v1.
package gmail

import (
	"encoding/base64"
	"fmt"
	"strings"

	gm "google.golang.org/api/gmail/v1"
)

// MessageSummary matches the JSON output of search_emails.py.
type MessageSummary struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
}

// FullMessage matches the JSON output of read_email.py with --format full.
type FullMessage struct {
	ID        string   `json:"id"`
	ThreadID  string   `json:"thread_id"`
	MessageID string   `json:"message_id,omitempty"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	CC        string   `json:"cc,omitempty"`
	Subject   string   `json:"subject"`
	Date      string   `json:"date"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels,omitempty"`
	Snippet   string   `json:"snippet,omitempty"`
}

// AttachmentInfo holds metadata about a message attachment.
type AttachmentInfo struct {
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	AttachmentID string `json:"attachment_id,omitempty"`
}

// FullMessageWithAttachments extends FullMessage with attachment and size info.
type FullMessageWithAttachments struct {
	FullMessage
	Attachments  []AttachmentInfo `json:"attachments,omitempty"`
	SizeEstimate int64            `json:"size_estimate,omitempty"`
}

// Search finds messages matching a Gmail query and returns summaries.
// This replaces search_emails.py.
func Search(svc *gm.Service, query string, maxResults int64) ([]MessageSummary, error) {
	resp, err := svc.Users.Messages.List("me").
		Q(query).
		MaxResults(maxResults).
		Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	if len(resp.Messages) == 0 {
		return nil, nil
	}

	summaries := make([]MessageSummary, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		detail, err := svc.Users.Messages.Get("me", msg.Id).
			Format("metadata").
			MetadataHeaders("From", "To", "Subject", "Date").
			Do()
		if err != nil {
			// Skip individual message failures.
			continue
		}

		headers := headerMap(detail.Payload.Headers)
		summaries = append(summaries, MessageSummary{
			ID:       detail.Id,
			ThreadID: detail.ThreadId,
			From:     headers["From"],
			To:       headers["To"],
			Subject:  defaultStr(headers["Subject"], "(no subject)"),
			Date:     headers["Date"],
			Snippet:  detail.Snippet,
		})
	}

	return summaries, nil
}

// ReadFull fetches a complete message by ID, decoding the body.
// This replaces read_email.py.
func ReadFull(svc *gm.Service, messageID string) (*FullMessage, error) {
	msg, err := svc.Users.Messages.Get("me", messageID).
		Format("full").
		Do()
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", messageID, err)
	}

	headers := headerMap(msg.Payload.Headers)

	return &FullMessage{
		ID:        msg.Id,
		ThreadID:  msg.ThreadId,
		MessageID: headers["Message-ID"],
		From:      headers["From"],
		To:        headers["To"],
		CC:        headers["Cc"],
		Subject:   defaultStr(headers["Subject"], "(no subject)"),
		Date:      headers["Date"],
		Body:      extractBody(msg.Payload),
		Labels:    msg.LabelIds,
		Snippet:   msg.Snippet,
	}, nil
}

// ReadFullWithAttachments fetches a complete message including attachment info.
func ReadFullWithAttachments(svc *gm.Service, messageID string) (*FullMessageWithAttachments, error) {
	msg, err := svc.Users.Messages.Get("me", messageID).
		Format("full").
		Do()
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", messageID, err)
	}

	headers := headerMap(msg.Payload.Headers)

	return &FullMessageWithAttachments{
		FullMessage: FullMessage{
			ID:        msg.Id,
			ThreadID:  msg.ThreadId,
			MessageID: headers["Message-ID"],
			From:      headers["From"],
			To:        headers["To"],
			CC:        headers["Cc"],
			Subject:   defaultStr(headers["Subject"], "(no subject)"),
			Date:      headers["Date"],
			Body:      extractBody(msg.Payload),
			Labels:    msg.LabelIds,
			Snippet:   msg.Snippet,
		},
		Attachments:  extractAttachments(msg.Payload),
		SizeEstimate: msg.SizeEstimate,
	}, nil
}

// extractBody gets the plain text body from a message payload.
// Handles multipart messages recursively, preferring text/plain over text/html.
func extractBody(payload *gm.MessagePart) string {
	// Direct body on the payload itself.
	if payload.Body != nil && payload.Body.Data != "" {
		if decoded, err := decodeBase64URL(payload.Body.Data); err == nil {
			return decoded
		}
	}

	// Recurse into parts — prefer text/plain first pass.
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			if decoded, err := decodeBase64URL(part.Body.Data); err == nil {
				return decoded
			}
		}
		// Recurse into nested multipart.
		if len(part.Parts) > 0 {
			if body := extractBody(part); body != "" {
				return body
			}
		}
	}

	// Second pass: fall back to HTML.
	for _, part := range payload.Parts {
		if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
			if decoded, err := decodeBase64URL(part.Body.Data); err == nil {
				return "(HTML content)\n" + decoded
			}
		}
	}

	return "(No readable body found)"
}

// extractAttachments gets attachment metadata from a message payload.
func extractAttachments(payload *gm.MessagePart) []AttachmentInfo {
	var attachments []AttachmentInfo

	var scan func(parts []*gm.MessagePart)
	scan = func(parts []*gm.MessagePart) {
		for _, part := range parts {
			if part.Filename != "" {
				att := AttachmentInfo{
					Filename: part.Filename,
					MimeType: part.MimeType,
				}
				if part.Body != nil {
					att.Size = part.Body.Size
					att.AttachmentID = part.Body.AttachmentId
				}
				attachments = append(attachments, att)
			}
			if len(part.Parts) > 0 {
				scan(part.Parts)
			}
		}
	}

	if len(payload.Parts) > 0 {
		scan(payload.Parts)
	}
	return attachments
}

// headerMap converts Gmail API headers into a simple key-value map.
func headerMap(headers []*gm.MessagePartHeader) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		m[h.Name] = h.Value
	}
	return m
}

// decodeBase64URL decodes Gmail's base64url-encoded content.
func decodeBase64URL(data string) (string, error) {
	// Gmail uses URL-safe base64 without padding.
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")
	// Add padding if needed.
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
