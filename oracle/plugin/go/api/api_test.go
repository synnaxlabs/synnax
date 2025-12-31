// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/api"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestGoAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go API Plugin Suite")
}

var _ = Describe("Go API Plugin", func() {
	var (
		ctx       context.Context
		loader    *testutil.MockFileLoader
		apiPlugin *api.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		apiPlugin = api.New(api.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("should have correct name", func() {
			Expect(apiPlugin.Name()).To(Equal("go/api"))
		})

		It("should filter on api domain", func() {
			Expect(apiPlugin.Domains()).To(Equal([]string{"api"}))
		})

		It("should require go/types and pb/types", func() {
			Expect(apiPlugin.Requires()).To(Equal([]string{"go/types", "pb/types"}))
		})
	})

	Describe("Generate", func() {
		Context("skipping", func() {
			It("should skip structs without api domain", func() {
				source := `
					@go output "core/pkg/service/user"

					User struct {
						key uuid
						name string
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
					OutputDir:   "out",
				}

				resp, err := apiPlugin.Generate(req)
				Expect(err).To(BeNil())
				Expect(resp.Files).To(BeEmpty())
			})

			It("should skip omitted structs", func() {
				source := `
					@go output "core/pkg/service/user"
					@api output "core/pkg/api/grpc"

					User struct {
						key uuid
						name string

						@api omit
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
					OutputDir:   "out",
				}

				resp, err := apiPlugin.Generate(req)
				Expect(err).To(BeNil())
				Expect(resp.Files).To(BeEmpty())
			})

			It("should skip alias when api and go outputs are the same", func() {
				source := `
					@go output "core/pkg/service/user"
					@api output "core/pkg/service/user"

					User struct {
						key uuid
						name string
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
					OutputDir:   "out",
				}

				resp, err := apiPlugin.Generate(req)
				Expect(err).To(BeNil())
				// No alias files should be generated when outputs are the same
				Expect(resp.Files).To(BeEmpty())
			})

			It("should skip alias for generic types", func() {
				source := `
					@go output "core/pkg/status"
					@api output "core/pkg/api/grpc"

					Status struct<D> {
						data D
						message string
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "status", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
					OutputDir:   "out",
				}

				resp, err := apiPlugin.Generate(req)
				Expect(err).To(BeNil())
				// Generic types cannot be aliased without instantiation
				Expect(resp.Files).To(BeEmpty())
			})

			It("should skip alias for structs with extends", func() {
				source := `
					@go output "core/pkg/user"
					@api output "core/pkg/api/grpc"

					Base struct {
						key uuid

						@api omit
					}

					User struct extends Base {
						name string
					}
				`
				table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
				Expect(diag.HasErrors()).To(BeFalse())

				req := &plugin.Request{
					Resolutions: table,
					OutputDir:   "out",
				}

				resp, err := apiPlugin.Generate(req)
				Expect(err).To(BeNil())
				// Extends structs are handled by go/types, not aliased
				Expect(resp.Files).To(BeEmpty())
			})
		})
	})
})
