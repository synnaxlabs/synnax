// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	DescribeTable("Match",
		func(body map[string]any, expected bool) {
			Expect(svc.Match(body)).To(Equal(expected))
		},
		Entry("nodes and props", map[string]any{"nodes": nil, "props": nil}, true),
		Entry("nodes without props", map[string]any{"nodes": nil}, false),
		Entry("props without nodes", map[string]any{"props": nil}, false),
		Entry("empty body", map[string]any{}, false),
		Entry("table markers", map[string]any{"layout": nil, "cells": nil}, false),
	)

	Describe("Export", func() {
		It("Should export a schematic as a versioned envelope", func(ctx SpecContext) {
			s := schematic.Schematic{Project: &proj.Key, Name: "exported"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, schematic.ResourceID(s.Key)))
			Expect(env.Version).To(Equal(versions.Latest))
			Expect(env.Type).To(Equal("schematic"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(
				WireRoundTrip(env).Decode[schematic.Schematic](ctx),
			)
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := resource.ID{
				Type: resource.TypeSchematic,
				Key:  uuid.New().String(),
			}
			Expect(svc.Export(ctx, db, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := resource.ID{Type: resource.TypeSchematic, Key: "not-a-uuid"}
			Expect(
				svc.Export(ctx, db, id),
			).Error().
				To(MatchError(ContainSubstring("uuid")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, path string, opts imex.ImportOptions,
		) schematic.Schematic {
			GinkgoHelper()
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = imexSvc.Import(ctx, tx, LoadEnvelope(path), opts)
				return err
			})).To(Succeed())
			Expect(id.Type).To(Equal(resource.TypeSchematic))
			key := MustSucceed(uuid.Parse(id.Key))
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx,
				"versions/testdata/import_v7.json",
				imex.ImportOptions{Parent: proj.ResourceID()},
			)
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(4)))
			Expect(res.Edges[0].Source).To(
				Equal(schematic.Handle{Node: "n1", Param: "out"}),
			)
			Expect(res.Configs).To(HaveKey("n1"))
		})

		It(
			"Should import a v6 Console typed export carrying camelCase keys",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_typed_console.json",
					imex.ImportOptions{Parent: proj.ResourceID()},
				)
				Expect(res.Name).To(Equal("Console Typed"))
				Expect(res.Nodes[0].ZIndex).To(Equal(int16(4)))
				cfg, ok := res.Configs["n1"].Variant.(schematic.ValveElementConfig)
				Expect(ok).To(BeTrue())
				Expect(cfg.OnClickDelay).To(HaveValue(Equal(2.0)))
			},
		)

		It(
			"Should import a v3 Console state through the legacy chain",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_v3_state.json",
					imex.ImportOptions{
						FileName: "Legacy Schematic.json",
						Parent:   proj.ResourceID(),
					},
				)
				Expect(res.Name).To(Equal("Legacy Schematic"))
				Expect(res.Nodes[0].ZIndex).To(Equal(int16(3)))
				Expect(res.Edges[0].Source).To(
					Equal(schematic.Handle{Node: "n1", Param: "a"}),
				)
				Expect(res.Edges[0].Target).To(
					Equal(schematic.Handle{Node: "n2", Param: "b"}),
				)
				cfg, ok := res.Configs["n1"].Variant.(schematic.ValveElementConfig)
				Expect(ok).To(BeTrue())
				Expect(cfg.StrokeColor).
					To(HaveValue(Equal(MustSucceed(color.FromHex("#ff0000")))))
			},
		)

		It(
			"Should refuse a group that is not in the parent project",
			func(ctx SpecContext) {
				other := project.Project{Name: "other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return projectSvc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				g := group.Group{
					Name:   "in other",
					Parent: new(project.ResourceID(other.Key)),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := imexSvc.Import(
						ctx,
						tx,
						LoadEnvelope("versions/testdata/import_v7.json"),
						imex.ImportOptions{Parent: proj.ResourceID(), Group: &g.Key},
					)
					return err
				})).To(SatisfyAll(
					MatchError(ContainSubstring(fmt.Sprintf(
						"group: group %s is not in project %s", g.Key, proj.Key,
					))),
					MatchError(ContainSubstring("validation error")),
				))
				Expect(svc.NewRetrieve().
					Where(schematic.MatchGroups(g.Key)).
					Count(ctx, db)).To(BeZero())
			},
		)

		It("Should reject a zero parent", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := imexSvc.Import(
					ctx,
					tx,
					LoadEnvelope("versions/testdata/import_v7.json"),
					imex.ImportOptions{},
				)
				return err
			})).To(SatisfyAll(
				MatchError(ContainSubstring(`parent: must be a project, got ""`)),
				MatchError(ContainSubstring("validation error")),
			))
		})

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := imexSvc.Import(
						ctx,
						tx,
						LoadEnvelope("versions/testdata/import_bad_version.json"),
						imex.ImportOptions{Parent: proj.ResourceID()},
					)
					return err
				})).To(SatisfyAll(
					MatchError(ContainSubstring("schematic version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				var id resource.ID
				Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
					id, err = imexSvc.Import(ctx, tx, LoadEnvelope(
						"versions/testdata/import_v7.json",
					),
						imex.ImportOptions{Parent: proj.ResourceID()})
					return err
				})).To(Succeed())
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)
	})

	Describe("Round trip", func() {
		It(
			"Should preserve schematic content through export then import",
			func(ctx SpecContext) {
				original := schematic.Schematic{
					Project: &proj.Key,
					Name:    "round-trip",
					Nodes:   []schematic.Node{{Key: "n1", ZIndex: 5}},
					Edges: []schematic.Edge{{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1", Param: "out"},
						Target: schematic.Handle{Node: "n2", Param: "in"},
					}},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &original)
				})).To(Succeed())
				env := MustSucceed(
					svc.Export(ctx, db, schematic.ResourceID(original.Key)),
				)
				var id resource.ID
				Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
					id, err = imexSvc.Import(ctx, tx, WireRoundTrip(env),
						imex.ImportOptions{Parent: proj.ResourceID()})
					return err
				})).To(Succeed())
				key := MustSucceed(uuid.Parse(id.Key))
				Expect(key).ToNot(Equal(original.Key))
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("round-trip"))
				Expect(res.Nodes).To(Equal(original.Nodes))
				Expect(res.Edges).To(Equal(original.Edges))
			},
		)
	})
})
