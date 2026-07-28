package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	evidencecontext "gnosis/internal/evidencecontext"
	"gnosis/internal/vault"
	"gnosis/ui"
	"go.yaml.in/yaml/v4"

	"github.com/adrg/frontmatter"
)

const defaultHTTPAddress = "127.0.0.1:8080"

func newServeHTTPCommand(options *rootOptions) *cobra.Command {
	var address string
	var allowKnowledgeWrites bool
	command := &cobra.Command{
		Use:   "http [flags]",
		Short: "Serve the gnosis API, document UI, and MCP over HTTP",
		Args:  cobra.NoArgs,
		Example: "gnosis serve http --address 127.0.0.1:8080\n" +
			"gnosis --vault <path> serve http\n" +
			"gnosis --vault <path> serve http --allow-knowledge-writes",
		RunE: func(command *cobra.Command, _ []string) error {
			return serveHTTPWithKnowledgeWrites(
				command.Context(),
				address,
				options.vaultPath,
				allowKnowledgeWrites,
				command.ErrOrStderr(),
			)
		},
	}
	command.Flags().StringVar(&address, "address", defaultHTTPAddress, "HTTP listen address")
	command.Flags().BoolVar(
		&allowKnowledgeWrites,
		"allow-knowledge-writes",
		false,
		"register approval-gated general knowledge apply",
	)
	return command
}

func serveHTTP(ctx context.Context, address, vaultPath string, output io.Writer) error {
	return serveHTTPWithKnowledgeWrites(ctx, address, vaultPath, false, output)
}

func serveHTTPWithKnowledgeWrites(
	ctx context.Context,
	address,
	vaultPath string,
	allowKnowledgeWrites bool,
	output io.Writer,
) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("serve http: listen: %w", err)
	}
	server := &http.Server{
		Handler:           newHTTPHandlerWithKnowledgeWrites(vaultPath, allowKnowledgeWrites),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
	}
	fmt.Fprintf(output, "gnosis serving at http://%s\n", listener.Addr())

	exited := make(chan error, 1)
	go func() {
		exited <- server.Serve(listener)
	}()

	select {
	case err := <-exited:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("serve http: shutdown: %w", err)
		}
		if err := <-exited; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	}
}

func newHTTPHandler(vaultPath string) http.Handler {
	return newHTTPHandlerWithKnowledgeWrites(vaultPath, false)
}

func newHTTPHandlerWithKnowledgeWrites(vaultPath string, allowKnowledgeWrites bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", serveUI)
	mux.HandleFunc("GET /api/v1/vaults", serveVaults(vaultPath))
	mux.HandleFunc("GET /api/v1/concepts", serveConcepts(vaultPath))
	mux.HandleFunc("GET /api/v1/pages", servePages(vaultPath))
	mux.HandleFunc("GET /api/v1/page", servePage(vaultPath))
	mux.HandleFunc("GET /api/v1/history", serveHistory(vaultPath))
	mux.HandleFunc("GET /api/v1/diff", serveDiff(vaultPath))
	mux.HandleFunc("GET /api/v1/changes", serveChanges(vaultPath))
	mux.HandleFunc("GET /api/v1/graph", serveGraph(vaultPath))
	mux.HandleFunc("GET /api/v1/search", serveSearch(vaultPath))
	mux.HandleFunc("POST /api/v1/context", serveContext(vaultPath))
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return newMCPServerWithKnowledgeWrites(vaultPath, allowKnowledgeWrites)
	}, nil)
	mux.Handle("/mcp", http.NewCrossOriginProtection().Handler(mcpHandler))
	return mux
}

func serveUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(ui.HTML); err != nil {
		return
	}
}

func serveVaults(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		catalog, err := vault.Vaults(vaultPath)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, catalog)
	}
}

func serveConcepts(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		_, concepts, err := getMCPConcepts(vaultPath, request.URL.Query().Get("type"))
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, concepts)
	}
}

func servePages(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		pages, err := vault.ListPages(vaultPath)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, map[string]any{"pages": pages})
	}
}

func servePage(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		uri := request.URL.Query().Get("uri")
		if !strings.HasPrefix(uri, "gnosis://") {
			writeHTTPError(w, http.StatusBadRequest, errors.New("uri must be a gnosis URI"))
			return
		}
		resolveCurrent := false
		if raw := request.URL.Query().Get("resolve_current"); raw != "" {
			parsed, err := strconv.ParseBool(raw)
			if err != nil {
				writeHTTPError(w, http.StatusBadRequest, errors.New("resolve_current must be true or false"))
				return
			}
			resolveCurrent = parsed
		}
		page, err := vault.ReadPageWithOptions(
			vaultPath,
			uri,
			vault.ReadOptions{ResolveCurrent: resolveCurrent},
		)
		if err != nil {
			writeHTTPError(w, http.StatusNotFound, err)
			return
		}
		html, err := renderPageMarkdown(page.Markdown)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, pageResponse{Page: page, HTML: html})
	}
}

