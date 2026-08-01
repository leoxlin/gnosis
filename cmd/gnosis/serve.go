package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"gnosis/internal/codeintel"
	evidencecontext "gnosis/internal/evidencecontext"
	agentlearning "gnosis/internal/learning"
	agentmemory "gnosis/internal/memory"
	"gnosis/internal/search"
	agenttrace "gnosis/internal/trace"
	"gnosis/internal/vault"
)

const mcpPageResourceTemplate = "gnosis://{vault}/{+path}"
const maxMCPGraphNeighbors = 100

type emptyInput struct{}

type getConceptsInput struct {
	Type string `json:"type,omitempty" jsonschema:"exact concept type"`
}

type conceptsOutput struct {
	ConceptTypes []vault.ConceptTypeSummary `json:"concept_types,omitempty"`
	Type         string                     `json:"type,omitempty"`
	Concepts     []map[string]any           `json:"concepts,omitempty"`
}

type getPageInput struct {
	URI            string `json:"uri" jsonschema:"canonical gnosis URI"`
	ResolveCurrent bool   `json:"resolve_current,omitempty" jsonschema:"follow the bounded supersession chain"`
}

type searchKnowledgeInput struct {
	Question string `json:"question" jsonschema:"knowledge question"`
	Backend  string `json:"backend,omitempty" jsonschema:"retrieval backend: vector or lexical"`
	Top      *int   `json:"top,omitempty" jsonschema:"number of candidate pages"`
	MaxRead  *int   `json:"max_read,omitempty" jsonschema:"maximum pages to recommend reading"`
	Depth    *int   `json:"depth,omitempty" jsonschema:"maximum graph traversal depth"`
}

type addMemoryInput struct {
	Text string `json:"text" jsonschema:"durable memory text"`
}

type searchMemoryInput struct {
	Query string `json:"query" jsonschema:"memory search query"`
	Limit *int   `json:"limit,omitempty" jsonschema:"maximum memories to return, from 1 through 20"`
}

type recordTraceInput struct {
	RunID             string         `json:"run_id" jsonschema:"non-empty run identity, at most 256 bytes"`
	Sequence          int64          `json:"sequence" jsonschema:"non-negative run sequence"`
	Kind              string         `json:"kind" jsonschema:"trace kind: run, plan, tool, patch, test, failure, outcome, knowledge_use, or feedback"`
	OccurredAt        string         `json:"occurred_at" jsonschema:"RFC 3339 occurrence time"`
	Content           string         `json:"content" jsonschema:"non-empty trace content, at most 65536 bytes"`
	Metadata          map[string]any `json:"metadata,omitempty" jsonschema:"optional JSON metadata, at most 65536 encoded bytes"`
	KnowledgeURI      string         `json:"knowledge_uri,omitempty" jsonschema:"canonical cited gnosis URI for knowledge_use or feedback"`
	KnowledgeRevision string         `json:"knowledge_revision,omitempty" jsonschema:"exact cited sha256 revision for knowledge_use or feedback"`
	Feedback          string         `json:"feedback,omitempty" jsonschema:"feedback value: helpful, harmful, irrelevant, or unassessed"`
}

type getRunTraceInput struct {
	RunID         string `json:"run_id" jsonschema:"exact non-empty run identity"`
	Cursor        *int64 `json:"cursor,omitempty" jsonschema:"first sequence to return"`
	MaxEntries    *int   `json:"max_entries,omitempty" jsonschema:"maximum entries to return, from 1 through 1000"`
	MaxCharacters *int   `json:"max_characters,omitempty" jsonschema:"maximum content characters to return, from 1 through 1048576"`
}

