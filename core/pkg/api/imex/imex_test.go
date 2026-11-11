// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

// recordingImporter captures the ImportOptions the API layer hands to the registry so
// specs can assert on how the out-of-band request params were parsed.
type recordingImporter struct {
	opts imex.ImportOptions
}

func (*recordingImporter) Type() resource.Type {
	return resource.TypeChannel
}

func (r *recordingImporter) Decode(
	_ context.Context,
	env imex.Envelope,
) (imex.Write, error) {
	return func(
		_ context.Context,
		_ gorp.Tx,
		opts imex.ImportOptions,
	) (resource.ID, error) {
		r.opts = opts
		return resource.ID{Type: resource.TypeChannel, Key: env.Name}, nil
	}, nil
}

// Match claims typeless envelopes carrying the legacy_marker key so specs can exercise
// type resolution.
func (*recordingImporter) Match(body map[string]any) bool {
	_, ok := body["legacy_marker"]
	return ok
}

// namingExporter exports a channel as an envelope named by the key of the channel.
type namingExporter struct{}

func (namingExporter) Type() resource.Type { return resource.TypeChannel }

func (namingExporter) Export(
	_ context.Context,
	_ gorp.Snapshot,
	id resource.ID,
) (imex.Envelope, error) {
	return imex.Envelope{Version: 1, Type: string(id.Type), Name: id.Key}, nil
}

func testEnvelope(name string) apiimex.ImportRequest {
	GinkgoHelper()
	var env imex.Envelope
	Expect(json.Unmarshal(fmt.Appendf(nil,
		`{"version":1,"type":%q,"name":%q}`, resource.TypeChannel, name,
	), &env)).To(Succeed())
	return env
}

