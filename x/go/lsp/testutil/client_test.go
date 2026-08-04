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
	"time"

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

		It("should capture diagnostics from PublishDiagnostics", func(ctx SpecContext) {
			client := &testutil.MockClient{}
			diags := []protocol.Diagnostic{
				{
					Message:  protocol.String("undefined symbol: x"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Message:  protocol.String("unused variable: y"),
					Severity: protocol.DiagnosticSeverityWarning,
				},
			}
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI:         "file:///test.arc",
				Diagnostics: diags,
			})).To(Succeed())
			Expect(client.Diagnostics()).To(HaveLen(2))
			Expect(
				testutil.DiagnosticMessage(client.Diagnostics()[0]),
			).To(Equal("undefined symbol: x"))
			Expect(
				testutil.DiagnosticMessage(client.Diagnostics()[1]),
			).To(Equal("unused variable: y"))
		})

		It(
			"should replace diagnostics on subsequent PublishDiagnostics calls",
			func(ctx SpecContext) {
				client := &testutil.MockClient{}
				Expect(
					client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
						URI: "file:///test.arc",
						Diagnostics: []protocol.Diagnostic{
							{Message: protocol.String("first error")},
						},
					}),
				).To(Succeed())
				Expect(client.Diagnostics()).To(HaveLen(1))
				Expect(
					client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
						URI: "file:///test.arc",
						Diagnostics: []protocol.Diagnostic{
							{Message: protocol.String("second error")},
							{Message: protocol.String("third error")},
						},
					}),
				).To(Succeed())
				Expect(client.Diagnostics()).To(HaveLen(2))
				Expect(
					testutil.DiagnosticMessage(client.Diagnostics()[0]),
				).To(Equal("second error"))
			},
		)

		It(
			"should clear diagnostics when publishing empty slice",
			func(ctx SpecContext) {
				client := &testutil.MockClient{}
				Expect(
					client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
						URI: "file:///test.arc",
						Diagnostics: []protocol.Diagnostic{
							{Message: protocol.String("error")},
						},
					}),
				).To(Succeed())
				Expect(client.Diagnostics()).To(HaveLen(1))
				Expect(
					client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
						URI:         "file:///test.arc",
						Diagnostics: []protocol.Diagnostic{},
					}),
				).To(Succeed())
				Expect(client.Diagnostics()).To(BeEmpty())
			},
		)
	})

	Describe("Stubbed Methods", func() {
		var client *testutil.MockClient

		BeforeEach(func() {
			client = &testutil.MockClient{}
		})

		It("should return nil from ShowMessage", func(ctx SpecContext) {
			Expect(client.ShowMessage(ctx, &protocol.ShowMessageParams{
				Type:    protocol.MessageTypeInfo,
				Message: "test",
			})).To(Succeed())
		})

		It("should return nil from LogMessage", func(ctx SpecContext) {
			Expect(client.LogMessage(ctx, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeLog,
				Message: "log entry",
			})).To(Succeed())
		})

		It("should return nil from LogTrace", func(ctx SpecContext) {
			Expect(client.LogTrace(ctx, &protocol.LogTraceParams{})).To(Succeed())
		})

		It("should return nil from Telemetry", func(ctx SpecContext) {
			Expect(client.Telemetry(ctx, nil)).To(Succeed())
		})

		It("should return nil from Progress", func(ctx SpecContext) {
			Expect(client.Progress(ctx, &protocol.ProgressParams{})).To(Succeed())
		})

		It(
			"should return not-implemented for methods without overrides",
			func(ctx SpecContext) {
				Expect(
					client.ShowMessageRequest(
						ctx,
						&protocol.ShowMessageRequestParams{},
					),
				).
					Error().
					To(MatchError(ContainSubstring("not implemented")))
			},
		)
	})
})

var _ = Describe("MockClient Counters", func() {
	var client *testutil.MockClient

	BeforeEach(func() { client = &testutil.MockClient{} })

	Describe("PublishCount", func() {
		It("Should count each PublishDiagnostics call", func(ctx SpecContext) {
			Expect(client.PublishCount()).To(Equal(0))
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI: "file:///test.arc",
			})).To(Succeed())
			Expect(client.PublishCount()).To(Equal(1))
		})
	})

	Describe("WaitForDiagnostics", func() {
		It("Should observe a publish past the baseline", func(ctx SpecContext) {
			Expect(client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
				URI: "file:///test.arc",
			})).To(Succeed())
			Expect(client.WaitForDiagnostics(0, 50*time.Millisecond)).To(BeTrue())
		})

		It("Should time out when no publish happens", func() {
			Expect(client.WaitForDiagnostics(0, 20*time.Millisecond)).To(BeFalse())
		})
	})

	Describe("SemanticTokensRefresh", func() {
		It("Should count each refresh", func(ctx SpecContext) {
			Expect(client.SemanticRefreshCount()).To(Equal(0))
			Expect(client.SemanticTokensRefresh(ctx)).To(Succeed())
			Expect(client.SemanticRefreshCount()).To(Equal(1))
		})
	})

	Describe("WaitForSemanticRefresh", func() {
		It("Should observe a refresh past the baseline", func(ctx SpecContext) {
			Expect(client.SemanticTokensRefresh(ctx)).To(Succeed())
			Expect(client.WaitForSemanticRefresh(0, 50*time.Millisecond)).To(BeTrue())
		})

		It("Should time out when no refresh happens", func() {
			Expect(client.WaitForSemanticRefresh(0, 20*time.Millisecond)).To(BeFalse())
		})
	})
})