type proposeRunLearningInput struct {
	Runs             []agentlearning.Selection `json:"runs" jsonschema:"explicit run identities and caller-authored learning keys"`
	Type             string                    `json:"type" jsonschema:"curated page type: Event or Reflection"`
	URI              string                    `json:"uri" jsonschema:"canonical target gnosis URI"`
	Title            string                    `json:"title" jsonschema:"curated page title"`
	Statement        string                    `json:"statement" jsonschema:"caller-authored Event or Reflection statement"`
	OccurredAt       string                    `json:"occurred_at,omitempty" jsonschema:"RFC 3339 Event occurrence time"`
	ExpectedRevision string                    `json:"expected_revision,omitempty" jsonschema:"required current revision for update"`
	ExpectedAbsent   bool                      `json:"expected_absent,omitempty" jsonschema:"require the target page to be absent"`
}

type traceGraphInput struct {
	URI       string   `json:"uri" jsonschema:"canonical source gnosis URI"`
	TargetURI string   `json:"target_uri,omitempty" jsonschema:"canonical target gnosis URI for path mode"`
	Direction string   `json:"direction,omitempty" jsonschema:"edge direction: out, in, or both"`
	Relations []string `json:"relations,omitempty" jsonschema:"relationship type filters"`
	Depth     *int     `json:"depth,omitempty" jsonschema:"maximum path depth"`
	Limit     *int     `json:"limit,omitempty" jsonschema:"maximum neighbor edges, from 1 through 100"`
}

type boundedGraphNeighbors struct {
	vault.GraphNeighbors
	Total        int      `json:"total"`
	Truncated    bool     `json:"truncated"`
	Continuation []string `json:"continuation,omitempty"`
}

type traceGraphOutput struct {
	Mode      string                 `json:"mode"`
	Neighbors *boundedGraphNeighbors `json:"neighbors,omitempty"`
	Path      *vault.GraphPath       `json:"path,omitempty"`
}

type getProceduresInput struct {
	URI  string   `json:"uri,omitempty" jsonschema:"canonical Procedure gnosis URI"`
	Tags []string `json:"tags,omitempty" jsonschema:"require all Procedure tags in discovery mode"`
}

type getProceduresOutput struct {
	Mode       string                   `json:"mode"`
	Procedures []map[string]any         `json:"procedures,omitempty"`
	Procedure  *vault.ProcessInvocation `json:"procedure,omitempty"`
}

type proposeKnowledgeChangeInput struct {
	URI              string `json:"uri" jsonschema:"canonical gnosis URI"`
	Candidate        string `json:"candidate" jsonschema:"complete typed Markdown candidate"`
	ExpectedRevision string `json:"expected_revision,omitempty" jsonschema:"required current revision for update or archive"`
	ExpectedAbsent   bool   `json:"expected_absent,omitempty" jsonschema:"require the target page to be absent"`
}

type applyKnowledgeChangeInput struct {
	URI              string `json:"uri" jsonschema:"canonical gnosis URI"`
	Candidate        string `json:"candidate" jsonschema:"complete typed Markdown candidate"`
	ExpectedRevision string `json:"expected_revision,omitempty" jsonschema:"required current revision for update or archive"`
	ExpectedAbsent   bool   `json:"expected_absent,omitempty" jsonschema:"require the target page to be absent"`
	Digest           string `json:"digest" jsonschema:"digest returned by propose_knowledge_change"`
}

type getHistoryInput struct {
	URI    string `json:"uri" jsonschema:"canonical gnosis URI"`
	Cursor string `json:"cursor,omitempty" jsonschema:"opaque continuation cursor"`
	Limit  *int   `json:"limit,omitempty" jsonschema:"maximum committed entries, from 1 through 100"`
}

type getDiffInput struct {
	URI          string `json:"uri" jsonschema:"canonical gnosis URI"`
	FromRevision string `json:"from_revision" jsonschema:"exact earlier content revision"`
	ToRevision   string `json:"to_revision" jsonschema:"exact later content revision"`
	Limit        *int   `json:"limit,omitempty" jsonschema:"maximum diff characters"`
}

type getChangesInput struct {
	Cursor string `json:"cursor,omitempty" jsonschema:"opaque committed effective-view cursor"`
	Limit  *int   `json:"limit,omitempty" jsonschema:"maximum changes, from 1 through 100"`
}

