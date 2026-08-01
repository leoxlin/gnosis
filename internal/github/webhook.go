package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gnosis/internal/evidence"
	"gnosis/internal/vault"
)

type WebhookResult struct {
	Created   int `json:"created"`
	Unchanged int `json:"unchanged"`
	Rejected  int `json:"rejected"`
}

// Webhook returns the opt-in GitHub delivery handler.
func Webhook(vaultPath string) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		repository := strings.ToLower(request.PathValue("owner") + "/" + request.PathValue("repository"))
		vaultName, config, err := vault.GitHubRepositoryConfig(vaultPath, repository)
		if err != nil {
			writeWebhookError(response, http.StatusNotFound, err)
			return
		}
		secret := os.Getenv(config.WebhookSecretEnv)
		if config.WebhookSecretEnv == "" || secret == "" {
			writeWebhookError(response, http.StatusServiceUnavailable, fmt.Errorf("github webhook secret is not configured"))
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, config.MaxBodyBytes))
		if err != nil {
			writeWebhookError(response, http.StatusRequestEntityTooLarge, fmt.Errorf("github webhook body exceeds %d bytes", config.MaxBodyBytes))
			return
		}
		if !validSignature(body, request.Header.Get("X-Hub-Signature-256"), secret) {
			writeWebhookError(response, http.StatusUnauthorized, fmt.Errorf("invalid github webhook signature"))
			return
		}
		objects, visibility, payloadRepository, err := webhookObjects(request.Header.Get("X-GitHub-Event"), body)
		if err != nil {
			writeWebhookJSON(response, http.StatusAccepted, WebhookResult{Rejected: 1})
			return
		}
		if !strings.EqualFold(payloadRepository, repository) {
			writeWebhookError(response, http.StatusBadRequest, fmt.Errorf("github webhook repository does not match route"))
			return
		}
		deliveryID := strings.TrimSpace(request.Header.Get("X-GitHub-Delivery"))
		store, err := evidence.New(config.EvidenceDir)
		if err != nil {
			writeWebhookError(response, http.StatusInternalServerError, err)
			return
		}
		sum := sha256.Sum256(body)
		payloadDigest := hex.EncodeToString(sum[:])
		deliveryStatus, err := store.CheckDelivery(deliveryID, payloadDigest)
		if err != nil {
			writeWebhookError(response, http.StatusConflict, err)
			return
		}
		if deliveryStatus == evidence.StatusUnchanged {
			writeWebhookJSON(response, http.StatusOK, WebhookResult{Unchanged: 1})
			return
		}

		result := WebhookResult{}
		now := time.Now().UTC()
		clientConfig := Config{Vault: vaultName, Repository: repository}
		for _, object := range objects {
			input, _, err := normalize(object.kind, object.raw, now, visibility, clientConfig)
			if err != nil {
				result.Rejected++
				continue
			}
			input.DeliveryID = deliveryID
			recorded, err := store.Record(input)
			if err != nil {
				writeWebhookError(response, http.StatusInternalServerError, err)
				return
			}
			if recorded.Status == evidence.StatusCreated {
				result.Created++
			} else {
				result.Unchanged++
			}
		}
		if _, err := store.ClaimDelivery(deliveryID, payloadDigest); err != nil {
			writeWebhookError(response, http.StatusConflict, err)
			return
		}
		writeWebhookJSON(response, http.StatusOK, result)
	}
}

type webhookObject struct {
	kind string
	raw  json.RawMessage
}

func webhookObjects(event string, body []byte) ([]webhookObject, string, string, error) {
	var envelope struct {
		Repository struct {
			Visibility string `json:"visibility"`
			Private    bool   `json:"private"`
			FullName   string `json:"full_name"`
		} `json:"repository"`
		PullRequest json.RawMessage   `json:"pull_request"`
		Issue       json.RawMessage   `json:"issue"`
		Review      json.RawMessage   `json:"review"`
		Comment     json.RawMessage   `json:"comment"`
		Commits     []json.RawMessage `json:"commits"`
		HeadCommit  json.RawMessage   `json:"head_commit"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, "", "", err
	}
	visibility := envelope.Repository.Visibility
	if visibility == "" {
		if envelope.Repository.Private {
			visibility = "private"
		} else {
			visibility = "public"
		}
	}
	var objects []webhookObject
	switch event {
	case "pull_request":
		objects = appendObject(objects, "pull_request", envelope.PullRequest)
	case "issues":
		objects = appendObject(objects, "issue", envelope.Issue)
	case "pull_request_review":
		objects = appendObject(objects, "review", envelope.Review)
	case "issue_comment":
		objects = appendObject(objects, "issue_comment", envelope.Comment)
	case "pull_request_review_comment":
		objects = appendObject(objects, "review_comment", envelope.Comment)
	case "push":
		for _, commit := range envelope.Commits {
			objects = appendObject(objects, "commit", commit)
		}
		if len(objects) == 0 {
			objects = appendObject(objects, "commit", envelope.HeadCommit)
		}
	default:
		return nil, visibility, envelope.Repository.FullName, fmt.Errorf("unsupported github event %q", event)
	}
	if len(objects) == 0 {
		return nil, visibility, envelope.Repository.FullName, fmt.Errorf("github event %q has no supported object", event)
	}
	return objects, visibility, envelope.Repository.FullName, nil
}

func appendObject(objects []webhookObject, kind string, raw json.RawMessage) []webhookObject {
	if len(raw) > 0 && string(raw) != "null" {
		return append(objects, webhookObject{kind: kind, raw: raw})
	}
	return objects
}

func validSignature(body []byte, signature, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func writeWebhookJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	json.NewEncoder(response).Encode(value)
}

func writeWebhookError(response http.ResponseWriter, status int, err error) {
	writeWebhookJSON(response, status, map[string]string{"error": err.Error()})
}
