// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/migrate"
	. "github.com/synnaxlabs/oracle/testutil"
)

func TestGoMigrate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Migrate Plugin Suite")
}

var _ = Describe("Go Migrate Plugin", func() {
	var (
		ctx    context.Context
		loader *MockFileLoader
		p      *migrate.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
		p = migrate.New(migrate.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("go/migrate"))
		})

		It("Should filter on go domain", func() {
			Expect(p.Domains()).To(Equal([]string{"go"}))
		})

		It("Should require go/types and go/marshal", func() {
			Expect(p.Requires()).To(Equal([]string{"go/types", "go/marshal"}))
		})
	})

	Describe("Generate", func() {
		Context("uuid key type", func() {
			It("Should generate v1 snapshot and migrate file", func() {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Key = uuid

					Schematic struct {
						key Key { @key }
						name     string
						data     json
						snapshot bool

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, p)
				Expect(resp.Files).To(HaveLen(2))

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						"package v1",
						"type Key = uuid.UUID",
						"type Schematic struct {",
						`Key Key`,
						`Name string`,
						`Data binary.MsgpackEncodedJSON`,
						`Snapshot bool`,
						"func (s Schematic) GorpKey() Key { return s.Key }",
						"func (s Schematic) SetOptions() []any { return nil }",
					).
					ToNotContain(
						`"github.com/synnaxlabs/synnax/pkg/service/schematic"`,
					)

				ExpectContent(resp, "migrate.gen.go").
					ToContain(
						"package schematic",
						`"github.com/synnaxlabs/x/gorp"`,
						"func SchematicMigrations(codec gorp.Codec[Schematic]) []gorp.Migration {",
						`gorp.NewCodecTransition[Key, Schematic](`,
						`"msgpack_to_protobuf"`,
						"codec,",
					).
					ToNotContain(
						"package migrations",
					)
			})
		})

		Context("string key type", func() {
			It("Should generate correct key alias", func() {
				source := `
					@go output "core/pkg/service/device"
					@pb

					Key = string

					Device struct {
						key Key { @key }
						name string

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "device", loader, p)
				Expect(resp.Files).To(HaveLen(2))

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						"type Key = string",
						"type Device struct {",
					)

				ExpectContent(resp, "migrate.gen.go").
					ToContain(
						"func DeviceMigrations(codec gorp.Codec[Device]) []gorp.Migration {",
						"gorp.NewCodecTransition[Key, Device](",
					)
			})
		})

		Context("uint32 distinct key type", func() {
			It("Should generate correct distinct key type", func() {
				source := `
					@go output "core/pkg/service/rack"
					@pb

					Key uint32

					Rack struct {
						key Key { @key }
						name string

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)
				Expect(resp.Files).To(HaveLen(2))

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						"type Key uint32",
						"type Rack struct {",
					)

				ExpectContent(resp, "migrate.gen.go").
					ToContain(
						"func RackMigrations(codec gorp.Codec[Rack]) []gorp.Migration {",
						"gorp.NewCodecTransition[Key, Rack](",
					)
			})
		})

		Context("uint64 distinct key type", func() {
			It("Should generate correct key for uint64", func() {
				source := `
					@go output "core/pkg/service/task"
					@pb

					Key uint64

					Task struct {
						key Key { @key }
						name string
						type string

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, p)

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain("type Key uint64")

				ExpectContent(resp, "migrate.gen.go").
					ToContain("gorp.NewCodecTransition[Key, Task](")
			})
		})

		Context("optional fields", func() {
			It("Should handle hard optional fields with pointer and omitempty", func() {
				source := `
					@go output "core/pkg/service/rack"
					@pb

					Key uint32

					Rack struct {
						key      Key { @key }
						name     string
						embedded bool??

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						`Embedded *bool`,
						`"embedded,omitempty"`,
					)
			})
		})

		Context("filtering", func() {
			It("Should skip types without @go migrate", func() {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Key = uuid

					Schematic struct {
						key Key { @key }
						name string

						@go marshal
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, p)
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should skip types without @key", func() {
				source := `
					@go output "core/pkg/service/schematic"

					Schematic struct {
						name string

						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, p)
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should skip types without @go output", func() {
				source := `
					Key = uuid

					Schematic struct {
						key Key { @key }
						name string

						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, p)
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("file paths", func() {
			It("Should place v1 in migrations/v1/ and migrate in the service package", func() {
				source := `
					@go output "core/pkg/service/schematic"
					@pb

					Key = uuid

					Schematic struct {
						key Key { @key }
						name string

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, p)
				Expect(resp.Files).To(HaveLen(2))

				var paths []string
				for _, f := range resp.Files {
					paths = append(paths, f.Path)
				}
				Expect(paths).To(ContainElement("core/pkg/service/schematic/migrations/v1/v1.gen.go"))
				Expect(paths).To(ContainElement("core/pkg/service/schematic/migrate.gen.go"))
			})
		})

		Context("@go name override", func() {
			It("Should use the @go name for the struct name", func() {
				source := `
					@go output "core/pkg/service/rack"
					@pb

					Key uint32

					Rack struct {
						key Key { @key }
						name string

						@go marshal
						@go migrate
						@go name "Payload"
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						"type Payload struct {",
						"func (s Payload) GorpKey() Key",
					)

				ExpectContent(resp, "migrate.gen.go").
					ToContain(
						"func PayloadMigrations(codec gorp.Codec[Payload]) []gorp.Migration {",
						"gorp.NewCodecTransition[Key, Payload](",
					)
			})
		})

		Context("cross-namespace field types", func() {
			It("Should resolve types from imported schemas", func() {
				loader.Add("schemas/color.oracle", `
					@go output "x/go/color"

					Color = string
				`)

				source := `
					import "schemas/color"
					@go output "x/go/label"
					@pb

					Key = uuid

					Label struct {
						key   Key { @key }
						name  string
						color color.Color

						@go marshal
						@go migrate
					}
				`
				resp := MustGenerate(ctx, source, "label", loader, p)

				ExpectContent(resp, "v1/v1.gen.go").
					ToContain(
						"package v1",
						"type Label struct {",
						"Color color.Color",
					)
			})
		})
	})
})
