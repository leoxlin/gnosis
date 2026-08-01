// Package github synchronizes configured GitHub repositories into evidence storage.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gnosis/internal/evidence"
	"gnosis/internal/s3store"
	"gnosis/internal/vault"
)

const defaultAPIURL = "https://api.github.com"

type Config struct {
	Vault      string
	Repository string
	Token      string
	APIURL     string
	PerPage    int
	MaxPages   int
}

type Options struct {
	Since     time.Time
	MaxItems  int
	Reconcile bool
}

type Result struct {
	Created     int             `json:"created"`
	Unchanged   int             `json:"unchanged"`
	Tombstoned  int             `json:"tombstoned"`
	Rejected    int             `json:"rejected"`
	RateLimited bool            `json:"rate_limited"`
	RateReset   string          `json:"rate_reset,omitempty"`
	Cursor      evidence.Cursor `json:"cursor"`
}

type Client struct {
	config Config
	http   *http.Client
	store  evidence.Backend
	now    func() time.Time
}

func New(config Config, store evidence.Backend, client *http.Client) (*Client, error) {
	config.Vault = strings.TrimSpace(config.Vault)
	config.Repository = strings.ToLower(strings.TrimSpace(config.Repository))
	config.Token = strings.TrimSpace(config.Token)
	if config.Vault == "" || config.Repository == "" || config.Token == "" {
		return nil, fmt.Errorf("github vault, repository, and token must not be empty")
	}
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}
	if _, err := url.ParseRequestURI(config.APIURL); err != nil {
		return nil, fmt.Errorf("github API URL: %w", err)
	}
	if config.PerPage < 1 || config.PerPage > 100 || config.MaxPages < 1 {
		return nil, fmt.Errorf("github pagination bounds are invalid")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{config: config, http: client, store: store, now: time.Now}, nil
}

func NewConfigured(vaultPath, repository string) (*Client, vault.GitHubConfig, error) {
	vaultName, config, err := vault.GitHubRepositoryConfig(vaultPath, repository)
	if err != nil {
		return nil, vault.GitHubConfig{}, err
	}
	token := strings.TrimSpace(os.Getenv(config.TokenEnv))
	if token == "" {
		return nil, vault.GitHubConfig{}, fmt.Errorf("environment variable %s is empty", config.TokenEnv)
	}
	store, err := configuredEvidenceStore(context.Background(), config)
	if err != nil {
		return nil, vault.GitHubConfig{}, err
	}
	client, err := New(Config{
		Vault: vaultName, Repository: config.Repository, Token: token,
		PerPage: config.PerPage, MaxPages: config.MaxPages,
	}, store, nil)
	return client, config, err
}

func configuredEvidenceStore(ctx context.Context, config vault.GitHubConfig) (evidence.Backend, error) {
	if config.EvidenceBackend == "s3" {
		return evidence.NewS3(ctx, s3store.Config{Bucket: config.S3Bucket, Region: config.S3Region, Prefix: config.S3Prefix})
	}
	return evidence.New(config.EvidenceDir)
}

func (c *Client) Sync(ctx context.Context, options Options) (Result, error) {
	if options.Reconcile && (!options.Since.IsZero() || options.MaxItems > 0) {
		return Result{}, fmt.Errorf("github reconciliation cannot use backfill bounds")
	}
	cursor, err := c.store.LoadCursor(c.config.Repository)
	if err != nil {
		return Result{}, err
	}
	started := c.now().UTC()
	if !options.Reconcile && options.Since.IsZero() && strings.HasPrefix(cursor.Value, "complete:") {
		completed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimPrefix(cursor.Value, "complete:"))
		if parseErr == nil {
			options.Since = completed.Add(-time.Minute)
		}
	}
	result := Result{Cursor: cursor}
	visibility, limited, reset, err := c.repositoryVisibility(ctx)
	if err != nil {
		return result, err
	}
	if limited {
		result.RateLimited, result.RateReset = true, reset
		return result, nil
	}

	seen := map[string]bool{}
	pulls := []int64{}
	endpoints := []struct {
		kind string
		path string
	}{
		{"pull_request", "/pulls?state=all"},
		{"issue", "/issues?state=all"},
		{"review_comment", "/pulls/comments"},
		{"issue_comment", "/issues/comments"},
		{"commit", "/commits"},
	}
	for _, endpoint := range endpoints {
		complete, stop, err := c.syncEndpoint(
			ctx, endpoint.kind, endpoint.path, visibility, options,
			seen, &pulls, &result,
		)
		if err != nil {
			return result, err
		}
		if stop {
			return result, nil
		}
		if !complete {
			return result, nil
		}
	}
	for _, number := range pulls {
		complete, stop, err := c.syncEndpoint(
			ctx, "review", fmt.Sprintf("/pulls/%d/reviews", number), visibility,
			options, seen, nil, &result,
		)
		if err != nil {
			return result, err
		}
		if stop {
			return result, nil
		}
		if !complete {
			return result, nil
		}
	}

	if options.Reconcile && (options.MaxItems == 0 || result.Created+result.Unchanged < options.MaxItems) {
		if err := c.reconcile(seen, visibility, &result); err != nil {
			return result, err
		}
	}
	cursor, err = c.store.CommitCursor(result.Cursor, "complete:"+started.Format(time.RFC3339Nano))
	result.Cursor = cursor
	return result, err
}

