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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// wireRoundTrip marshals env to JSON and unmarshals it back, binding the codec that
// Decode needs. Export-side envelopes carry a body but no codec, so a decode of an
// exported envelope (and the import path generally) must first pass through the wire.
func wireRoundTrip(env imex.Envelope) imex.Envelope {
	b := MustSucceed(json.Marshal(env))
	var out imex.Envelope
	Expect(json.Unmarshal(b, &out)).To(Succeed())
	return out
}

// loadEnvelope reads a wire-format envelope fixture from migrations/testdata and
// unmarshals it into an imex.Envelope, binding the codec that Decode needs. The fixtures
// cover both legacy camelCase Console exports (v0, v1) and the current Core-typed
// snake_case shape (v2) that Import dispatches on by version.
func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	// reg is the registry the suite passed to log.OpenService, so svc registered itself
	// as the log importer/exporter on open — these specs exercise that wiring rather than
	// registering by hand. It is bound in BeforeEach because imexReg is not set until the
	// suite's BeforeSuite runs, after the spec tree is constructed.
	var reg *imex.Service
	BeforeEach(func() { reg = imexReg })

	createLog := func(ctx SpecContext, l log.Log) log.Log {
		Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &l)).To(Succeed())
		return l
	}

	Describe("Export", func() {
		It("Should export a created log as a versioned envelope", func(ctx SpecContext) {
			l := createLog(ctx, log.Log{
				Name: "exported",
				Channels: []log.ChannelEntry{
					{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000")},
				},
				TimestampPrecision: 2,
				ShowChannelNames:   true,
			})
			env := MustSucceed(reg.Export(ctx, log.OntologyID(l.Key)))
			Expect(env.Version).To(Equal(log.ImExVersion))
			Expect(env.Type).To(Equal("log"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[log.Log](ctx, wireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.TimestampPrecision).To(Equal(int32(2)))
			Expect(decoded.ShowChannelNames).To(BeTrue())
			Expect(decoded.Channels).To(HaveLen(1))
			Expect(decoded.Channels[0].Channel).To(Equal(channel.Key(1)))
		})

		It("Should return not found when the log does not exist", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeLog, Key: uuid.NewString()}
			Expect(reg.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Import", func() {
		const (
			v0Fixture = "migrations/testdata/import_v0.json"
			v1Fixture = "migrations/testdata/import_v1.json"
			v2Fixture = "migrations/testdata/import_v2.json"
		)

		importAndRetrieve := func(ctx SpecContext, path string) (ontology.ID, log.Log) {
			id := MustSucceed(reg.Import(ctx, db, loadEnvelope(path)))
			Expect(id.Type).To(Equal(ontology.ResourceTypeLog))
			key := MustSucceed(uuid.Parse(id.Key))
			var res log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(key)).Entry(&res).Exec(ctx, db)).
				To(Succeed())
			return id, res
		}

		It("Should import a legacy v0 envelope, lifting bare channel keys forward", func(ctx SpecContext) {
			_, res := importAndRetrieve(ctx, v0Fixture)
			Expect(res.Name).To(Equal("Test Log V0"))
			Expect(res.Channels).To(HaveLen(3))
			Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(res.Channels[1].Channel).To(Equal(channel.Key(2)))
			Expect(res.Channels[2].Channel).To(Equal(channel.Key(3)))
			Expect(res.Channels[0].Notation).To(Equal(notation.NotationStandard))
		})

		It("Should import a legacy v1 envelope carrying camelCase keys", func(ctx SpecContext) {
			_, res := importAndRetrieve(ctx, v1Fixture)
			Expect(res.Name).To(Equal("Test Log V1"))
			Expect(res.TimestampPrecision).To(Equal(int32(1)))
			Expect(res.ShowChannelNames).To(BeTrue())
			Expect(res.ShowReceiptTimestamp).To(BeFalse())
			Expect(res.Channels).To(HaveLen(2))
			Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(res.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
			Expect(res.Channels[0].Precision).To(Equal(int32(2)))
			Expect(res.Channels[0].Alias).To(Equal("temp"))
			Expect(res.Channels[1].Channel).To(Equal(channel.Key(5)))
		})

		It("Should import a v2 envelope carrying snake_case keys", func(ctx SpecContext) {
			_, res := importAndRetrieve(ctx, v2Fixture)
			Expect(res.Name).To(Equal("Test Log V2"))
			Expect(res.TimestampPrecision).To(Equal(int32(1)))
			Expect(res.ShowChannelNames).To(BeTrue())
			Expect(res.Channels).To(HaveLen(2))
			Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(res.Channels[0].Color).To(Equal(color.MustFromHex("#7f1d1d")))
			Expect(res.Channels[0].Notation).To(Equal(notation.NotationScientific))
			Expect(res.Channels[0].Precision).To(Equal(int32(2)))
			Expect(res.Channels[0].Alias).To(Equal("temp"))
			Expect(res.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatISO))
			Expect(res.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
			Expect(res.Channels[1].Channel).To(Equal(channel.Key(5)))
		})

		It("Should register the imported log in the ontology", func(ctx SpecContext) {
			id, _ := importAndRetrieve(ctx, v2Fixture)
			var resource ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(id).Entry(&resource).Exec(ctx, db)).
				To(Succeed())
			Expect(resource.ID).To(Equal(id))
		})

		It("Should generate a fresh key, discarding the key on the wire", func(ctx SpecContext) {
			id := MustSucceed(reg.Import(ctx, db, loadEnvelope(v2Fixture)))
			Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
		})

		It("Should reject an envelope newer than the supported version", func(ctx SpecContext) {
			Expect(reg.Import(ctx, db,
				loadEnvelope("migrations/testdata/import_bad_version.json"),
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("log version 99")),
				MatchError(ContainSubstring("newer than this Core supports")),
			))
		})
	})

	Describe("Round trip", func() {
		It("Should preserve log content through export then import", func(ctx SpecContext) {
			original := createLog(ctx, log.Log{
				Name: "round-trip",
				Channels: []log.ChannelEntry{
					{Channel: channel.Key(1), Alias: "first", Precision: 4},
					{Channel: channel.Key(2), Alias: "second"},
				},
				TimestampPrecision:   1,
				ShowReceiptTimestamp: true,
			})
			env := MustSucceed(reg.Export(ctx, log.OntologyID(original.Key)))
			id := MustSucceed(reg.Import(ctx, db, wireRoundTrip(env)))

			key := MustSucceed(uuid.Parse(id.Key))
			var res log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(key)).Entry(&res).Exec(ctx, db)).
				To(Succeed())
			Expect(res.Name).To(Equal("round-trip"))
			Expect(res.TimestampPrecision).To(Equal(int32(1)))
			Expect(res.ShowReceiptTimestamp).To(BeTrue())
			Expect(res.Channels).To(HaveLen(2))
			Expect(res.Channels[0].Alias).To(Equal("first"))
			Expect(res.Channels[0].Precision).To(Equal(int32(4)))
			Expect(res.Channels[1].Channel).To(Equal(channel.Key(2)))
		})
	})
})
