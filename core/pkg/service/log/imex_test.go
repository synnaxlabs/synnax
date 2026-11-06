// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
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
	createLog := func(ctx SpecContext, l log.Log) log.Log {
		GinkgoHelper()
		l.Project = &proj.Key
		Expect(svc.NewWriter(nil).Create(ctx, &l)).To(Succeed())
		return l
	}

	DescribeTable(
		"Match",
		func(body map[string]any, expected bool) {
			Expect(svc.Match(body)).To(Equal(expected))
		},
		Entry("channels array", map[string]any{"channels": []any{}}, true),
		Entry(
			"channels not an array",
			map[string]any{"channels": map[string]any{}},
			false,
		),
		Entry("empty body", map[string]any{}, false),
		Entry("table markers", map[string]any{"layout": nil, "cells": nil}, false),
	)

	Describe("Export", func() {
		It(
			"Should export a created log as a versioned envelope",
			func(ctx SpecContext) {
				l := createLog(ctx, log.Log{
					Name: "exported",
					Channels: []log.ChannelEntry{
						{
							Channel: ch1,
							Color:   MustSucceed(color.FromHex("#ff0000")),
						},
					},
					TimestampPrecision: 2,
					ChannelNamesHidden: true,
				})
				env := MustSucceed(imexSvc.Export(ctx, l.ResourceID()))
				Expect(env.Version).To(Equal(versions.Latest))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("exported"))

				decoded := MustSucceed(WireRoundTrip(env).Decode[log.Log](ctx))
				Expect(decoded.Name).To(Equal("exported"))
				Expect(decoded.TimestampPrecision).To(Equal(int32(2)))
				Expect(decoded.ChannelNamesHidden).To(BeTrue())
				Expect(decoded.Channels).To(HaveLen(1))
				Expect(decoded.Channels[0].Channel).To(Equal(ch1))
			},
		)

		It(
			"Should return not found when the log does not exist",
			func(ctx SpecContext) {
				id := resource.ID{
					Type: resource.TypeLog,
					Key:  uuid.New().String(),
				}
				Expect(
					imexSvc.Export(ctx, id),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should return an error when the id key is not a valid UUID",
			func(ctx SpecContext) {
				id := resource.ID{Type: resource.TypeLog, Key: "not-a-uuid"}
				Expect(
					imexSvc.Export(ctx, id),
				).Error().
					To(MatchError(ContainSubstring("uuid")))
			},
		)
	})

	Describe("Import", func() {
		const v2Fixture = "versions/testdata/import_v2.json"

		importAndRetrieve := func(
			ctx SpecContext,
			env imex.Envelope,
		) (resource.ID, log.Log) {
			GinkgoHelper()
			id := MustSucceed(
				imexSvc.Import(
					ctx,
					db,
					env,
					imex.ImportOptions{Parent: proj.ResourceID()},
				),
			)
			Expect(id.Type).To(Equal(resource.TypeLog))
			key := MustSucceed(uuid.Parse(id.Key))
			var res log.Log
			Expect(
				svc.NewRetrieve().Where(log.MatchKeys(key)).Entry(&res).Exec(ctx, db),
			).
				To(Succeed())
			return id, res
		}

		It(
			"Should import a legacy v0 envelope, lifting bare channel keys forward",
			func(ctx SpecContext) {
				_, res := importAndRetrieve(ctx, envelope(fmt.Sprintf(`{
					"version": "0.0.0",
					"type": "log",
					"key": "660e8400-e29b-41d4-a716-446655440000",
					"name": "Test Log V0",
					"channels": [%d, %d, %d],
					"remoteCreated": true
				}`, ch1, ch2, ch3)))
				Expect(res.Name).To(Equal("Test Log V0"))
				Expect(res.Channels).To(HaveLen(3))
				Expect(res.Channels[0].Channel).To(Equal(ch1))
				Expect(res.Channels[1].Channel).To(Equal(ch2))
				Expect(res.Channels[2].Channel).To(Equal(ch3))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationStandard))
			},
		)

		It(
			"Should import a typeless v0 state through the body matcher",
			func(ctx SpecContext) {
				id := MustSucceed(imexSvc.Import(
					ctx,
					db,
					envelope(fmt.Sprintf(`{
						"version": "0.0.0",
						"key": "770e8400-e29b-41d4-a716-446655440000",
						"channels": [%d, %d],
						"remoteCreated": false
					}`, ch4, ch5)),
					imex.ImportOptions{
						FileName: "Legacy Log.json",
						Parent:   proj.ResourceID(),
					},
				))
				Expect(id.Type).To(Equal(resource.TypeLog))
				key := MustSucceed(uuid.Parse(id.Key))
				var res log.Log
				Expect(
					svc.NewRetrieve().
						Where(log.MatchKeys(key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Name).To(Equal("Legacy Log"))
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(ch4))
				Expect(res.Channels[1].Channel).To(Equal(ch5))
			},
		)

		It(
			"Should import a legacy v1 envelope carrying camelCase keys",
			func(ctx SpecContext) {
				_, res := importAndRetrieve(ctx, envelope(fmt.Sprintf(`{
					"version": "1.0.0",
					"type": "log",
					"key": "550e8400-e29b-41d4-a716-446655440000",
					"name": "Test Log V1",
					"channels": [
						{
							"channel": %d,
							"color": "#ff0000",
							"notation": "scientific",
							"precision": 2,
							"alias": "temp"
						},
						{
							"channel": %d,
							"color": "",
							"notation": "standard",
							"precision": -1,
							"alias": ""
						}
					],
					"remoteCreated": false,
					"timestampPrecision": 1,
					"showChannelNames": true,
					"showReceiptTimestamp": false
				}`, ch1, ch5)))
				Expect(res.Name).To(Equal("Test Log V1"))
				Expect(res.TimestampPrecision).To(Equal(int32(1)))
				Expect(res.ChannelNamesHidden).To(BeFalse())
				Expect(res.TimestampsHidden).To(BeTrue())
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(ch1))
				Expect(
					res.Channels[0].Color,
				).To(Equal(MustSucceed(color.FromHex("#ff0000"))))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
				Expect(res.Channels[0].Precision).To(Equal(new(int32(2))))
				Expect(res.Channels[0].Alias).To(Equal("temp"))
				Expect(res.Channels[1].Channel).To(Equal(ch5))
			},
		)

		It(
			"Should import a v2 envelope carrying snake_case keys",
			func(ctx SpecContext) {
				_, res := importAndRetrieve(ctx, envelope(fmt.Sprintf(`{
					"version": 2,
					"type": "log",
					"key": "11111111-2222-3333-4444-555555555555",
					"name": "Test Log V2",
					"channels": [
						{
							"channel": %d,
							"color": {"r": 127, "g": 29, "b": 29, "a": 1},
							"notation": "scientific",
							"precision": 2,
							"alias": "temp",
							"timestamp": {"format": "ISO", "tz": "UTC"}
						},
						{
							"channel": %d,
							"color": {"r": 0, "g": 0, "b": 0, "a": 0},
							"notation": "standard",
							"precision": -1,
							"alias": "",
							"timestamp": {"format": "", "tz": ""}
						}
					],
					"timestamp_precision": 1,
					"channel_names_hidden": false,
					"receipt_timestamp_hidden": true
				}`, ch1, ch5)))
				Expect(res.Name).To(Equal("Test Log V2"))
				Expect(res.TimestampPrecision).To(Equal(int32(1)))
				Expect(res.ChannelNamesHidden).To(BeFalse())
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(ch1))
				Expect(
					res.Channels[0].Color,
				).To(Equal(MustSucceed(color.FromHex("#7f1d1d"))))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
				Expect(res.Channels[0].Precision).To(Equal(new(int32(2))))
				Expect(res.Channels[0].Alias).To(Equal("temp"))
				Expect(
					res.Channels[0].Timestamp.Format,
				).To(Equal(telem.TimestampFormatISO))
				Expect(res.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
				Expect(res.Channels[1].Channel).To(Equal(ch5))
			},
		)

		It("Should register the imported log in the ontology", func(ctx SpecContext) {
			id, _ := importAndRetrieve(ctx, LoadEnvelope(v2Fixture))
			var resource ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(id).Entry(&resource).Exec(ctx, db)).
				To(Succeed())
			Expect(resource.ID).To(Equal(id))
		})

		It(
			"Should give the imported log the parent project from the options",
			func(ctx SpecContext) {
				_, l := importAndRetrieve(ctx, LoadEnvelope(v2Fixture))
				Expect(l.Project).To(HaveValue(Equal(proj.Key)))
				Expect(l.Range).To(BeNil())
			},
		)

		It("Should reject a parent project that does not exist", func(ctx SpecContext) {
			missing := uuid.New()
			Expect(imexSvc.Import(
				ctx, db, LoadEnvelope(v2Fixture),
				imex.ImportOptions{Parent: project.ResourceID(missing)},
			)).Error().To(MatchError(
				ContainSubstring(
					"project: project " + missing.String() + " does not exist",
				),
			))
		})

		It("Should reject a parent that is not a project", func(ctx SpecContext) {
			Expect(imexSvc.Import(
				ctx, db, LoadEnvelope(v2Fixture),
				imex.ImportOptions{Parent: resource.ID{Type: "group", Key: "g1"}},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("parent: must be a project")),
				MatchError(ContainSubstring("validation error")),
			))
		})

		It(
			"Should name the imported log from the file name when the body has no name",
			func(ctx SpecContext) {
				raw := MustSucceed(os.ReadFile(v2Fixture))
				var body map[string]any
				Expect(json.Unmarshal(raw, &body)).To(Succeed())
				delete(body, "name")
				var env imex.Envelope
				Expect(
					json.Unmarshal(MustSucceed(json.Marshal(body)), &env),
				).To(Succeed())
				id := MustSucceed(imexSvc.Import(
					ctx,
					db,
					env,
					imex.ImportOptions{
						FileName: "From File.json",
						Parent:   proj.ResourceID(),
					},
				))
				key := MustSucceed(uuid.Parse(id.Key))
				var res log.Log
				Expect(
					svc.NewRetrieve().
						Where(log.MatchKeys(key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Name).To(Equal("From File"))
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				id := MustSucceed(
					imexSvc.Import(
						ctx,
						db,
						LoadEnvelope(v2Fixture),
						imex.ImportOptions{Parent: proj.ResourceID()},
					),
				)
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)

		It("Should reject a zero parent", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				LoadEnvelope(v2Fixture),
				imex.ImportOptions{},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring(`parent: must be a project, got ""`)),
				MatchError(ContainSubstring("validation error")),
			))
		})

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					LoadEnvelope("versions/testdata/import_bad_version.json"),
					imex.ImportOptions{Parent: proj.ResourceID()},
				)).Error().To(SatisfyAll(
					MatchError(ContainSubstring("log version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
			},
		)

		It(
			"Should return an error when a current-version body cannot be decoded",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					LoadEnvelope("versions/testdata/import_bad_v2.json"),
					imex.ImportOptions{Parent: proj.ResourceID()},
				)).Error().To(MatchError(ContainSubstring("decode")))
			},
		)
	})

	Describe("Round trip", func() {
		It(
			"Should preserve log content through export then import",
			func(ctx SpecContext) {
				original := createLog(ctx, log.Log{
					Name: "round-trip",
					Channels: []log.ChannelEntry{
						{
							Channel:   ch1,
							Alias:     "first",
							Precision: new(int32(4)),
						},
						{Channel: ch2, Alias: "second"},
					},
					TimestampPrecision: 1,
					TimestampsHidden:   true,
				})
				env := MustSucceed(imexSvc.Export(ctx, original.ResourceID()))
				id := MustSucceed(
					imexSvc.Import(
						ctx,
						db,
						WireRoundTrip(env),
						imex.ImportOptions{Parent: proj.ResourceID()},
					),
				)

				key := MustSucceed(uuid.Parse(id.Key))
				var res log.Log
				Expect(
					svc.NewRetrieve().
						Where(log.MatchKeys(key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				// Import mints a fresh key; every other field must round-trip intact.
				Expect(res.Key).ToNot(Equal(original.Key))
				original.Key = res.Key
				Expect(res).To(Equal(original))
			},
		)
	})
})
