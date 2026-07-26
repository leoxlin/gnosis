// Package memory provides scoped agent memory backed by Mem0 or a gnosis vault.
package memory

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	mem0 "github.com/mem0ai/mem0-go"
	"gnosis/internal/search"
	"gnosis/internal/vault"
	"go.yaml.in/yaml/v4"
)

const (
	EnvAPIKey   = "GNOSIS_MEMORY_API_KEY"
	EnvUserID   = "GNOSIS_MEMORY_USER_ID"
	EnvAgentID  = "GNOSIS_MEMORY_AGENT_ID"
	EnvProvider = "GNOSIS_MEMORY_PROVIDER"
	EnvBaseURL  = "GNOSIS_MEMORY_BASE_URL"

	ProviderHosted     = "hosted"
	ProviderSelfHosted = "self-hosted"

	BackendMem0  = "mem0"
	BackendVault = "vault"

	DefaultSearchLimit = 5
	MaxSearchLimit     = 20
)

// Config selects one fixed memory identity scope and optional external backend.
type Config struct {
	APIKey   string
	UserID   string
	AgentID  string
	Provider string
	BaseURL  string
}

// Record is the compact backend-neutral form of a memory.
type Record struct {
	ID        string         `json:"id"`
	Text      string         `json:"text"`
	Event     string         `json:"event,omitempty"`
	Score     *float64       `json:"score,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Backend   string         `json:"backend"`
	Origin    *vault.Origin  `json:"origin,omitempty"`
}

// Result contains bounded memory records.
type Result struct {
	Count    int      `json:"count"`
	Memories []Record `json:"memories"`
}

type backend interface {
	add(context.Context, string) ([]Record, error)
	search(context.Context, string, int) ([]Record, error)
}

// Service performs memory operations within one configured identity scope.
type Service struct {
	backend backend
}

// NewFromEnv selects a memory backend from GNOSIS_MEMORY_*.
func NewFromEnv(vaultPath string) (*Service, error) {
	return New(Config{
		APIKey:   os.Getenv(EnvAPIKey),
		UserID:   os.Getenv(EnvUserID),
		AgentID:  os.Getenv(EnvAgentID),
		Provider: os.Getenv(EnvProvider),
		BaseURL:  os.Getenv(EnvBaseURL),
	}, vaultPath)
}

// New validates config and constructs exactly one backend without network I/O.
func New(config Config, vaultPath string) (*Service, error) {
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.UserID = strings.TrimSpace(config.UserID)
	config.AgentID = strings.TrimSpace(config.AgentID)
	config.Provider = strings.TrimSpace(config.Provider)
	config.BaseURL = strings.TrimSpace(config.BaseURL)

	for _, required := range []struct {
		name  string
		value string
	}{
		{name: EnvUserID, value: config.UserID},
		{name: EnvAgentID, value: config.AgentID},
	} {
		if required.value == "" {
			return nil, fmt.Errorf("memory config: %s must not be empty", required.name)
		}
	}

	if config.APIKey == "" && config.Provider == "" && config.BaseURL == "" {
		return &Service{backend: &vaultBackend{
			root:    vaultPath,
			userID:  config.UserID,
			agentID: config.AgentID,
			now:     time.Now,
		}}, nil
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("memory config: %s must not be empty when external memory is configured", EnvAPIKey)
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
	return &Service{backend: &mem0Backend{
		client: client,
		entity: mem0.EntityOptions{UserID: config.UserID, AgentID: config.AgentID},
	}}, nil
}

// Add stores one explicit user-authored memory.
func (s *Service) Add(ctx context.Context, text string) (Result, error) {
	if strings.TrimSpace(text) == "" {
		return Result{}, fmt.Errorf("add memory: text must not be empty")
	}
	records, err := s.backend.add(ctx, text)
	if err != nil {
		return Result{}, err
	}
	return Result{Count: len(records), Memories: records}, nil
}

// Search retrieves bounded memories from the fixed identity scope.
func (s *Service) Search(ctx context.Context, query string, limit *int) (Result, error) {
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
	records, err := s.backend.search(ctx, query, top)
	if err != nil {
		return Result{}, err
	}
	if len(records) > top {
		records = records[:top]
	}
	return Result{Count: len(records), Memories: records}, nil
}

type mem0Backend struct {
	client *mem0.Client
	entity mem0.EntityOptions
}

func (b *mem0Backend) add(ctx context.Context, text string) ([]Record, error) {
	infer := true
	memories, err := b.client.Add(
		ctx,
		[]mem0.Message{{Role: mem0.RoleUser, Content: text}},
		mem0.AddOptions{EntityOptions: b.entity, Infer: &infer},
	)
	if err != nil {
		return nil, requestError("add memory", err)
	}
	return compactMem0(memories), nil
}

func (b *mem0Backend) search(ctx context.Context, query string, limit int) ([]Record, error) {
	result, err := b.client.Search(ctx, query, mem0.SearchOptions{
		Filters: map[string]any{
			"user_id":  b.entity.UserID,
			"agent_id": b.entity.AgentID,
		},
		TopK: &limit,
	})
	if err != nil {
		return nil, requestError("search memory", err)
	}
	return compactMem0(result.Results), nil
}

func compactMem0(memories []mem0.Memory) []Record {
	records := make([]Record, 0, len(memories))
	for _, item := range memories {
		text := item.Memory
		if text == "" && item.Data != nil {
			text = item.Data.Memory
		}
		records = append(records, Record{
			ID:        item.ID,
			Text:      text,
			Event:     string(item.Event),
			Score:     item.Score,
			Metadata:  item.Metadata,
			CreatedAt: formatTime(item.CreatedAt),
			UpdatedAt: formatTime(item.UpdatedAt),
			Backend:   BackendMem0,
		})
	}
	return records
}

type vaultBackend struct {
	root    string
	userID  string
	agentID string
	now     func() time.Time
}

type memoryPage struct {
	Type        string         `yaml:"type"`
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Scope       string         `yaml:"scope"`
	UserID      string         `yaml:"user_id"`
	AgentID     string         `yaml:"agent_id"`
	Source      string         `yaml:"source"`
	ObservedAt  string         `yaml:"observed_at"`
	CreatedAt   string         `yaml:"created_at"`
	UpdatedAt   string         `yaml:"updated_at"`
	Hash        string         `yaml:"hash"`
	Metadata    map[string]any `yaml:"metadata,omitempty"`
	Status      string         `yaml:"status"`
}

func (b *vaultBackend) add(_ context.Context, text string) ([]Record, error) {
	statementHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	existing, err := b.memories()
	if err != nil {
		return nil, fmt.Errorf("add memory: load vault: %w", err)
	}
	for _, document := range existing {
		if metadataString(document.Metadata, "hash") == statementHash {
			record := recordFromDocument(document, nil)
			record.Event = "NOOP"
			return []Record{record}, nil
		}
	}

	scopeHash := sha256.Sum256([]byte(b.userID + "\x00" + b.agentID + "\x00" + text))
	uri := fmt.Sprintf("gnosis://_/memories/%x.md", scopeHash)
	timestamp := b.now().UTC().Truncate(time.Second).Format(time.RFC3339)
	page := memoryPage{
		Type:        "Memory",
		Title:       "Memory " + statementHash[:12],
		Description: firstLine(text),
		Scope:       "user",
		UserID:      b.userID,
		AgentID:     b.agentID,
		Source:      "gnosis-memory-api",
		ObservedAt:  timestamp,
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
		Hash:        statementHash,
		Status:      "active",
	}
	frontmatter, err := yaml.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("add memory: encode page: %w", err)
	}
	content := append([]byte("---\n"), frontmatter...)
	content = append(content, []byte("---\n\n# Memory\n\n"+text+"\n")...)
	if _, err := vault.WriteDocument(b.root, uri, content, false); err != nil {
		return nil, fmt.Errorf("add memory: write vault: %w", err)
	}

	documents, err := b.memories()
	if err != nil {
		return nil, fmt.Errorf("add memory: reload vault: %w", err)
	}
	for _, document := range documents {
		if metadataString(document.Metadata, "hash") == statementHash {
			record := recordFromDocument(document, nil)
			record.Event = "ADD"
			return []Record{record}, nil
		}
	}
	return nil, fmt.Errorf("add memory: written vault record was not readable")
}

func (b *vaultBackend) search(_ context.Context, query string, limit int) ([]Record, error) {
	candidates, err := search.QueryMemoryLexical(b.root, query, b.userID, b.agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("search memory: query vault: %w", err)
	}
	records := make([]Record, 0, len(candidates))
	for _, candidate := range candidates {
		score := candidate.Score
		records = append(records, recordFromDocument(candidate.Document, &score))
	}
	return records, nil
}

func (b *vaultBackend) memories() ([]vault.Document, error) {
	documents, err := vault.LoadDocuments(b.root)
	if err != nil {
		return nil, err
	}
	memories := make([]vault.Document, 0, len(documents))
	for _, document := range documents {
		if document.Type == "Memory" &&
			metadataString(document.Metadata, "status") == "active" &&
			metadataString(document.Metadata, "user_id") == b.userID &&
			metadataString(document.Metadata, "agent_id") == b.agentID {
			memories = append(memories, document)
		}
	}
	return memories, nil
}

func recordFromDocument(document vault.Document, score *float64) Record {
	origin := document.Origin
	metadata, _ := document.Metadata["metadata"].(map[string]any)
	return Record{
		ID:        document.URI,
		Text:      memoryText(document.Body),
		Score:     score,
		Metadata:  metadata,
		CreatedAt: metadataString(document.Metadata, "created_at"),
		UpdatedAt: metadataString(document.Metadata, "updated_at"),
		Backend:   BackendVault,
		Origin:    &origin,
	}
}

func memoryText(body string) string {
	body = strings.TrimSpace(body)
	if rest, found := strings.CutPrefix(body, "# Memory"); found {
		return strings.TrimSpace(rest)
	}
	return body
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	runes := []rune(strings.TrimSpace(line))
	if len(runes) > 120 {
		runes = runes[:120]
	}
	return string(runes)
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
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
