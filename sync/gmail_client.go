// ABOUTME: Google Gmail API client for email sync
// ABOUTME: Creates authenticated Gmail API service for fetching high-signal emails
package sync

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// NewGmailClient creates a new Google Gmail API client
func NewGmailClient(token *oauth2.Token) (*gmail.Service, error) {
	if token == nil {
		return nil, fmt.Errorf("token cannot be nil")
	}

	config := NewOAuthConfig()
	client := config.Client(context.Background(), token)

	service, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}
