// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"
	"sync"
	"time"

	"go.lsp.dev/protocol"
)

// MockClient implements protocol.Client and xlsp.Client for testing LSP servers.
type MockClient struct {
	mu                   sync.Mutex
	diagnostics          []protocol.Diagnostic
	publishCount         int
	semanticRefreshCount int
}

// Diagnostics returns the most recently published diagnostics.
func (m *MockClient) Diagnostics() []protocol.Diagnostic {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.diagnostics
}

// PublishCount returns the number of times PublishDiagnostics has been called.
func (m *MockClient) PublishCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishCount
}

// WaitForDiagnostics blocks until publishCount changes from the given
// baseline value or timeout elapses. Returns true if a new publish was observed.
func (m *MockClient) WaitForDiagnostics(
	baseline int,
	timeout time.Duration,
) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.PublishCount() != baseline {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return m.PublishCount() != baseline
}

func (m *MockClient) ShowMessage(context.Context, *protocol.ShowMessageParams) error {
	return nil
}

func (m *MockClient) ShowMessageRequest(context.Context, *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return nil, nil
}

func (m *MockClient) LogMessage(context.Context, *protocol.LogMessageParams) error {
	return nil
}

func (m *MockClient) Telemetry(context.Context, interface{}) error {
	return nil
}

func (m *MockClient) RegisterCapability(context.Context, *protocol.RegistrationParams) error {
	return nil
}

func (m *MockClient) UnregisterCapability(context.Context, *protocol.UnregistrationParams) error {
	return nil
}

func (m *MockClient) WorkspaceFolders(context.Context) ([]protocol.WorkspaceFolder, error) {
	return nil, nil
}

func (m *MockClient) Configuration(context.Context, *protocol.ConfigurationParams) ([]any, error) {
	return nil, nil
}

func (m *MockClient) ApplyEdit(context.Context, *protocol.ApplyWorkspaceEditParams) (bool, error) {
	return false, nil
}

// PublishDiagnostics stores the diagnostics and increments the publish count.
func (m *MockClient) PublishDiagnostics(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.diagnostics = params.Diagnostics
	m.publishCount++
	return nil
}

func (m *MockClient) Progress(context.Context, *protocol.ProgressParams) error {
	return nil
}

func (m *MockClient) WorkDoneProgressCreate(context.Context, *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

func (m *MockClient) ShowDocument(context.Context, *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, nil
}

func (m *MockClient) Request(context.Context, string, any) (any, error) {
	return nil, nil
}

// SemanticTokensRefresh increments the semantic refresh counter.
func (m *MockClient) SemanticTokensRefresh(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.semanticRefreshCount++
	return nil
}

// SemanticRefreshCount returns the number of times SemanticTokensRefresh has been called.
func (m *MockClient) SemanticRefreshCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.semanticRefreshCount
}

// WaitForSemanticRefresh blocks until semanticRefreshCount changes from the
// given baseline value or timeout elapses.
func (m *MockClient) WaitForSemanticRefresh(baseline int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.SemanticRefreshCount() != baseline {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return m.SemanticRefreshCount() != baseline
}