func (c *Client) syncEndpoint(
	ctx context.Context,
	kind, path, visibility string,
	options Options,
	seen map[string]bool,
	pulls *[]int64,
	result *Result,
) (complete, stop bool, err error) {
	for page := 1; page <= c.config.MaxPages; page++ {
		raw, limited, reset, err := c.getPage(ctx, path, page)
		if err != nil {
			return false, false, err
		}
		if limited {
			result.RateLimited, result.RateReset = true, reset
			return false, true, nil
		}
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return false, false, fmt.Errorf("decode github %s page: %w", kind, err)
		}
		for _, item := range items {
			normalized, number, err := normalize(kind, item, c.now().UTC(), visibility, c.config)
			if err != nil {
				result.Rejected++
				continue
			}
			if pulls != nil && number > 0 {
				*pulls = append(*pulls, number)
			}
			if !options.Since.IsZero() {
				updated, _ := time.Parse(time.RFC3339Nano, normalized.UpdatedAt)
				if updated.Before(options.Since) {
					continue
				}
			}
			recorded, err := c.store.Record(normalized)
			if err != nil {
				return false, false, err
			}
			seen[kind+"\x00"+normalized.SourceID] = true
			if recorded.Status == evidence.StatusCreated {
				result.Created++
			} else {
				result.Unchanged++
			}
			if options.MaxItems > 0 && result.Created >= options.MaxItems {
				cursor, err := c.store.CommitCursor(result.Cursor, fmt.Sprintf("%s:%d", kind, page))
				result.Cursor = cursor
				return false, true, err
			}
		}
		cursor, err := c.store.CommitCursor(result.Cursor, fmt.Sprintf("%s:%d", kind, page+1))
		result.Cursor = cursor
		if err != nil {
			return false, false, err
		}
		if len(items) < c.config.PerPage {
			return true, false, nil
		}
	}
	return false, false, nil
}

func (c *Client) getPage(ctx context.Context, path string, page int) ([]byte, bool, string, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	target := strings.TrimRight(c.config.APIURL, "/") + "/repos/" + c.config.Repository +
		path + separator + "per_page=" + strconv.Itoa(c.config.PerPage) + "&page=" + strconv.Itoa(page)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, false, "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+c.config.Token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, false, "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		return nil, true, response.Header.Get("X-RateLimit-Reset"), nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, false, "", fmt.Errorf("github %s: %s: %s", path, response.Status, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, evidence.MaxPayloadBytes+1))
	if err != nil {
		return nil, false, "", err
	}
	if len(body) > evidence.MaxPayloadBytes {
		return nil, false, "", fmt.Errorf("github response exceeds %d bytes", evidence.MaxPayloadBytes)
	}
	return body, false, "", nil
}

func (c *Client) repositoryVisibility(ctx context.Context) (string, bool, string, error) {
	target := strings.TrimRight(c.config.APIURL, "/") + "/repos/" + c.config.Repository
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", false, "", err
	}
	request.Header.Set("Authorization", "Bearer "+c.config.Token)
	response, err := c.http.Do(request)
	if err != nil {
		return "", false, "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		return "", true, response.Header.Get("X-RateLimit-Reset"), nil
	}
	if response.StatusCode != http.StatusOK {
		return "", false, "", fmt.Errorf("github repository metadata: %s", response.Status)
	}
	var metadata struct {
		Visibility string `json:"visibility"`
		Private    bool   `json:"private"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&metadata); err != nil {
		return "", false, "", err
	}
	if metadata.Visibility != "" {
		return metadata.Visibility, false, "", nil
	}
	if metadata.Private {
		return "private", false, "", nil
	}
	return "public", false, "", nil
}

func normalize(kind string, raw json.RawMessage, observed time.Time, visibility string, config Config) (evidence.Input, int64, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return evidence.Input{}, 0, err
	}
	sourceID := stringField(object, "node_id")
	if sourceID == "" {
		sourceID = stringField(object, "sha")
	}
	if sourceID == "" {
		sourceID = numberField(object, "id")
	}
	updated := firstString(object, "updated_at", "submitted_at", "created_at")
	if kind == "commit" {
		updated = firstString(object, "timestamp")
		if updated == "" {
			updated = nestedString(object, "commit", "committer", "date")
		}
	}
	if sourceID == "" || updated == "" {
		return evidence.Input{}, 0, fmt.Errorf("github %s lacks stable identity or timestamp", kind)
	}
	if _, err := time.Parse(time.RFC3339Nano, updated); err != nil {
		return evidence.Input{}, 0, err
	}
	number, _ := strconv.ParseInt(numberField(object, "number"), 10, 64)
	return evidence.Input{
		Vault: config.Vault, Repository: config.Repository, Kind: kind,
		SourceID: sourceID, UpdatedAt: updated, ObservedAt: observed.Format(time.RFC3339Nano),
		Visibility: visibility, URL: stringField(object, "html_url"), RawPayload: raw,
	}, number, nil
}

func (c *Client) reconcile(seen map[string]bool, visibility string, result *Result) error {
	latest, err := c.store.Latest(c.config.Vault, c.config.Repository)
	if err != nil {
		return err
	}
	now := c.now().UTC().Format(time.RFC3339Nano)
	for key, record := range latest {
		if seen[key] || record.Tombstone {
			continue
		}
		tombstone, err := c.store.Tombstone(evidence.Input{
			Vault: c.config.Vault, Repository: c.config.Repository, Kind: record.Kind,
			SourceID: record.SourceID, UpdatedAt: now, ObservedAt: now, Visibility: visibility,
		})
		if err != nil {
			return err
		}
		if tombstone.Status == evidence.StatusCreated {
			result.Tombstoned++
		}
	}
	return nil
}

func stringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func numberField(object map[string]any, key string) string {
	switch value := object[key].(type) {
	case json.Number:
		return value.String()
	case string:
		return value
	default:
		return ""
	}
}

func firstString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(object, key); value != "" {
			return value
		}
	}
	return ""
}

func nestedString(object map[string]any, keys ...string) string {
	var current any = object
	for _, key := range keys {
		fields, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = fields[key]
	}
	value, _ := current.(string)
	return value
}
