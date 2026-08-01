package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/codeintel"
	"gnosis/internal/vault"
)

func TestMCPResourceSubscriptionsObserveWritesAndExternalEdits(t *testing.T) {
	root := historyCommandVault(t)
	uri := "gnosis://test/note.md"
	updates := make(chan string, 8)
	session, _ := connectMCPServerWithOptions(t, newMCPServerWithOptions(root, true, codeintel.NewService(root)), &mcp.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, request *mcp.ResourceUpdatedNotificationRequest) {
			updates <- request.Params.URI
		},
	})
	capabilities := session.InitializeResult().Capabilities.Resources
	if capabilities == nil || !capabilities.Subscribe || !capabilities.ListChanged {
		t.Fatalf("resource capabilities = %+v", capabilities)
	}
	if err := session.Subscribe(context.Background(), &mcp.SubscribeParams{URI: uri}); err != nil {
		t.Fatal(err)
	}

	applyMCPPageUpdate(t, session, root, uri, "second", "third")
	assertResourceUpdate(t, updates, uri)

	if err := session.Unsubscribe(context.Background(), &mcp.UnsubscribeParams{URI: uri}); err != nil {
		t.Fatal(err)
	}
	applyMCPPageUpdate(t, session, root, uri, "third", "fourth")
	assertNoResourceUpdate(t, updates)

	if err := session.Subscribe(context.Background(), &mcp.SubscribeParams{URI: uri}); err != nil {
		t.Fatal(err)
	}
	writeCommandFile(t, root, "docs/note.md", strings.ReplaceAll(
		readCommandPage(t, root, uri),
		"fourth",
		"external",
	))
	assertNoResourceUpdate(t, updates)
	callMCPTool(t, session, "get_page", map[string]any{"uri": uri})
	assertResourceUpdate(t, updates, uri)
}

func TestMCPResourceSubscriptionsObserveRemoteRefreshAndListChanges(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://history.example.test/team/notifications.git")
	uri := "gnosis://remote/notes/remote.md"
	updates := make(chan string, 4)
	listChanges := make(chan struct{}, 4)
	session, _ := connectMCPServerWithOptions(t, newMCPServer(fixture.url), &mcp.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, request *mcp.ResourceUpdatedNotificationRequest) {
			updates <- request.Params.URI
		},
		ResourceListChangedHandler: func(context.Context, *mcp.ResourceListChangedRequest) {
			listChanges <- struct{}{}
		},
	})
	if err := session.Subscribe(context.Background(), &mcp.SubscribeParams{URI: uri}); err != nil {
		t.Fatal(err)
	}
	updateCommandRemoteNote(t, fixture, "Remote refreshed")
	assertNoResourceUpdate(t, updates)
	callMCPTool(t, session, "get_page", map[string]any{"uri": uri})
	assertResourceUpdate(t, updates, uri)

	writeCommandFile(t, fixture.seed, "notes/added.md", `---
type: Note
title: Added remotely
---

# Added remotely
`)
	runCommandGit(t, "-C", fixture.seed, "add", "notes/added.md")
	runCommandGit(t, "-C", fixture.seed, "commit", "-m", "add remote page")
	runCommandGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
	select {
	case <-listChanges:
		t.Fatal("resource list notification arrived before gnosis observed the edit")
	case <-time.After(50 * time.Millisecond):
	}
	callMCPTool(t, session, "get_vaults", map[string]any{})
	select {
	case <-listChanges:
	case <-time.After(time.Second):
		t.Fatal("resource list notification was not delivered")
	}
}

func TestMCPResourceSubscriptionDisconnectCleanup(t *testing.T) {
	root := historyCommandVault(t)
	server := newMCPServer(root)
	session, serverSession := connectMCPServerWithOptions(t, server, nil)
	if err := session.Subscribe(
		context.Background(),
		&mcp.SubscribeParams{URI: "gnosis://test/note.md"},
	); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := serverSession.Close(); err != nil {
		t.Fatal(err)
	}
	count := 0
	for range server.Sessions() {
		count++
	}
	if count != 0 {
		t.Fatalf("server sessions after disconnect = %d", count)
	}
	if err := server.ResourceUpdated(
		context.Background(),
		&mcp.ResourceUpdatedNotificationParams{URI: "gnosis://test/note.md"},
	); err != nil {
		t.Fatal(err)
	}
}

func connectMCPServerWithOptions(
	t *testing.T,
	server *mcp.Server,
	options *mcp.ClientOptions,
) (*mcp.ClientSession, *mcp.ServerSession) {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(
		&mcp.Implementation{Name: "gnosis-notification-test", Version: "0.0.0"},
		options,
	)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	return clientSession, serverSession
}

func applyMCPPageUpdate(
	t *testing.T,
	session *mcp.ClientSession,
	root, uri, oldValue, newValue string,
) {
	t.Helper()
	page, err := vault.ReadPage(root, uri)
	if err != nil {
		t.Fatal(err)
	}
	candidate := strings.ReplaceAll(page.Markdown, oldValue, newValue)
	input := map[string]any{
		"uri": uri, "candidate": candidate, "expected_revision": page.Document.Revision,
	}
	var plan vault.KnowledgeChangePlan
	decodeMCPResult(t, callMCPTool(t, session, "propose_knowledge_change", input), &plan)
	input["digest"] = plan.Digest
	callMCPTool(t, session, "apply_knowledge_change", input)
}

func readCommandPage(t *testing.T, root, uri string) string {
	t.Helper()
	page, err := vault.ReadPage(root, uri)
	if err != nil {
		t.Fatal(err)
	}
	return page.Markdown
}

func assertResourceUpdate(t *testing.T, updates <-chan string, uri string) {
	t.Helper()
	select {
	case got := <-updates:
		if got != uri {
			t.Fatalf("resource update = %q, want %q", got, uri)
		}
	case <-time.After(time.Second):
		t.Fatalf("resource update for %q was not delivered", uri)
	}
}

func assertNoResourceUpdate(t *testing.T, updates <-chan string) {
	t.Helper()
	select {
	case got := <-updates:
		t.Fatalf("unexpected resource update for %q", got)
	case <-time.After(50 * time.Millisecond):
	}
}
