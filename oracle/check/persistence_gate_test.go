// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("PersistenceGate", func() {
	run := func(ctx context.Context, source string) check.GateReport {
		GinkgoHelper()
		table, diag := analyzer.AnalyzeSource(
			ctx, source, "test", testutil.NewMockFileLoader())
		Expect(diag == nil || diag.Ok()).To(BeTrue())
		r := &pipeline.Result{Resolutions: table}
		return check.NewPersistenceGate(false).Run(ctx, r, check.Env{})
	}

	messages := func(r check.GateReport) []string {
		out := make([]string, len(r.Findings))
		for i, f := range r.Findings {
			out[i] = f.Message
		}
		return out
	}

	It("Should pass a versioned persisted type", func(ctx SpecContext) {
		r := run(ctx, `
			@go output "out"
			Entry struct {
			    @go version 0
				key uuid @key
				@go marshal
			}
		`)
		Expect(r.Findings).To(BeEmpty())
	})

	It(
		"Should warn on a versioned type that is never persisted",
		func(ctx SpecContext) {
			r := run(ctx, `
			@go output "out"
			Entry struct {
			    @go version 0
				key uuid @key
				@go marshal
			}
			Loose struct {
			    @go version 0
				name string
			}
		`)
			Expect(messages(r)).To(ContainElement(
				"test.Loose declares @go version but is never persisted"))
		},
	)

	It("Should not warn when the unpersisted version is pinned", func(ctx SpecContext) {
		r := run(ctx, `
			@go output "out"
			Entry struct {
			    @go version 0
				key uuid @key
				@go marshal
			}
			Loose struct {
			    @go version 0 pinned
				name string
			}
		`)
		Expect(r.Findings).To(BeEmpty())
	})

	It("Should warn when a pinned type is actually persisted", func(ctx SpecContext) {
		r := run(ctx, `
			@go output "out"
			Entry struct {
			    @go version 0 pinned
				key uuid @key
				@go marshal
			}
		`)
		Expect(messages(r)).To(ContainElement(
			"test.Entry pins its @go version but is persisted"))
	})

	It(
		"Should warn on a persisted type without a version at a versioned path",
		func(ctx SpecContext) {
			r := run(ctx, `
			@go output "out"
			Entry struct {
			    @go version 0
				key uuid @key
				sibling Sibling
				@go marshal
			}
			Sibling struct {
				name string
			}
		`)
			Expect(messages(r)).To(ContainElement(
				"test.Sibling is persisted but lacks @go version at a versioned path"))
		},
	)
})
