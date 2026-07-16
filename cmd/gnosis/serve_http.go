package main

import (
	"context"
	_ "embed"
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
	"gnosis/internal/vault"
)

const defaultHTTPAddress = "127.0.0.1:8080"

//go:embed ui.html
var uiHTML []byte

func newServeHTTPCommand(options *rootOptions) *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:   "http [flags]",
		Short: "Serve the gnosis API, document UI, and MCP over HTTP",
		Args:  cobra.NoArgs,
		Example: "gnosis serve http --address 127.0.0.1:8080\n" +
			"gnosis --vault <path> serve http",
		RunE: func(command *cobra.Command, _ []string) error {
			return serveHTTP(
				command.Context(), address, options.vaultPath, command.ErrOrStderr(),
			)
		},
	}
	command.Flags().StringVar(&address, "address", defaultHTTPAddress, "HTTP listen address")
	return command
}

func serveHTTP(ctx context.Context, address, vaultPath string, output io.Writer) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("serve http: listen: %w", err)
	}
	server := &http.Server{
		Handler:           newHTTPHandler(vaultPath),
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", serveUI)
	mux.HandleFunc("GET /api/v1/vaults", serveVaults(vaultPath))
	mux.HandleFunc("GET /api/v1/concepts", serveConcepts(vaultPath))
	mux.HandleFunc("GET /api/v1/pages", servePages(vaultPath))
	mux.HandleFunc("GET /api/v1/page", servePage(vaultPath))
	mux.HandleFunc("GET /api/v1/graph", serveGraph(vaultPath))
	mux.HandleFunc("GET /api/v1/search", serveSearch(vaultPath))
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return newMCPServer(vaultPath)
	}, nil)
	mux.Handle("/mcp", http.NewCrossOriginProtection().Handler(mcpHandler))
	return mux
}

func serveUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(uiHTML); err != nil {
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
		page, err := vault.ReadPage(vaultPath, uri)
		if err != nil {
			writeHTTPError(w, http.StatusNotFound, err)
			return
		}
		writeHTTPJSON(w, http.StatusOK, page)
	}
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