type searchCodeInput struct {
	Scope    string `json:"scope" jsonschema:"configured code scope"`
	Query    string `json:"query" jsonschema:"symbol name query"`
	Language string `json:"language,omitempty" jsonschema:"canonical language filter"`
	Limit    *int   `json:"limit,omitempty" jsonschema:"maximum symbols to return"`
}

type getCodeSymbolInput struct {
	Scope string `json:"scope" jsonschema:"configured code scope"`
	ID    string `json:"id" jsonschema:"exact normalized code symbol ID"`
}

type traceCodeInput struct {
	Scope     string `json:"scope" jsonschema:"configured code scope"`
	ID        string `json:"id" jsonschema:"exact normalized code symbol ID"`
	TargetID  string `json:"target_id,omitempty" jsonschema:"exact target symbol ID for path mode"`
	Mode      string `json:"mode,omitempty" jsonschema:"relations, neighbors, or path"`
	Direction string `json:"direction,omitempty" jsonschema:"incoming or outgoing"`
	Depth     *int   `json:"depth,omitempty" jsonschema:"maximum path depth"`
	Limit     *int   `json:"limit,omitempty" jsonschema:"maximum relations to return"`
}

type getCodeDiagnosticsInput struct {
	Scope    string `json:"scope" jsonschema:"configured code scope"`
	Path     string `json:"path,omitempty" jsonschema:"repository-relative indexed path"`
	Language string `json:"language,omitempty" jsonschema:"canonical language filter"`
	Category string `json:"category,omitempty" jsonschema:"diagnostic category filter"`
	Limit    *int   `json:"limit,omitempty" jsonschema:"maximum diagnostics to return"`
}

type getCodeStatusInput struct {
	Scope string `json:"scope" jsonschema:"configured code scope"`
}

func newMCPServer(vaultPath string) *mcp.Server {
	return newMCPServerWithKnowledgeWrites(vaultPath, false)
}

func newMCPServerWithKnowledgeWrites(vaultPath string, allowKnowledgeWrites bool) *mcp.Server {
	return newMCPServerWithOptions(vaultPath, allowKnowledgeWrites, nil)
}

