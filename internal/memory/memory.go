// Package memory provides scoped agent memory backed by Mem0.
package memory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	mem0 "github.com/mem0ai/mem0-go"
)

const (
	EnvAPIKey   = "GNOSIS_MEMORY_API_KEY"
	EnvUserID   = "GNOSIS_MEMORY_USER_ID"
	EnvAgentID  = "GNOSIS_MEMORY_AGENT_ID"
	EnvProvider = "GNOSIS_MEMORY_PROVIDER"
	EnvBaseURL  = "GNOSIS_MEMORY_BASE_URL"

	ProviderHosted     = "hosted"
	ProviderSelfHosted = "self-hosted"

	DefaultSearchLimit = 5
	MaxSearchLimit     = 20
)

// Config selects one fixed Mem0 identity scope.
type Config struct {
	APIKey   string
	UserID   string
	AgentID  string
	Provider string
	BaseURL  string
}

// Client performs memory operations within one configured identity scope.
type Client struct {
	client *mem0.Client
	entity mem0.EntityOptions
}

// Record is the compact agent-facing form of a Mem0 memory.
type Record struct {
	ID       string         `json:"id"`
	Text     string         `json:"text"`
	Event    string         `json:"event,omitempty"`
	Score    *float64       `json:"score,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Result contains bounded memory records.
type Result struct {
	Count    int      `json:"count"`
	Memories []Record `json:"memories"`
}

// NewFromEnv constructs a client from the GNOSIS_MEMORY_* environment.
func NewFromEnv() (*Client, error) {
	return New(Config{
		APIKey:   os.Getenv(EnvAPIKey),
		UserID:   os.Getenv(EnvUserID),
		AgentID:  os.Getenv(EnvAgentID),
		Provider: os.Getenv(EnvProvider),
		BaseURL:  os.Getenv(EnvBaseURL),
	})
}

// New validates config and constructs a client without network I/O.
func New(config Config) (*Client, error) {
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.UserID = strings.TrimSpace(config.UserID)
	config.AgentID = strings.TrimSpace(config.AgentID)
	config.Provider = strings.TrimSpace(config.Provider)
	config.BaseURL = strings.TrimSpace(config.BaseURL)

	for _, required := range []struct {
		name  string
		value string
	}{
		{name: EnvAPIKey, value: config.APIKey},
		{name: EnvUserID, value: config.UserID},
		{name: EnvAgentID, value: config.AgentID},
	} {
		if required.value == "" {
			return nil, fmt.Errorf("memory config: %s must not be empty", required.name)
		}
	}
	if config.Provider == "" {
		config.Provider = ProviderHosted
	}
	if config.Provider != ProviderHosted && config.Provider != ProviderSelfHosted {
		return nil, fmt.Errorf(
			"memory config: %s must be %q or %q",
			EnvProvider,
			ProviderHosted,
			ProviderSelfHosted,
		)
	}
	if config.Provider == ProviderSelfHosted && config.BaseURL == "" {
		return nil, fmt.Errorf(
			"memory config: %s must not be empty for self-hosted memory",
			EnvBaseURL,
		)
	}

	options := make([]mem0.Option, 0, 2)
	if config.BaseURL != "" {
		options = append(options, mem0.WithBaseURL(config.BaseURL))
	}
	if config.Provider == ProviderSelfHosted {
		options = append(options, mem0.WithSelfHosted())
	}
	client, err := mem0.New(config.APIKey, options...)
	if err != nil {
		return nil, fmt.Errorf("memory config: %w", err)
	}
	return &Client{
		client: client,
		entity: mem0.EntityOptions{UserID: config.UserID, AgentID: config.AgentID},
	}, nil
}

// Add stores one explicit user-authored memory.
func (c *Client) Add(ctx context.Context, text string) (Result, error) {
	if strings.TrimSpace(text) == "" {
		return Result{}, fmt.Errorf("add memory: text must not be empty")
	}
	infer := true
	memories, err := c.client.Add(
		ctx,
		[]mem0.Message{{Role: mem0.RoleUser, Content: text}},
		mem0.AddOptions{EntityOptions: c.entity, Infer: &infer},
	)
	if err != nil {
		return Result{}, requestError("add memory", err)
	}
	return compact(memories), nil
}

// Search retrieves bounded memories from the fixed identity scope.
func (c *Client) Search(ctx context.Context, query string, limit *int) (Result, error) {
	if strings.TrimSpace(query) == "" {
		return Result{}, fmt.Errorf("search memory: query must not be empty")
	}
	top := DefaultSearchLimit
	if limit != nil {
		top = *limit
	}
	if top < 1 || top > MaxSearchLimit {
		return Result{}, fmt.Errorf(
			"search memory: limit must be between 1 and %d",
			MaxSearchLimit,
		)
	}

	result, err := c.client.Search(ctx, query, mem0.SearchOptions{
		Filters: map[string]any{
			"user_id":  c.entity.UserID,
			"agent_id": c.entity.AgentID,
		},
		TopK: &top,
	})
	if err != nil {
		return Result{}, requestError("search memory", err)
	}
	if len(result.Results) > top {
		result.Results = result.Results[:top]
	}
	return compact(result.Results), nil
}

func compact(memories []mem0.Memory) Result {
	records := make([]Record, 0, len(memories))
	for _, item := range memories {
		text := item.Memory
		if text == "" && item.Data != nil {
			text = item.Data.Memory
		}
		records = append(records, Record{
			ID:       item.ID,
			Text:     text,
			Event:    string(item.Event),
			Score:    item.Score,
			Metadata: item.Metadata,
		})
	}
	return Result{Count: len(records), Memories: records}
}

func requestError(operation string, err error) error {
	var apiError *mem0.APIError
	if errors.As(err, &apiError) {
		request := ""
		if apiError.RequestID != "" {
			request = fmt.Sprintf(" (request %s)", apiError.RequestID)
		}
		return fmt.Errorf(
			"%s: memory service returned status %d%s",
			operation,
			apiError.StatusCode,
			request,
		)
	}
	return fmt.Errorf("%s: memory service request failed: %w", operation, err)
}