func serveHistory(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		limit, err := queryLimit(request, "limit")
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		result, err := vault.ReadPageHistory(
			vaultPath,
			request.URL.Query().Get("uri"),
			request.URL.Query().Get("cursor"),
			limit,
		)
		if err != nil {
			writeHistoryHTTPError(w, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, result)
	}
}

func serveDiff(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		limit, err := queryLimit(request, "limit")
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		query := request.URL.Query()
		result, err := vault.DiffPage(
			vaultPath,
			query.Get("uri"),
			query.Get("from"),
			query.Get("to"),
			limit,
		)
		if err != nil {
			writeHistoryHTTPError(w, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, result)
	}
}

func serveChanges(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		limit, err := queryLimit(request, "limit")
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		result, err := vault.ChangesSince(
			vaultPath,
			request.URL.Query().Get("cursor"),
			limit,
		)
		if err != nil {
			writeHistoryHTTPError(w, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, result)
	}
}

func queryLimit(request *http.Request, name string) (int, error) {
	raw := request.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func writeHistoryHTTPError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, vault.ErrPageNotFound),
		errors.Is(err, vault.ErrRevisionNotFound):
		status = http.StatusNotFound
	case errors.Is(err, vault.ErrCursorExpired):
		status = http.StatusGone
	}
	writeHTTPError(w, status, err)
}

// pageResponse adds a rendered HTML view of the Markdown source to the page
// payload so the document UI can present pages as documents instead of text.
type pageResponse struct {
	vault.Page
	HTML string `json:"html"`
}

// pageMarkdown renders vault Markdown with GFM extensions and goldmark's
// default safe HTML handling, which escapes raw HTML in the source.
var pageMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

var pageFrontmatter = frontmatter.NewFormat("---", "---", yaml.Unmarshal)

func renderPageMarkdown(markdown string) (string, error) {
	var output bytes.Buffer
	if err := pageMarkdown.Convert([]byte(markdownBody(markdown)), &output); err != nil {
		return "", fmt.Errorf("render page markdown: %w", err)
	}
	return output.String(), nil
}

// markdownBody drops the YAML frontmatter block from a canonical page record;
// records without frontmatter render as-is.
func markdownBody(markdown string) string {
	fields := map[string]any{}
	body, err := frontmatter.MustParse(strings.NewReader(markdown), &fields, pageFrontmatter)
	if err != nil {
		return markdown
	}
	return string(body)
}

func serveGraph(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		graph, err := vault.ReadGraph(vaultPath)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, graph)
	}
}

func serveSearch(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		input, err := searchInput(request)
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		result, err := searchMCPKnowledge(request.Context(), vaultPath, input)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, result)
	}
}

func serveContext(vaultPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		var input evidencecontext.Request
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeHTTPError(w, http.StatusBadRequest, fmt.Errorf("invalid context request: %w", err))
			return
		}
		input = evidencecontext.Defaults(input)
		if err := evidencecontext.Validate(input); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		result, err := evidencecontext.Resolve(request.Context(), vaultPath, input)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, result)
	}
}

func searchInput(request *http.Request) (searchKnowledgeInput, error) {
	query := request.URL.Query()
	input := searchKnowledgeInput{
		Question: strings.TrimSpace(query.Get("question")),
		Backend:  strings.TrimSpace(query.Get("backend")),
	}
	if input.Question == "" {
		return searchKnowledgeInput{}, errors.New("question must not be empty")
	}
	if input.Backend != "" && input.Backend != "vector" && input.Backend != "lexical" {
		return searchKnowledgeInput{}, fmt.Errorf("unknown backend %q", input.Backend)
	}

	values := []struct {
		name   string
		target **int
	}{
		{name: "top", target: &input.Top},
		{name: "max_read", target: &input.MaxRead},
		{name: "depth", target: &input.Depth},
	}
	for _, value := range values {
		raw := query.Get(value.name)
		if raw == "" {
			continue
		}
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return searchKnowledgeInput{}, fmt.Errorf("%s must be an integer", value.name)
		}
		*value.target = &parsed
	}

	top, maxRead, depth := 3, 3, 3
	if input.Top != nil {
		top = *input.Top
	}
	if input.MaxRead != nil {
		maxRead = *input.MaxRead
	}
	if input.Depth != nil {
		depth = *input.Depth
	}
	if err := validateQueryOptions(top, maxRead, depth); err != nil {
		return searchKnowledgeInput{}, err
	}
	return input, nil
}

func writeHTTPJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeHTTPError(w http.ResponseWriter, status int, err error) {
	writeHTTPJSON(w, status, map[string]string{"error": err.Error()})
}
