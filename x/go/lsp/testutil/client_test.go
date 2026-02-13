// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var ctx context.Context

var _ = BeforeEach(func() {
	ctx = context.Background()
})

var _ = Describe("MockClient", func() {
	Describe("Diagnostics", func() {
		It("should return empty diagnostics initially", func() {
			client := &testutil.MockClient{}
			Expect(client.Diagnostics()).To(BeEmpty())
		})

		It("should capture diagnostics from PublishDiagnostics", func() {
			client := &testutil.MockClient{}
			ctx := context.Background()
			diags := []protocol.Diagnostic{
				{Message: "undefined symbol: x", Severity: protocol.DiagnosticSeverityError},
				{Message: "unused variable: y", Severity: protocol.DiagnosticSeverityWarning},
			}
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI:         "file:///test.arc",
				Diagnostics: diags,
			})).To(Succeed())
			Expect(client.Diagnostics()).To(HaveLen(2))
			Expect(client.Diagnostics()[0].Message).To(Equal("undefined symbol: x"))
			Expect(client.Diagnostics()[1].Message).To(Equal("unused variable: y"))
		})

		It("should replace diagnostics on subsequent PublishDiagnostics calls", func() {
			client := &testutil.MockClient{}
			ctx := context.Background()
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI: "file:///test.arc",
				Diagnostics: []protocol.Diagnostic{
					{Message: "first error"},
				},
			})).To(Succeed())
			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI: "file:///test.arc",
				Diagnostics: []protocol.Diagnostic{
					{Message: "second error"},
					{Message: "third error"},
				},
			})).To(Succeed())
			Expect(client.Diagnostics()).To(HaveLen(2))
			Expect(client.Diagnostics()[0].Message).To(Equal("second error"))
		})

		It("should clear diagnostics when publishing empty slice", func() {
			client := &testutil.MockClient{}
			ctx := context.Background()
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI:         "file:///test.arc",
				Diagnostics: []protocol.Diagnostic{{Message: "error"}},
			})).To(Succeed())
			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI:         "file:///test.arc",
				Diagnostics: []protocol.Diagnostic{},
			})).To(Succeed())
			Expect(client.Diagnostics()).To(BeEmpty())
		})
	})

	Describe("Stubbed Methods", func() {
		var client *testutil.MockClient

		BeforeEach(func() {
			client = &testutil.MockClient{}
		})

		It("should return nil from ShowMessage", func() {
			Expect(client.ShowMessage(ctx, &protocol.ShowMessageParams{
				Type:    protocol.MessageTypeInfo,
				Message: "test",
			})).To(Succeed())
		})

		It("should return nil from ShowMessageRequest", func() {
			Expect(MustSucceed(client.ShowMessageRequest(ctx, &protocol.ShowMessageRequestParams{}))).To(BeNil())
		})

		It("should return nil from LogMessage", func() {
			Expect(client.LogMessage(ctx, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeLog,
				Message: "log entry",
			})).To(Succeed())
		})

		It("should return nil from Telemetry", func() {
			Expect(client.Telemetry(ctx, map[string]string{"key": "value"})).To(Succeed())
		})

		It("should return nil from RegisterCapability", func() {
			Expect(client.RegisterCapability(ctx, &protocol.RegistrationParams{})).To(Succeed())
		})

		It("should return nil from UnregisterCapability", func() {
			Expect(client.UnregisterCapability(ctx, &protocol.UnregistrationParams{})).To(Succeed())
		})

		It("should return nil from WorkspaceFolders", func() {
			Expect(MustSucceed(client.WorkspaceFolders(ctx))).To(BeNil())
		})

		It("should return nil from Configuration", func() {
			Expect(MustSucceed(client.Configuration(ctx, &protocol.ConfigurationParams{}))).To(BeNil())
		})

		It("should return false from ApplyEdit", func() {
			Expect(MustSucceed(client.ApplyEdit(ctx, &protocol.ApplyWorkspaceEditParams{}))).To(BeFalse())
		})

		It("should return nil from Progress", func() {
			Expect(client.Progress(ctx, &protocol.ProgressParams{})).To(Succeed())
		})

		It("should return nil from WorkDoneProgressCreate", func() {
			Expect(client.WorkDoneProgressCreate(ctx, &protocol.WorkDoneProgressCreateParams{})).To(Succeed())
		})

		It("should return nil from ShowDocument", func() {
			Expect(MustSucceed(client.ShowDocument(ctx, &protocol.ShowDocumentParams{URI: "file:///test"}))).To(BeNil())
		})

		It("should return nil from Request", func() {
			Expect(MustSucceed(client.Request(ctx, "custom/method", nil))).To(BeNil())
		})
	})
})