var _ = Describe("Import", func() {
	It(
		"Should hand the file_name, parent, and group params to the importer",
		func(ctx SpecContext) {
			key := uuid.New()
			groupKey := uuid.New()
			fctx := rootCtx(ctx)
			fctx.Set("params", fmt.Sprintf(
				`{"file_name":"Metrics Log.json","parent":"project:%s","group":%q}`,
				key,
				groupKey,
			))
			id := MustSucceed(importEnvelope(fctx, testEnvelope("with-options")))
			Expect(id.Key).To(Equal("with-options"))
			Expect(importer.opts.FileName).To(Equal("Metrics Log.json"))
			Expect(importer.opts.Parent).To(Equal(resource.ID{
				Type: resource.TypeProject, Key: key.String(),
			}))
			Expect(importer.opts.Group).To(Equal(&groupKey))
		},
	)

	It(
		"Should hand no parent and no group to the importer when params omit them",
		func(ctx SpecContext) {
			fctx := rootCtx(ctx)
			fctx.Set("params", `{"file_name":"Metrics Log.json"}`)
			MustSucceed(importEnvelope(fctx, testEnvelope("no-parent")))
			Expect(importer.opts.Parent.IsZero()).To(BeTrue())
			Expect(importer.opts.Group).To(BeNil())
		},
	)

	It(
		"Should resolve a typeless envelope through the registered matcher",
		func(ctx SpecContext) {
			var env imex.Envelope
			Expect(json.Unmarshal(
				[]byte(`{"version":"1.0.0","legacy_marker":true}`), &env,
			)).To(Succeed())
			fctx := rootCtx(ctx)
			fctx.Set("params", fmt.Sprintf(
				`{"file_name":"Legacy State.json","parent":"project:%s"}`, uuid.New(),
			))
			id := MustSucceed(
				importEnvelope(fctx, env),
			)
			Expect(id.Key).To(Equal("Legacy State"))
		},
	)

	It(
		"Should need no access on a parent that the importer does not write",
		func(ctx SpecContext) {
			u := MustSucceed(
				rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
					Name: "parentless-" + uuid.New().String(),
				}),
			)
			fctx := subjectCtx(ctx, u)
			fctx.Set("params", fmt.Sprintf(
				`{"file_name":"Metrics Log.json","parent":"project:%s"}`,
				uuid.New(),
			))
			id := MustSucceed(importEnvelope(fctx, testEnvelope("ignored-parent")))
			Expect(id.Key).To(Equal("ignored-parent"))
		},
	)

	It("Should reject a typeless envelope no matcher claims", func(ctx SpecContext) {
		var env imex.Envelope
		Expect(json.Unmarshal(
			[]byte(`{"version":"1.0.0","unclaimed":true}`), &env,
		)).To(Succeed())
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Mystery.json","parent":"project:%s"}`, uuid.New(),
		))
		Expect(importEnvelope(fctx, env)).Error().To(
			MatchError(ContainSubstring("does not match any known resource format")),
		)
	})

	It("Should reject a request with no params", func(ctx SpecContext) {
		Expect(importEnvelope(rootCtx(ctx), testEnvelope("no-params"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("params")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject empty params", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", "")
		Expect(importEnvelope(fctx, testEnvelope("empty-params"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("params")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject params without a file_name", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(`{"parent":"project:%s"}`, uuid.New()))
		Expect(importEnvelope(fctx, testEnvelope("no-file-name"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("file_name")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject a parent with an empty key", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json","parent":"project:"}`)
		Expect(importEnvelope(fctx, testEnvelope("empty-parent-key"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("parent")),
				MatchError(ContainSubstring("must carry a non-empty key")),
			))
	})

	It("Should reject a parent that is not a valid resource ID", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json","parent":"no-colon"}`)
		Expect(importEnvelope(fctx, testEnvelope("bad-parent"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("failed to parse id: no-colon")),
				MatchError(ContainSubstring("validation error")),
			))
	})

	It("Should reject params that are not valid JSON", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", "not-json")
		Expect(importEnvelope(fctx, testEnvelope("bad-params"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("invalid params")),
				MatchError(ContainSubstring("validation error")),
			))
	})

	Context("With an importer that creates a record", func() {
		var (
			u user.User
			g group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = MustSucceed(
				rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
					Name: "importer-" + uuid.New().String(),
				}),
			)
			g = group.Group{Name: "Imported", Scope: resource.TypeSchematicSymbol}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		grant := func(ctx SpecContext, action access.Action, object resource.ID) {
			GinkgoHelper()
			MustSucceed(rbactestutil.Grant(
				ctx, db, rbacSvc, userSvc, u.Key, []access.Action{action}, object,
			))
		}
		importSymbol := func(ctx SpecContext) (resource.ID, error) {
			var env imex.Envelope
			Expect(json.Unmarshal([]byte(
				`{"version":2,"type":"schematic_symbol",`+
					`"data":{"svg":"<svg/>","variant":"valve"}}`,
			), &env)).To(Succeed())
			fctx := subjectCtx(ctx, u)
			fctx.Set("params", fmt.Sprintf(
				`{"file_name":"Inlet.json","group":%q}`, g.Key,
			))
			return importEnvelope(fctx, env)
		}
		symbolType := resource.ID{Type: resource.TypeSchematicSymbol}
		importLog := func(ctx SpecContext) (resource.ID, error) {
			var env imex.Envelope
			Expect(json.Unmarshal(fmt.Appendf(nil,
				`{"version":2,"type":"log","name":"Pressures",`+
					`"channels":[{"channel":%d}]}`,
				ch,
			), &env)).To(Succeed())
			fctx := subjectCtx(ctx, u)
			fctx.Set("params", fmt.Sprintf(
				`{"file_name":"Pressures.json","parent":"project:%s"}`, proj.Key,
			))
			return importEnvelope(fctx, env)
		}
		grantLogImport := func(ctx SpecContext) {
			GinkgoHelper()
			grant(ctx, access.ActionCreate, resource.ID{Type: resource.TypeLog})
			grant(ctx, access.ActionUpdate, project.ResourceID(proj.Key))
		}

		It(
			"Should import the record with create on it and update on its group",
			func(ctx SpecContext) {
				grant(ctx, access.ActionCreate, symbolType)
				grant(ctx, access.ActionUpdate, g.ResourceID())
				Expect(MustSucceed(importSymbol(ctx)).Type).
					To(Equal(resource.TypeSchematicSymbol))
			},
		)
		It("Should reject the request without create on the record", func(
			ctx SpecContext,
		) {
			grant(ctx, access.ActionUpdate, g.ResourceID())
			Expect(importSymbol(ctx)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"%s cannot create %s:",
					u.ResourceID(),
					resource.TypeSchematicSymbol,
				)),
			))
		})
		It("Should reject the request without update on its group", func(
			ctx SpecContext,
		) {
			grant(ctx, access.ActionCreate, symbolType)
			Expect(importSymbol(ctx)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"%s cannot update %s",
					u.ResourceID(),
					g.ResourceID(),
				)),
			))
		})
		It("Should import a log with retrieve on the channel it uses", func(
			ctx SpecContext,
		) {
			grantLogImport(ctx)
			grant(ctx, access.ActionRetrieve, channel.ResourceID(ch))
			Expect(MustSucceed(importLog(ctx)).Type).To(Equal(resource.TypeLog))
		})
		It("Should reject a log without retrieve on the channel it uses", func(
			ctx SpecContext,
		) {
			grantLogImport(ctx)
			Expect(importLog(ctx)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"%s cannot retrieve %s",
					u.ResourceID(),
					channel.ResourceID(ch),
				)),
			))
		})
	})
})

