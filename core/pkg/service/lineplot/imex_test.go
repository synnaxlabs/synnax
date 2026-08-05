// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// loadEnvelope reads a wire-format envelope fixture from versions/testdata and
// unmarshals it into an imex.Envelope, binding the codec that Decode needs.
func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	DescribeTable("Match",
		func(body map[string]any, expected bool) {
			Expect(svc.Match(body)).To(Equal(expected))
		},
		Entry("axes and channels", map[string]any{"axes": nil, "channels": nil}, true),
		Entry("selectedRules alone", map[string]any{"selectedRules": nil}, true),
		Entry("hiddenLines alone", map[string]any{"hiddenLines": nil}, true),
		Entry("axes without channels", map[string]any{"axes": nil}, false),
		Entry("empty body", map[string]any{}, false),
		Entry("table markers", map[string]any{"layout": nil, "cells": nil}, false),
	)

	Describe("Export", func() {
		It("Should export a line plot as a versioned envelope", func(ctx SpecContext) {
			lp := lineplot.LinePlot{Name: "exported"}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &lp)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, lineplot.OntologyID(lp.Key)))
			Expect(env.Version).To(Equal(versions.Latest))
			Expect(env.Type).To(Equal("lineplot"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(
				imex.Decode[lineplot.LinePlot](ctx, WireRoundTrip(env)),
			)
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{
				Type: ontology.ResourceTypeLineplot,
				Key:  uuid.NewString(),
			}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, path string, opts imex.ImportOptions,
		) lineplot.LinePlot {
			id := MustSucceed(imexSvc.Import(ctx, db, loadEnvelope(path), opts))
			Expect(id.Type).To(Equal(ontology.ResourceTypeLineplot))
			key := MustSucceed(uuid.Parse(id.Key))
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx,
				"versions/testdata/import_v6.json",
				imex.ImportOptions{Parent: proj.OntologyID()},
			)
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Channels.Y1).To(Equal([]channel.Key{1, 2}))
			Expect(res.Ranges.X1).To(Equal([]string{"recent"}))
			// Lines are derived from the channels and ranges bindings on create.
			Expect(res.Lines).To(HaveLen(2))
		})

		It(
			"Should import a Console typed export carrying camelCase keys",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_typed_console.json",
					imex.ImportOptions{Parent: proj.OntologyID()},
				)
				Expect(res.Name).To(Equal("Console Typed"))
				Expect(res.Channels.Y1).To(Equal([]channel.Key{3}))
				Expect(res.Axes.Y1.Label).To(Equal("speed"))
				Expect(res.Axes.Y1.ManualBounds).To(
					Equal(lineplot.ManualBounds{Lower: true, Upper: false}),
				)
			},
		)

		It(
			"Should reject a v5 Console state (rc-era, never released)",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(
					ctx,
					db,
					loadEnvelope("versions/testdata/import_v5_state.json"),
					imex.ImportOptions{
						FileName: "My Plot.json",
						Parent:   proj.OntologyID(),
					},
				)).Error().To(
					MatchError(ContainSubstring("unknown line plot data version 5")),
				)
			},
		)

		It(
			"Should import a v0 Console state through the legacy chain",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_v0_state.json",
					imex.ImportOptions{
						FileName: "Legacy Plot.json",
						Parent:   proj.OntologyID(),
					},
				)
				Expect(res.Name).To(Equal("Legacy Plot"))
				Expect(res.Channels.Y1).To(Equal([]channel.Key{9}))
				Expect(res.Axes.Y1.Label).To(Equal("pressure"))
				Expect(res.Axes.Y1.ManualBounds).To(
					Equal(lineplot.ManualBounds{Lower: true, Upper: false}),
				)
				Expect(res.Rules).To(HaveLen(1))
				Expect(res.Rules[0].Label).To(Equal("max"))
				Expect(res.Rules[0].Position).To(Equal(42.0))
			},
		)

		It("Should reject a zero parent", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_v6.json"),
				imex.ImportOptions{},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("parent")),
				MatchError(ContainSubstring("required")),
			))
		})

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					loadEnvelope("versions/testdata/import_bad_version.json"),
					imex.ImportOptions{Parent: proj.OntologyID()},
				)).Error().To(SatisfyAll(
					MatchError(ContainSubstring("lineplot version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				id := MustSucceed(imexSvc.Import(
					ctx,
					db,
					loadEnvelope(
						"versions/testdata/import_v6.json",
					),
					imex.ImportOptions{Parent: proj.OntologyID()},
				))
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)
	})

	Describe("Round trip", func() {
		It(
			"Should preserve plot content through export then import",
			func(ctx SpecContext) {
				original := lineplot.LinePlot{
					Name:     "round-trip",
					Channels: lineplot.Channels{Y1: []channel.Key{4, 5}},
					Ranges:   lineplot.Ranges{X1: []string{"recent"}},
				}
				Expect(
					svc.NewWriter(nil).Create(ctx, proj.Key, &original),
				).To(Succeed())
				env := MustSucceed(svc.Export(ctx, lineplot.OntologyID(original.Key)))
				id := MustSucceed(imexSvc.Import(
					ctx,
					db,
					WireRoundTrip(env),
					imex.ImportOptions{Parent: proj.OntologyID()},
				))
				key := MustSucceed(uuid.Parse(id.Key))
				Expect(key).ToNot(Equal(original.Key))
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().
					Where(lineplot.MatchKeys(key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("round-trip"))
				Expect(res.Channels).To(Equal(original.Channels))
				Expect(res.Ranges).To(Equal(original.Ranges))
			},
		)
	})
})