func newMCPServerWithOptions(vaultPath string, allowKnowledgeWrites bool, codeService *codeintel.Service) *mcp.Server {
	observer := newResourceObserver(vaultPath)
	server := mcp.NewServer(
		&mcp.Implementation{Name: "gnosis", Version: "0.1.0"},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Resources: &mcp.ResourceCapabilities{Subscribe: true, ListChanged: true},
			},
			SubscribeHandler:   observer.subscribe,
			UnsubscribeHandler: observer.unsubscribe,
		},
	)
	observer.attach(server)
	addMCPResources(server, vaultPath, observer)
	addCodeMCPTools(server, vaultPath, codeService)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_vaults",
		Description: "List the effective gnosis vaults",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, vault.VaultCatalog, error) {
		catalog, err := vault.Vaults(vaultPath)
		return nil, catalog, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_concepts",
		Description: "List concept types or records of one exact type",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getConceptsInput) (*mcp.CallToolResult, conceptsOutput, error) {
		return getMCPConcepts(vaultPath, input.Type)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_page",
		Description: "Read one exact gnosis page by canonical URI",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getPageInput) (*mcp.CallToolResult, vault.Page, error) {
		page, err := vault.ReadPageWithOptions(
			vaultPath,
			input.URI,
			vault.ReadOptions{ResolveCurrent: input.ResolveCurrent},
		)
		return nil, page, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_history",
		Description: "Read bounded committed and working history for one canonical page",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getHistoryInput) (*mcp.CallToolResult, vault.PageHistoryResult, error) {
		result, err := vault.ReadPageHistory(
			vaultPath, input.URI, input.Cursor, intValue(input.Limit),
		)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_diff",
		Description: "Diff two exact revisions of one canonical page",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getDiffInput) (*mcp.CallToolResult, vault.PageDiffResult, error) {
		result, err := vault.DiffPage(
			vaultPath,
			input.URI,
			input.FromRevision,
			input.ToRevision,
			intValue(input.Limit),
		)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_changes",
		Description: "Read committed effective-vault changes after an opaque cursor",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getChangesInput) (*mcp.CallToolResult, vault.ChangeFeedResult, error) {
		result, err := vault.ChangesSince(vaultPath, input.Cursor, intValue(input.Limit))
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_knowledge",
		Description: "Search gnosis knowledge using vector or lexical retrieval",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchKnowledgeInput) (*mcp.CallToolResult, search.QueryResult, error) {
		result, err := searchMCPKnowledge(ctx, vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trace_graph",
		Description: "Trace bounded graph neighbors or a path between canonical gnosis URIs",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input traceGraphInput) (*mcp.CallToolResult, traceGraphOutput, error) {
		result, err := traceMCPGraph(vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_procedures",
		Description: "Discover eligible Procedures by tags or read one exact validated contract",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getProceduresInput) (*mcp.CallToolResult, getProceduresOutput, error) {
		result, err := getMCPProcedures(vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_evidence_context",
		Description: "Resolve bounded cited evidence without generating an answer",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input evidencecontext.Request) (*mcp.CallToolResult, evidencecontext.Result, error) {
		result, err := evidencecontext.Resolve(ctx, vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "audit_knowledge",
		Description: "Report bounded deterministic knowledge-health findings without mutation",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input vault.KnowledgeAuditRequest) (*mcp.CallToolResult, vault.KnowledgeAuditResult, error) {
		result, err := vault.AuditKnowledge(vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "propose_knowledge_change",
		Description: "Validate and diff one complete revision-bound knowledge candidate without side effects",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input proposeKnowledgeChangeInput) (*mcp.CallToolResult, vault.KnowledgeChangePlan, error) {
		result, err := vault.PlanKnowledgeChange(vaultPath, input.knowledgeChange())
		return nil, result, err
	})
	if allowKnowledgeWrites {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "apply_knowledge_change",
			Description: "Apply one accepted revision-bound knowledge change; host approval is required",
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input applyKnowledgeChangeInput) (*mcp.CallToolResult, vault.KnowledgeChangeResult, error) {
			result, err := vault.ApplyKnowledgeChange(ctx, vaultPath, input.knowledgeChange(), input.Digest)
			return nil, result, err
		})
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_memory",
		Description: "Store one durable memory in the configured user and agent scope",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input addMemoryInput) (*mcp.CallToolResult, agentmemory.Result, error) {
		client, err := agentmemory.NewFromEnv(vaultPath)
		if err != nil {
			return nil, agentmemory.Result{}, err
		}
		result, err := client.Add(ctx, input.Text)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_memory",
		Description: "Search durable memories in the configured user and agent scope",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchMemoryInput) (*mcp.CallToolResult, agentmemory.Result, error) {
		client, err := agentmemory.NewFromEnv(vaultPath)
		if err != nil {
			return nil, agentmemory.Result{}, err
		}
		result, err := client.Search(ctx, input.Query, input.Limit)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "record_trace",
		Description: "Append one explicit agent-run trace entry to configured local storage",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input recordTraceInput) (*mcp.CallToolResult, agenttrace.Result, error) {
		store, err := agenttrace.NewFromEnv()
		if err != nil {
			return nil, agenttrace.Result{}, err
		}
		result, err := store.Record(agenttrace.Input{
			RunID: input.RunID, Sequence: input.Sequence, Kind: input.Kind,
			OccurredAt: input.OccurredAt, Content: input.Content, Metadata: input.Metadata,
			KnowledgeURI: input.KnowledgeURI, KnowledgeRevision: input.KnowledgeRevision,
			Feedback: input.Feedback,
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_run_trace",
		Description: "Read one exact configured-agent run with deterministic bounds and diagnostics",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getRunTraceInput) (*mcp.CallToolResult, agenttrace.Run, error) {
		store, err := agenttrace.NewFromEnv()
		if err != nil {
			return nil, agenttrace.Run{}, err
		}
		result, err := store.Read(input.RunID, agenttrace.ReadOptions{
			Cursor: input.Cursor, MaxEntries: intValue(input.MaxEntries),
			MaxCharacters: intValue(input.MaxCharacters),
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "propose_run_learning",
		Description: "Build an evidence-bound Event or Reflection knowledge plan without writing the vault",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input proposeRunLearningInput) (*mcp.CallToolResult, agentlearning.Proposal, error) {
		store, err := agenttrace.NewFromEnv()
		if err != nil {
			return nil, agentlearning.Proposal{}, err
		}
		candidate, err := agentlearning.Build(store, input.Runs)
		if err != nil {
			return nil, agentlearning.Proposal{}, err
		}
		result, err := agentlearning.Propose(vaultPath, store, agentlearning.ProposalInput{
			Candidate: candidate, Type: input.Type, URI: input.URI,
			Title: input.Title, Statement: input.Statement, OccurredAt: input.OccurredAt,
			ExpectedRevision: input.ExpectedRevision, ExpectedAbsent: input.ExpectedAbsent,
		})
		return nil, result, err
	})
	return server
}

type codeReadView interface {
	Search(string, string, int) codeintel.SearchResult
	ReadSymbol(string) (codeintel.SymbolResult, error)
	Diagnostics(string, string, string, int) codeintel.DiagnosticResult
	Trace(string, string, int) (codeintel.TraceResult, error)
	Neighbors(string, string, int) (codeintel.TraceResult, error)
	Path(string, string, string, int, int) (codeintel.TraceResult, error)
}

func addCodeMCPTools(server *mcp.Server, workspace string, codeService *codeintel.Service) {
	mcp.AddTool(server, &mcp.Tool{Name: "search_code", Description: "Search one current configured code index with deterministic bounds"}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchCodeInput) (*mcp.CallToolResult, codeintel.SearchResult, error) {
		var result codeintel.SearchResult
		err := withCurrentCode(ctx, workspace, codeService, input.Scope, func(reader codeReadView) error {
			result = reader.Search(input.Query, input.Language, intValue(input.Limit))
			return nil
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "get_code_symbol", Description: "Read one exact symbol from a current configured code index"}, func(ctx context.Context, _ *mcp.CallToolRequest, input getCodeSymbolInput) (*mcp.CallToolResult, codeintel.SymbolResult, error) {
		var result codeintel.SymbolResult
		err := withCurrentCode(ctx, workspace, codeService, input.Scope, func(reader codeReadView) error {
			var err error
			result, err = reader.ReadSymbol(input.ID)
			return err
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "trace_code", Description: "Read bounded incoming or outgoing normalized code relations"}, func(ctx context.Context, _ *mcp.CallToolRequest, input traceCodeInput) (*mcp.CallToolResult, codeintel.TraceResult, error) {
		direction := input.Direction
		if direction == "" {
			direction = "outgoing"
		}
		if direction != "incoming" && direction != "outgoing" {
			return nil, codeintel.TraceResult{}, errors.New("direction must be incoming or outgoing")
		}
		var result codeintel.TraceResult
		err := withCurrentCode(ctx, workspace, codeService, input.Scope, func(reader codeReadView) error {
			var err error
			switch input.Mode {
			case "", "relations":
				result, err = reader.Trace(input.ID, direction, intValue(input.Limit))
			case "neighbors":
				result, err = reader.Neighbors(input.ID, direction, intValue(input.Limit))
			case "path":
				if input.TargetID == "" {
					return errors.New("target_id is required for path mode")
				}
				result, err = reader.Path(input.ID, input.TargetID, direction, intValue(input.Depth), intValue(input.Limit))
			default:
				return errors.New("mode must be relations, neighbors, or path")
			}
			return err
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "get_code_diagnostics", Description: "Read bounded diagnostics from one current configured code index"}, func(ctx context.Context, _ *mcp.CallToolRequest, input getCodeDiagnosticsInput) (*mcp.CallToolResult, codeintel.DiagnosticResult, error) {
		var result codeintel.DiagnosticResult
		err := withCurrentCode(ctx, workspace, codeService, input.Scope, func(reader codeReadView) error {
			result = reader.Diagnostics(input.Path, input.Language, input.Category, intValue(input.Limit))
			return nil
		})
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "get_code_index_status", Description: "Read code-index freshness, counts, coverage provenance, and snapshot identity"}, func(ctx context.Context, _ *mcp.CallToolRequest, input getCodeStatusInput) (*mcp.CallToolResult, codeintel.StatusResult, error) {
		if codeService != nil {
			status, err := codeService.Status(ctx, input.Scope)
			return nil, status, err
		}
		reader, err := codeintel.Open(workspace, input.Scope)
		if err != nil {
			return nil, codeintel.StatusResult{}, err
		}
		defer reader.Close()
		status := reader.Status()
		if err := reader.CheckCurrent(ctx); err != nil {
			status.Status = "not_current"
		}
		return nil, status, nil
	})
}

func withCurrentCode(ctx context.Context, workspace string, service *codeintel.Service, scope string, callback func(codeReadView) error) error {
	if service != nil {
		return service.ReadCurrent(ctx, scope, func(view codeintel.ReadView) error { return callback(view) })
	}
	reader, err := currentCodeReader(ctx, workspace, scope)
	if err != nil {
		return err
	}
	defer reader.Close()
	return callback(reader)
}

func currentCodeReader(ctx context.Context, workspace, scope string) (*codeintel.Reader, error) {
	if strings.TrimSpace(scope) == "" {
		return nil, errors.New("scope is required")
	}
	reader, err := codeintel.Open(workspace, scope)
	if err != nil {
		return nil, err
	}
	if err := reader.CheckCurrent(ctx); err != nil {
		reader.Close()
		return nil, err
	}
	return reader, nil
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func (input proposeKnowledgeChangeInput) knowledgeChange() vault.KnowledgeChangeInput {
	return vault.KnowledgeChangeInput{
		URI:              input.URI,
		Candidate:        input.Candidate,
		ExpectedRevision: input.ExpectedRevision,
		ExpectedAbsent:   input.ExpectedAbsent,
	}
}

func (input applyKnowledgeChangeInput) knowledgeChange() vault.KnowledgeChangeInput {
	return vault.KnowledgeChangeInput{
		URI:              input.URI,
		Candidate:        input.Candidate,
		ExpectedRevision: input.ExpectedRevision,
		ExpectedAbsent:   input.ExpectedAbsent,
	}
}

func traceMCPGraph(vaultPath string, input traceGraphInput) (traceGraphOutput, error) {
	input.URI = strings.TrimSpace(input.URI)
	input.TargetURI = strings.TrimSpace(input.TargetURI)
	if !vault.IsCanonicalURI(input.URI) {
		return traceGraphOutput{}, errors.New("trace graph: uri must be a canonical gnosis URI")
	}
	if input.Direction == "" {
		input.Direction = string(vault.DirectionBoth)
	}
	if err := validateDirection(input.Direction); err != nil {
		return traceGraphOutput{}, fmt.Errorf("trace graph: %w", err)
	}
	for _, relation := range input.Relations {
		if strings.TrimSpace(relation) == "" {
			return traceGraphOutput{}, errors.New("trace graph: relations must not contain empty values")
		}
	}

	if input.TargetURI == "" {
		if input.Depth != nil {
			return traceGraphOutput{}, errors.New("trace graph: depth is available only in path mode")
		}
		limit := maxMCPGraphNeighbors
		if input.Limit != nil {
			limit = *input.Limit
		}
		if limit < 1 || limit > maxMCPGraphNeighbors {
			return traceGraphOutput{}, fmt.Errorf(
				"trace graph: limit must be between 1 and %d", maxMCPGraphNeighbors,
			)
		}
		result, err := vault.TraceNeighbors(
			vaultPath, input.URI, vault.Direction(input.Direction), input.Relations,
		)
		if err != nil {
			return traceGraphOutput{}, err
		}
		total := len(result.Edges)
		bounded := boundedGraphNeighbors{GraphNeighbors: result, Total: total}
		if total > limit {
			bounded.Edges = bounded.Edges[:limit]
			bounded.Truncated = true
			bounded.Continuation = []string{
				"Refine direction or relations to continue with a deterministic subset.",
			}
		}
		return traceGraphOutput{Mode: "neighbors", Neighbors: &bounded}, nil
	}

	if !vault.IsCanonicalURI(input.TargetURI) {
		return traceGraphOutput{}, errors.New("trace graph: target_uri must be a canonical gnosis URI")
	}
	if input.Limit != nil {
		return traceGraphOutput{}, errors.New("trace graph: limit is available only in neighbor mode")
	}
	depth := 3
	if input.Depth != nil {
		depth = *input.Depth
	}
	if depth < 0 {
		return traceGraphOutput{}, errors.New("trace graph: depth must be zero or greater")
	}
	result, err := vault.TracePath(
		vaultPath,
		input.URI,
		input.TargetURI,
		vault.Direction(input.Direction),
		input.Relations,
		depth,
	)
	if err != nil {
		return traceGraphOutput{}, err
	}
	return traceGraphOutput{Mode: "path", Path: &result}, nil
}

func getMCPProcedures(vaultPath string, input getProceduresInput) (getProceduresOutput, error) {
	input.URI = strings.TrimSpace(input.URI)
	for index, tag := range input.Tags {
		input.Tags[index] = strings.TrimSpace(tag)
		if input.Tags[index] == "" {
			return getProceduresOutput{}, errors.New("get procedures: tags must not contain empty values")
		}
	}
	if input.URI == "" {
		catalog, err := vault.DiscoverProcesses(vaultPath, input.Tags)
		if err != nil {
			return getProceduresOutput{}, err
		}
		return getProceduresOutput{Mode: "discovery", Procedures: catalog["procedures"]}, nil
	}
	if !vault.IsCanonicalURI(input.URI) {
		return getProceduresOutput{}, errors.New("get procedures: uri must be a canonical gnosis URI")
	}
	if len(input.Tags) != 0 {
		return getProceduresOutput{}, errors.New("get procedures: tags are available only in discovery mode")
	}
	procedure, err := vault.InvokeProcess(vaultPath, input.URI)
	if err != nil {
		return getProceduresOutput{}, err
	}
	return getProceduresOutput{Mode: "contract", Procedure: &procedure}, nil
}

func addMCPResources(server *mcp.Server, vaultPath string, observer *resourceObserver) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: mcpPageResourceTemplate,
		Name:        "gnosis-page",
		Title:       "gnosis page",
		Description: "Read one effective gnosis page selected by its canonical URI",
		MIMEType:    vault.ResourceMediaType,
	}, func(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return readMCPPageResource(vaultPath, request.Params.URI)
	})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			var result mcp.Result
			var err error
			if method == "resources/list" {
				params := request.GetParams().(*mcp.ListResourcesParams)
				cursor := ""
				if params != nil {
					cursor = params.Cursor
				}
				page, listErr := vault.ListResourcePage(vaultPath, cursor)
				if errors.Is(listErr, vault.ErrInvalidResourceCursor) {
					return nil, &jsonrpc.Error{
						Code: jsonrpc.CodeInvalidParams, Message: listErr.Error(),
					}
				}
				if listErr != nil {
					return nil, listErr
				}
				listResult := &mcp.ListResourcesResult{
					Resources:  make([]*mcp.Resource, 0, len(page.Resources)),
					NextCursor: page.NextCursor,
				}
				for _, resource := range page.Resources {
					listResult.Resources = append(listResult.Resources, &mcp.Resource{
						URI:         resource.URI,
						Name:        resource.URI,
						Title:       resource.Title,
						Description: resource.Description,
						MIMEType:    resource.MediaType,
						Size:        resource.Size,
						Meta: mcp.Meta{
							"origin":   resource.Origin,
							"revision": resource.Revision,
						},
					})
				}
				result = listResult
			} else {
				result, err = next(ctx, method, request)
			}
			if err != nil {
				return nil, err
			}
			if err := observer.observe(ctx); err != nil {
				return nil, err
			}
			return result, nil
		}
	})
}

func readMCPPageResource(
	vaultPath, uri string,
) (*mcp.ReadResourceResult, error) {
	resource, err := vault.ReadResource(vaultPath, uri)
	if errors.Is(err, vault.ErrPageNotFound) {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI:      resource.URI,
		MIMEType: resource.MediaType,
		Text:     resource.Markdown,
		Meta: mcp.Meta{
			"origin":   resource.Origin,
			"revision": resource.Revision,
		},
	}}}, nil
}

func getMCPConcepts(vaultPath, conceptType string) (*mcp.CallToolResult, conceptsOutput, error) {
	conceptType = strings.TrimSpace(conceptType)
	if conceptType == "" {
		catalog, err := vault.Concepts(vaultPath, "")
		return nil, conceptsOutput{ConceptTypes: catalog.ConceptTypes}, err
	}

	types, err := vault.Concepts(vaultPath, "")
	if err != nil {
		return nil, conceptsOutput{}, err
	}
	found := false
	for _, summary := range types.ConceptTypes {
		if summary.Type == conceptType {
			found = true
			break
		}
	}
	if !found {
		return nil, conceptsOutput{}, fmt.Errorf("unknown concept type %q", conceptType)
	}
	records, err := vault.ConceptRecords(vaultPath, conceptType)
	if err != nil {
		return nil, conceptsOutput{}, err
	}
	return nil, conceptsOutput{Type: conceptType, Concepts: records["concepts"]}, nil
}

func searchMCPKnowledge(ctx context.Context, vaultPath string, input searchKnowledgeInput) (search.QueryResult, error) {
	options := search.QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3}
	if input.Top != nil {
		options.Top = *input.Top
	}
	if input.MaxRead != nil {
		options.MaxRead = *input.MaxRead
	}
	if input.Depth != nil {
		options.MaxDepth = *input.Depth
	}
	if err := validateQueryOptions(options.Top, options.MaxRead, options.MaxDepth); err != nil {
		return search.QueryResult{}, fmt.Errorf("search knowledge: %w", err)
	}

	backend := input.Backend
	if backend == "" {
		backend = "vector"
	}
	switch backend {
	case "vector":
		config, err := search.SemanticConfigFromEnv(vaultPath)
		if err != nil {
			return search.QueryResult{}, err
		}
		return search.QuerySemantic(ctx, vaultPath, input.Question, options, config)
	case "lexical":
		return search.QueryLexical(vaultPath, input.Question, options)
	default:
		return search.QueryResult{}, fmt.Errorf("search knowledge: unknown backend %q", backend)
	}
}

func newServeCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:     "serve",
		Short:   "Serve gnosis over a protocol transport",
		Args:    cobra.NoArgs,
		GroupID: "workspace",
		Example: "gnosis serve http --address 127.0.0.1:8080\n" +
			"gnosis serve mcp",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("serve: missing transport"))
		},
	}
	command.AddCommand(newServeHTTPCommand(options), newServeMCPCommand(options))
	return command
}

func newServeMCPCommand(options *rootOptions) *cobra.Command {
	var allowKnowledgeWrites bool
	command := &cobra.Command{
		Use:   "mcp [flags]",
		Short: "Serve gnosis tools over MCP stdio",
		Args:  cobra.NoArgs,
		Example: "gnosis serve mcp\n" +
			"gnosis --vault <name> serve mcp\n" +
			"gnosis --vault <name> serve mcp --allow-knowledge-writes",
		RunE: func(command *cobra.Command, _ []string) error {
			service, err := codeintel.OpenService(command.Context(), options.vaultPath)
			if err != nil {
				return err
			}
			runErr := newMCPServerWithOptions(options.vaultPath, allowKnowledgeWrites, service).Run(command.Context(), &mcp.StdioTransport{})
			return errors.Join(runErr, service.Close())
		},
	}
	command.Flags().BoolVar(
		&allowKnowledgeWrites,
		"allow-knowledge-writes",
		false,
		"register approval-gated general knowledge apply",
	)
	return command
}
