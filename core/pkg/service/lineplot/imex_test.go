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
	"encoding/json/v2"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// envelope unmarshals the wire-format envelope in raw.
func envelope(raw string) imex.Envelope {
	GinkgoHelper()
	var env imex.Envelope
	Expect(json.Unmarshal([]byte(raw), &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	DescribeTable("Match",
		func(body map[string]any, expected bool) {
			Expect(svc.Match(body)).To(Equal(expected))
		},
		Entry("axes and channels", map[string]any{"axes": nil, "channels": nil}, true),
		Entry("channels without axes", map[string]any{"channels": nil}, false),
		Entry("axes without channels", map[string]any{"axes": nil}, false),
		Entry("empty body", map[string]any{}, false),
		Entry("table markers", map[string]any{"layout": nil, "cells": nil}, false),
	)

	Describe("Export", func() {
		It("Should export a line plot as a versioned envelope", func(ctx SpecContext) {
			lp := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "exported"})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &lp)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, lineplot.ResourceID(lp.Key)))
			Expect(env.Version).To(Equal(versions.Latest))
			Expect(env.Type).To(Equal("lineplot"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(
				WireRoundTrip(env).Decode[lineplot.LinePlot](ctx),
			)
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := resource.ID{
				Type: resource.TypeLineplot,
				Key:  uuid.New().String(),
			}
			Expect(svc.Export(ctx, db, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := resource.ID{Type: resource.TypeLineplot, Key: "not-a-uuid"}
			Expect(
				svc.Export(ctx, db, id),
			).Error().
				To(MatchError(ContainSubstring("uuid")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, env imex.Envelope, opts imex.ImportOptions,
		) lineplot.LinePlot {
			GinkgoHelper()
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = imexSvc.Import(ctx, tx, env, opts)
				return err
			})).To(Succeed())
			Expect(id.Type).To(Equal(resource.TypeLineplot))
			key := MustSucceed(uuid.Parse(id.Key))
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(ctx, envelope(fmt.Sprintf(`{
				"version": 5,
				"type": "lineplot",
				"key": "11111111-2222-3333-4444-555555555555",
				"name": "Server Typed",
				"title": {"level": "h4", "visible": true},
				"legend": {"hidden": false, "position": {"x": 50, "y": 50}},
				"channels": {
					"x1": 0, "x2": 0, "y1": [%[1]d, %[2]d], "y2": [], "y3": [], "y4": []
				},
				"ranges": {"x1": ["recent"], "x2": []},
				"lines": [{
					"key": "y1---x1---recent---0---%[1]d",
					"color": "#ff0000",
					"stroke_width": 2.5,
					"downsample": 1,
					"downsample_mode": "decimate"
				}],
				"rules": []
			}`, ch1, ch2)), imex.ImportOptions{Parent: proj.ResourceID()})
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Channels.Y1).To(Equal([]channel.Key{ch1, ch2}))
			Expect(res.Ranges).To(Equal([]ranger.Ref{recentRef}))
			// Lines are derived from the channels and ranges bindings on create.
			Expect(res.Lines).To(HaveLen(2))
		})

		It(
			"Should import a v0 Console state through the legacy chain",
			func(ctx SpecContext) {
				res := importAndRetrieve(ctx, envelope(fmt.Sprintf(`{
					"key": "abc123",
					"version": "0.0.0",
					"remoteCreated": false,
					"title": {"level": "p", "visible": true},
					"legend": {"visible": true},
					"channels": {
						"x1": 0, "x2": 0, "y1": [%d], "y2": [], "y3": [], "y4": []
					},
					"ranges": {"x1": ["recent"], "x2": []},
					"axes": {
						"renderTrigger": 0,
						"hasHadChannelSet": true,
						"axes": {
							"y1": {
								"key": "y1",
								"label": "pressure",
								"labelDirection": "x",
								"labelLevel": "small",
								"bounds": {"lower": 0, "upper": 100},
								"autoBounds": {"lower": false, "upper": true},
								"tickSpacing": 75
							}
						}
					},
					"lines": [{
						"key": "l1",
						"color": "#ff0000",
						"strokeWidth": 2,
						"downsample": 1,
						"downsampleMode": "decimate"
					}],
					"rules": [{
						"key": "r1",
						"label": "max",
						"color": "#00ff00",
						"axis": "y1",
						"lineWidth": 1,
						"lineDash": 5,
						"units": "psi",
						"position": 42,
						"selected": true
					}]
				}`, ch9)), imex.ImportOptions{
					FileName: "Legacy Plot.json",
					Parent:   proj.ResourceID(),
				})
				Expect(res.Name).To(Equal("Legacy Plot"))
				Expect(res.Channels.Y1).To(Equal([]channel.Key{ch9}))
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := imexSvc.Import(
					ctx,
					tx,
					LoadEnvelope("versions/testdata/import_v5.json"),
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
					MatchError(ContainSubstring("lineplot version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
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
						LoadEnvelope("versions/testdata/import_v5.json"),
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
					Where(lineplot.MatchGroups(g.Key)).
					Count(ctx, db)).To(BeZero())
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				var id resource.ID
				Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
					id, err = imexSvc.Import(ctx, tx, LoadEnvelope(
						"versions/testdata/import_v5.json",
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
			"Should preserve plot content through export then import",
			func(ctx SpecContext) {
				original := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "round-trip",
					Channels: lineplot.Channels{Y1: []channel.Key{ch4, ch5}},
					Ranges:   []ranger.Ref{recentRef},
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &original)
				})).To(Succeed())
				env := MustSucceed(
					svc.Export(ctx, db, lineplot.ResourceID(original.Key)),
				)
				var id resource.ID
				Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
					id, err = imexSvc.Import(ctx, tx, WireRoundTrip(env),
						imex.ImportOptions{Parent: proj.ResourceID()})
					return err
				})).To(Succeed())
				key := MustSucceed(uuid.Parse(id.Key))
				Expect(key).ToNot(Equal(original.Key))
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().
					Where(lineplot.MatchKeys(key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("round-trip"))
				// A required collection always serializes, so an axis left nil comes
				// back allocated and empty rather than nil.
				Expect(res.Channels).To(Equal(lineplot.Channels{
					Y1: []channel.Key{ch4, ch5},
					Y2: []channel.Key{},
					Y3: []channel.Key{},
					Y4: []channel.Key{},
				}))
				Expect(res.Ranges).To(Equal([]ranger.Ref{recentRef}))
			},
		)
	})
})
