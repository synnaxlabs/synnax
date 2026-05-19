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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImportExport", func() {
	Describe("Import", func() {
		It("Should import a v1 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(uuid.Validate(key)).To(Succeed())
		})

		It("Should import and migrate a v0 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(uuid.Validate(key)).To(Succeed())
		})

		It("Should reject a future version", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			env.Version = 999
			Expect(svc.Import(ctx, tx, env)).Error().To(
				MatchError(ContainSubstring("newer than this Core supports")),
			)
		})

		It("Should reject invalid data with a user-friendly error", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_bad_data.json")
			Expect(svc.Import(ctx, tx, env)).Error().To(
				MatchError(ContainSubstring("channels")),
			)
		})

		It("Should reject a channel with a malformed color hex", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_invalid_color.json")
			Expect(svc.Import(ctx, tx, env)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("invalid hex color")),
				MatchError(ContainSubstring("not-a-hex")),
			))
		})
	})

	Describe("Export", func() {
		It("Should export an existing log", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			Expect(exported.Type).To(Equal("log"))
			Expect(exported.Name).To(Equal(env.Name))
			Expect(exported.Version).To(Equal(v1.Version))
		})

		It("Should reject a non-UUID key", func(ctx SpecContext) {
			Expect(svc.Export(ctx, "not-a-uuid")).Error().To(
				MatchError(ContainSubstring("invalid UUID")),
			)
		})

		It("Should error when the log does not exist", func(ctx SpecContext) {
			Expect(svc.Export(ctx, uuid.New().String())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
	})

	Describe("Round-trip", func() {
		It("Should import v0 and export as v1", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			Expect(exported.Data).To(HaveKey("channels"))
			Expect(exported.Data["channels"]).To(HaveLen(3))
		})
	})

	Describe("(Log).Data", func() {
		It("Should project every persisted body field with json-tag keys", func() {
			l := log.Log{
				Key:                  uuid.New(),
				Name:                 "ignored",
				Channels:             []log.ChannelEntry{{Channel: channel.Key(7)}},
				RemoteCreated:        true,
				TimestampPrecision:   2,
				ShowChannelNames:     false,
				ShowReceiptTimestamp: true,
			}
			data := l.Data()
			Expect(data).To(HaveKeyWithValue("remote_created", true))
			Expect(data).To(HaveKeyWithValue("timestamp_precision", int32(2)))
			Expect(data).To(HaveKeyWithValue("show_channel_names", false))
			Expect(data).To(HaveKeyWithValue("show_receipt_timestamp", true))
			Expect(data["channels"]).To(Equal(l.Channels))
		})

		It("Should not project the envelope-level Key or Name", func() {
			l := log.Log{Key: uuid.New(), Name: "should-not-appear"}
			Expect(l.Data()).ToNot(HaveKey("key"))
			Expect(l.Data()).ToNot(HaveKey("name"))
		})

		It("Should marshal cleanly to JSON via the channel struct tags", func() {
			l := log.Log{
				Channels: []log.ChannelEntry{{
					Channel:   channel.Key(42),
					Color:     color.MustFromHex("#ff0000"),
					Notation:  notation.NotationScientific,
					Precision: 3,
					Alias:     "temp",
					Timestamp: log.TimestampConfig{
						Format: telem.TimestampFormatISO,
						Tz:     telem.TimeZoneUTC,
					},
				}},
			}
			raw := MustSucceed(json.Marshal(l.Data()))
			Expect(string(raw)).To(SatisfyAll(
				ContainSubstring(`"channel":42`),
				ContainSubstring(`"notation":"scientific"`),
				ContainSubstring(`"alias":"temp"`),
				ContainSubstring(`"format":"ISO"`),
				ContainSubstring(`"tz":"UTC"`),
			))
		})

	})
})
