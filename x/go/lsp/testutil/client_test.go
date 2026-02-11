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
	"go.lsp.dev/protocol"
)

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
		var (
			client *testutil.MockClient
			ctx    context.Context
		)

		BeforeEach(func() {
			client = &testutil.MockClient{}
			ctx = context.Background()
		})

		It("should return nil from ShowMessage", func() {
			Expect(client.ShowMessage(ctx, &protocol.ShowMessageParams{
				Type:    protocol.MessageTypeInfo,
				Message: "test",
			})).To(Succeed())
		})

		It("should return nil from ShowMessageRequest", func() {
			result, err := client.ShowMessageRequest(ctx, &protocol.ShowMessageRequestParams{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
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
			folders, err := client.WorkspaceFolders(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(folders).To(BeNil())
		})

		It("should return nil from Configuration", func() {
			result, err := client.Configuration(ctx, &protocol.ConfigurationParams{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return false from ApplyEdit", func() {
			applied, err := client.ApplyEdit(ctx, &protocol.ApplyWorkspaceEditParams{})
			Expect(err).ToNot(HaveOccurred())
			Expect(applied).To(BeFalse())
		})

		It("should return nil from Progress", func() {
			Expect(client.Progress(ctx, &protocol.ProgressParams{})).To(Succeed())
		})

		It("should return nil from WorkDoneProgressCreate", func() {
			Expect(client.WorkDoneProgressCreate(ctx, &protocol.WorkDoneProgressCreateParams{})).To(Succeed())
		})

		It("should return nil from ShowDocument", func() {
			result, err := client.ShowDocument(ctx, &protocol.ShowDocumentParams{URI: "file:///test"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return nil from Request", func() {
			result, err := client.Request(ctx, "custom/method", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