var _ = Describe("Export", func() {
	id := resource.ID{Type: resource.TypeChannel, Key: "12"}

	It("Should export a resource the subject can retrieve", func(ctx SpecContext) {
		Expect(MustSucceed(export(rootCtx(ctx), apiimex.ExportRequest(id))).Name).
			To(Equal("12"))
	})

	It("Should deny a resource the subject cannot retrieve", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "exporter-" + uuid.New().String(),
		}))
		Expect(export(subjectCtx(ctx, u), apiimex.ExportRequest(id))).Error().To(
			SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(
					ContainSubstring("%s cannot retrieve %s", u.ResourceID(), id),
				),
			),
		)
	})
})

var _ = Describe("JSONCodec", func() {
	It("Should write the same bytes for the same value", func(ctx SpecContext) {
		v := map[string]int{"z": 1, "a": 2, "m": 3, "b": 4, "q": 5}
		first := MustSucceed(apiimex.JSONCodec.Encode(v))
		for range 20 {
			Expect(apiimex.JSONCodec.Encode(v)).To(Equal(first))
		}
		Expect(string(first)).To(ContainSubstring("\"a\": 2,\n  \"b\": 4"))
	})
})

var _ = Describe("ImportCodec", func() {
	It("Should replace a lone surrogate with U+FFFD", func(ctx SpecContext) {
		var env imex.Envelope
		Expect(apiimex.ImportCodec.Decode(
			[]byte(`{"version":1,"name":"Tank \ud83d"}`), &env,
		)).To(Succeed())
		Expect(env.Name).To(Equal("Tank \uFFFD"))
	})
})

var _ = Describe("ResolveEncoding", func() {
	It("Should return a pretty JSON encoder for the JSON encoding", func(
		ctx SpecContext,
	) {
		enc := MustSucceed(apiimex.ResolveEncoding(apiimex.EncodingJSON))
		Expect(enc.Extension()).To(Equal(".json"))
		b := MustSucceed(enc.Encode(map[string]int{"value": 1}))
		Expect(string(b)).To(Equal("{\n  \"value\": 1\n}\n"))
	})
	It("Should write markup literally rather than escaped", func(ctx SpecContext) {
		enc := MustSucceed(apiimex.ResolveEncoding(apiimex.EncodingJSON))
		b := MustSucceed(enc.Encode(map[string]string{"svg": `<svg id="a"/>`}))
		Expect(string(b)).To(ContainSubstring(`"<svg id=\"a\"/>"`))
	})
	It("Should write markup in an envelope body literally", func(ctx SpecContext) {
		enc := MustSucceed(apiimex.ResolveEncoding(apiimex.EncodingJSON))
		type symbol struct {
			SVG string `json:"svg"`
		}
		env := imex.Envelope{Version: 1, Type: "symbol", Name: "valve"}
		Expect(env.Encode(symbol{SVG: `<svg id="a"/>`})).To(Succeed())
		Expect(string(MustSucceed(enc.Encode(env)))).
			To(ContainSubstring(`"<svg id=\"a\"/>"`))
	})
	It("Should reject an unsupported encoding", func() {
		Expect(apiimex.ResolveEncoding("YAML")).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("encoding")),
				MatchError(ContainSubstring("YAML")),
			))
	})
})
