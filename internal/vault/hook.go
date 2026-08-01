package vault

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	hookDefaultTimeout = 10 * time.Second
	hookMaxTimeout     = 60 * time.Second
	hookOutputLimit    = 4096
)

// HookConfig declares one post-write command or webhook.
type HookConfig struct {
	Name      string   `toml:"name"`
	Kind      string   `toml:"kind"`
	Scope     string   `toml:"scope"`
	Target    string   `toml:"target"`
	Timeout   string   `toml:"timeout"`
	Command   []string `toml:"command"`
	URL       string   `toml:"url"`
	SecretEnv string   `toml:"secret_env"`
}

// VaultWriteEvent is the bounded event delivered after an authoritative write.
type VaultWriteEvent struct {
	Version         int     `json:"version"`
	ID              string  `json:"id"`
	Vault           string  `json:"vault"`
	URI             string  `json:"uri"`
	Operation       string  `json:"operation"`
	PriorRevision   string  `json:"prior_revision,omitempty"`
	NewRevision     string  `json:"new_revision"`
	Origin          Origin  `json:"origin"`
	OccurredAt      string  `json:"occurred_at"`
	KnowledgeChange *string `json:"knowledge_change,omitempty"`
}

// HookDeliveryResult reports one bounded post-commit delivery attempt.
type HookDeliveryResult struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

func validateHooks(hooks []HookConfig, vaultName string) error {
	names := make(map[string]struct{}, len(hooks))
	for i, hook := range hooks {
		prefix := fmt.Sprintf("hooks[%d]", i)
		if strings.TrimSpace(hook.Name) == "" {
			return fmt.Errorf("%s.name must not be empty", prefix)
		}
		if _, exists := names[hook.Name]; exists {
			return fmt.Errorf("%s.name %q is duplicated", prefix, hook.Name)
		}
		names[hook.Name] = struct{}{}
		if hook.Name != strings.TrimSpace(hook.Name) {
			return fmt.Errorf("%s.name must not have surrounding whitespace", prefix)
		}
		switch hook.Scope {
		case "vault":
			if hook.Target != "" {
				return fmt.Errorf("%s.target must be empty for vault scope", prefix)
			}
		case "page", "prefix":
			targetVault, _, ok := canonicalGnosisParts(hook.Target)
			if !ok || targetVault != vaultName {
				return fmt.Errorf("%s.target must be a canonical URI in vault %q", prefix, vaultName)
			}
		default:
			return fmt.Errorf("%s.scope must be %q, %q, or %q", prefix, "vault", "page", "prefix")
		}
		if _, err := hook.timeout(); err != nil {
			return fmt.Errorf("%s.timeout: %w", prefix, err)
		}
		switch hook.Kind {
		case "command":
			if len(hook.Command) == 0 || strings.TrimSpace(hook.Command[0]) == "" {
				return fmt.Errorf("%s.command must contain an executable", prefix)
			}
			if hook.URL != "" || hook.SecretEnv != "" {
				return fmt.Errorf("%s command hook must not set url or secret_env", prefix)
			}
		case "webhook":
			if len(hook.Command) != 0 {
				return fmt.Errorf("%s webhook must not set command", prefix)
			}
			if err := validateWebhookURL(hook.URL); err != nil {
				return fmt.Errorf("%s.url: %w", prefix, err)
			}
			if hook.SecretEnv != "" && !validEnvironmentName(hook.SecretEnv) {
				return fmt.Errorf("%s.secret_env must be an environment variable name", prefix)
			}
		default:
			return fmt.Errorf("%s.kind must be %q or %q", prefix, "command", "webhook")
		}
	}
	return nil
}

func (hook HookConfig) timeout() (time.Duration, error) {
	if hook.Timeout == "" {
		return hookDefaultTimeout, nil
	}
	timeout, err := time.ParseDuration(hook.Timeout)
	if err != nil {
		return 0, fmt.Errorf("must be a duration: %w", err)
	}
	if timeout <= 0 || timeout > hookMaxTimeout {
		return 0, fmt.Errorf("must be greater than zero and at most %s", hookMaxTimeout)
	}
	return timeout, nil
}

func validateWebhookURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("must be an absolute HTTPS URL without credentials or a fragment")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if parsed.Scheme == "http" && (host == "localhost" || ip != nil && ip.IsLoopback()) {
		return nil
	}
	return fmt.Errorf("must use HTTPS, except loopback HTTP")
}

