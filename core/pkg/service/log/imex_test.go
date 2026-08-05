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
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// loadEnvelope reads a wire-format envelope fixture from versions/testdata and
// unmarshals it into an imex.Envelope, binding the codec that Decode needs. The
// fixtures cover both legacy camelCase Console exports (v0, v1) and the current
// Core-typed snake_case shape (v2) that Import dispatches on by version.
func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	createLog := func(ctx SpecContext, l log.Log) log.Log {
		Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &l)).To(Succeed())
		return l
	}

	Describe("Export", func() {
		It(
			"Should export a created log as a versioned envelope",
			func(ctx SpecContext) {
				l := createLog(ctx, log.Log{
					Name: "exported",
					Channels: []log.ChannelEntry{
						{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000")},
					},
					TimestampPrecision: 2,
					HideChannelNames:   true,
				})
				env := MustSucceed(imexSvc.Export(ctx, l.OntologyID()))
				Expect(env.Version).To(Equal(log.Version))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("exported"))

				decoded := MustSucceed(imex.Decode[log.Log](ctx, WireRoundTrip(env)))
				Expect(decoded.Name).To(Equal("exported"))
				Expect(decoded.TimestampPrecision).To(Equal(int32(2)))
				Expect(decoded.HideChannelNames).To(BeTrue())
				Expect(decoded.Channels).To(HaveLen(1))
				Expect(decoded.Channels[0].Channel).To(Equal(channel.Key(1)))
			},
		)

		It(
			"Should return not found when the log does not exist",
			func(ctx SpecContext) {
				id := ontology.ID{Type: ontology.ResourceTypeLog, Key: uuid.NewString()}
				Expect(
					imexSvc.Export(ctx, id),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should return an error when the id key is not a valid UUID",
			func(ctx SpecContext) {
				id := ontology.ID{Type: ontology.ResourceTypeLog, Key: "not-a-uuid"}
				Expect(
					imexSvc.Export(ctx, id),
				).Error().
					To(MatchError(ContainSubstring("UUID")))
			},
		)
	})

	Describe("Import", func() {
		const (
			v0Fixture = "versions/testdata/import_v0.json"
			v1Fixture = "versions/testdata/import_v1.json"
			v2Fixture = "versions/testdata/import_v2.json"
		)

		importAndRetrieve := func(ctx SpecContext, path string) (ontology.ID, log.Log) {
			id := MustSucceed(
				imexSvc.Import(ctx, db, loadEnvelope(path), imex.ImportOptions{}),
			)
			Expect(id.Type).To(Equal(ontology.ResourceTypeLog))
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
				_, res := importAndRetrieve(ctx, v0Fixture)
				Expect(res.Name).To(Equal("Test Log V0"))
				Expect(res.Channels).To(HaveLen(3))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
				Expect(res.Channels[1].Channel).To(Equal(channel.Key(2)))
				Expect(res.Channels[2].Channel).To(Equal(channel.Key(3)))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationStandard))
			},
		)

		It(
			"Should import a legacy v1 envelope carrying camelCase keys",
			func(ctx SpecContext) {
				_, res := importAndRetrieve(ctx, v1Fixture)
				Expect(res.Name).To(Equal("Test Log V1"))
				Expect(res.TimestampPrecision).To(Equal(int32(1)))
				Expect(res.HideChannelNames).To(BeFalse())
				Expect(res.HideReceiptTimestamp).To(BeTrue())
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
				Expect(res.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
				Expect(res.Channels[0].Precision).To(Equal(int32(2)))
				Expect(res.Channels[0].Alias).To(Equal("temp"))
				Expect(res.Channels[1].Channel).To(Equal(channel.Key(5)))
			},
		)

		It(
			"Should import a v2 envelope carrying snake_case keys",
			func(ctx SpecContext) {
				_, res := importAndRetrieve(ctx, v2Fixture)
				Expect(res.Name).To(Equal("Test Log V2"))
				Expect(res.TimestampPrecision).To(Equal(int32(1)))
				Expect(res.HideChannelNames).To(BeFalse())
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
				Expect(res.Channels[0].Color).To(Equal(color.MustFromHex("#7f1d1d")))
				Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
				Expect(res.Channels[0].Precision).To(Equal(int32(2)))
				Expect(res.Channels[0].Alias).To(Equal("temp"))
				Expect(
					res.Channels[0].Timestamp.Format,
				).To(Equal(telem.TimestampFormatISO))
				Expect(res.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
				Expect(res.Channels[1].Channel).To(Equal(channel.Key(5)))
			},
		)

		It("Should register the imported log in the ontology", func(ctx SpecContext) {
			id, _ := importAndRetrieve(ctx, v2Fixture)
			var resource ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(id).Entry(&resource).Exec(ctx, db)).
				To(Succeed())
			Expect(resource.ID).To(Equal(id))
		})

		It(
			"Should create the imported log under the project from the options",
			func(ctx SpecContext) {
				id := MustSucceed(imexSvc.Import(
					ctx, db, loadEnvelope(v2Fixture),
					imex.ImportOptions{Project: proj.Key},
				))
				Expect(otg.RelationshipExists(ctx, nil, ontology.Relationship{
					From: proj.OntologyID(),
					Type: ontology.RelationshipTypeParentOf,
					To:   id,
				})).To(BeTrue())
			},
		)

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			Expect(imexSvc.Import(
				ctx, db, loadEnvelope(v2Fixture),
				imex.ImportOptions{Project: uuid.New()},
			)).Error().To(MatchError(query.ErrNotFound))
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
					ctx, db, env, imex.ImportOptions{FileName: "From File.json"},
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
						loadEnvelope(v2Fixture),
						imex.ImportOptions{},
					),
				)
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					loadEnvelope("versions/testdata/import_bad_version.json"),
					imex.ImportOptions{},
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
					loadEnvelope("versions/testdata/import_bad_v2.json"),
					imex.ImportOptions{},
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
						{Channel: channel.Key(1), Alias: "first", Precision: 4},
						{Channel: channel.Key(2), Alias: "second"},
					},
					TimestampPrecision:   1,
					HideReceiptTimestamp: true,
				})
				env := MustSucceed(imexSvc.Export(ctx, original.OntologyID()))
				id := MustSucceed(
					imexSvc.Import(ctx, db, WireRoundTrip(env), imex.ImportOptions{}),
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