func validEnvironmentName(name string) bool {
	for i, r := range name {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return name != ""
}

func newVaultWriteEvent(prepared preparedDocumentWrite, operation, knowledgeChange string, now time.Time) VaultWriteEvent {
	event := VaultWriteEvent{
		Version:     1,
		Vault:       prepared.candidate.document.Origin.Vault,
		URI:         prepared.candidate.document.URI,
		Operation:   operation,
		NewRevision: prepared.candidate.document.Revision,
		Origin:      prepared.candidate.document.Origin,
		OccurredAt:  now.UTC().Format(time.RFC3339Nano),
	}
	if prepared.current != nil {
		event.PriorRevision = prepared.current.document.Revision
	}
	if knowledgeChange != "" {
		event.KnowledgeChange = &knowledgeChange
	}
	identity := strings.Join([]string{event.Vault, event.URI, event.NewRevision, event.Operation}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	event.ID = "sha256:" + hex.EncodeToString(digest[:])
	return event
}

func dispatchHooks(ctx context.Context, hooks []HookConfig, event VaultWriteEvent) []HookDeliveryResult {
	body, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	results := make([]HookDeliveryResult, 0, len(hooks))
	for _, hook := range hooks {
		if !hook.matches(event.URI) {
			continue
		}
		timeout, _ := hook.timeout()
		deliveryContext, cancel := context.WithTimeout(ctx, timeout)
		var result HookDeliveryResult
		if hook.Kind == "command" {
			result = deliverCommand(deliveryContext, hook, body)
		} else {
			result = deliverWebhook(deliveryContext, hook, event, body)
		}
		cancel()
		result.Name = hook.Name
		result.Kind = hook.Kind
		redactDelivery(&result, hooks)
		results = append(results, result)
	}
	return results
}

func (hook HookConfig) matches(uri string) bool {
	switch hook.Scope {
	case "vault":
		return true
	case "page":
		return uri == hook.Target
	case "prefix":
		return strings.HasPrefix(uri, strings.TrimSuffix(hook.Target, "/")+"/")
	default:
		return false
	}
}

func deliverCommand(ctx context.Context, hook HookConfig, body []byte) HookDeliveryResult {
	command := exec.CommandContext(ctx, hook.Command[0], hook.Command[1:]...)
	command.Stdin = bytes.NewReader(body)
	output := &boundedBuffer{limit: hookOutputLimit}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	result := HookDeliveryResult{Status: "success", Output: output.String()}
	if err == nil {
		return result
	}
	result.Status, result.Error = deliveryError(ctx, err)
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		code := exitError.ExitCode()
		result.ExitCode = &code
	}
	return result
}

func deliverWebhook(ctx context.Context, hook HookConfig, event VaultWriteEvent, body []byte) HookDeliveryResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		return HookDeliveryResult{Status: "failed", Error: err.Error()}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Gnosis-Event-Version", strconv.Itoa(event.Version))
	request.Header.Set("X-Gnosis-Event-ID", event.ID)
	if hook.SecretEnv != "" {
		secret := os.Getenv(hook.SecretEnv)
		if secret == "" {
			return HookDeliveryResult{
				Status: "failed",
				Error:  fmt.Sprintf("environment variable %s is empty", hook.SecretEnv),
			}
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		request.Header.Set("X-Gnosis-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(request)
	if err != nil {
		status, message := deliveryError(ctx, err)
		return HookDeliveryResult{Status: status, Error: message}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, hookOutputLimit+1))
	result := HookDeliveryResult{Status: "success", HTTPStatus: response.StatusCode}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		result.Status = "failed"
		result.Error = "HTTP " + response.Status
	}
	return result
}

func deliveryError(ctx context.Context, err error) (string, string) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "timeout", context.DeadlineExceeded.Error()
	case errors.Is(ctx.Err(), context.Canceled):
		return "canceled", context.Canceled.Error()
	default:
		return "failed", err.Error()
	}
}

func redactDelivery(result *HookDeliveryResult, hooks []HookConfig) {
	for _, hook := range hooks {
		if hook.SecretEnv == "" {
			continue
		}
		secret := os.Getenv(hook.SecretEnv)
		if secret == "" {
			continue
		}
		result.Output = strings.ReplaceAll(result.Output, secret, "[REDACTED]")
		result.Error = strings.ReplaceAll(result.Error, secret, "[REDACTED]")
	}
}

type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		_, _ = buffer.buffer.Write(data[:min(len(data), remaining)])
	}
	return len(data), nil
}

func (buffer *boundedBuffer) String() string {
	return buffer.buffer.String()
}
