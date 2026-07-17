// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/types/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// loadV55 reads a stored log body fixture and wraps it in the v0 snapshot shape that
// the gorp boot migration consumes. The fixture's promoted key and name become the
// envelope-level Key and Name; the whole object is also handed through as the raw Data
// blob, so any envelope-only keys (version, type) are simply ignored by the lenient v1
// parse the way an unrecognized persisted field would be.
func loadV55(path string) v0.Log {
	raw := MustSucceed(os.ReadFile(path))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	key := MustSucceed(uuid.Parse(m["key"].(string)))
	name, _ := m["name"].(string)
	return v0.Log{Key: key, Name: name, Data: m}
}

var _ = Describe("MigrateLog", func() {
	It("Should lift the v0 envelope fields", func(ctx SpecContext) {
		old := v0.Log{
			Key:  uuid.New(),
			Name: "my-log",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{
						"channel":   42,
						"color":     "#ff0000",
						"notation":  "scientific",
						"precision": 3,
						"alias":     "temp",
					},
				},
				"remoteCreated":        true,
				"timestampPrecision":   2,
				"showChannelNames":     false,
				"showReceiptTimestamp": true,
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Name).To(Equal("my-log"))
		Expect(out.TimestampPrecision).To(Equal(int32(2)))
		Expect(out.HideChannelNames).To(BeTrue())
		Expect(out.HideReceiptTimestamp).To(BeFalse())
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Channel).To(Equal(channel.Key(42)))
		Expect(out.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationScientific))
		Expect(out.Channels[0].Precision).To(Equal(int32(3)))
		Expect(out.Channels[0].Alias).To(Equal("temp"))
	})

	It("Should preserve per-channel timestamp config that the v1 schema discards", func(ctx SpecContext) {
		old := v0.Log{
			Key:  uuid.New(),
			Name: "with-ts",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{
						"channel": 1,
						"timestamp": map[string]any{
							"format": "ISO",
							"tz":     "UTC",
						},
					},
				},
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatISO))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
	})

	It("Should default the per-channel timestamp when the source omits it", func(ctx SpecContext) {
		old := v0.Log{
			Key:  uuid.New(),
			Name: "no-ts",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": 1},
				},
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatPreciseDate))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
	})

	It("Should drop UI-only fields the console persisted alongside the typed body", func(ctx SpecContext) {
		// The console used to send `setData = { ...state, key: undefined }`, which
		// included its toolbar state and persisted-state version. These must not appear
		// on the typed Log.
		old := v0.Log{
			Key:  uuid.New(),
			Name: "with-noise",
			Data: map[string]any{
				"version":  "1.0.0",
				"toolbar":  map[string]any{"activeTab": "channels"},
				"channels": []any{},
			},
		}
		Expect(v1.MigrateLog(ctx, old)).Error().ToNot(HaveOccurred())
	})

	It("Should be a no-op for an empty Data blob", func(ctx SpecContext) {
		old := v0.Log{Key: uuid.New(), Name: "empty"}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Name).To(Equal("empty"))
		Expect(out.Channels).To(BeEmpty())
	})

	It("Should default an unknown notation value instead of failing", func(ctx SpecContext) {
		old := v0.Log{
			Key:  uuid.New(),
			Name: "bad-notation",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": 1, "notation": "unsupported"},
				},
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
	})

	It("Should default every typed enum in a channel that carries multiple bad fields", func(ctx SpecContext) {
		old := v0.Log{
			Key:  uuid.New(),
			Name: "fully-bad",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{
						"channel":  7,
						"color":    "not-a-hex",
						"notation": "unsupported",
						"timestamp": map[string]any{
							"format": "calendar",
							"tz":     "MST",
						},
					},
				},
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Channel).To(Equal(channel.Key(7)))
		Expect(out.Channels[0].Color).To(Equal(color.Color{}))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatPreciseDate))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
	})

	It("Should drop the body and keep Key+Name when the data blob is unparseable", func(ctx SpecContext) {
		// channel is required + must coerce to uint32; a string here can't recover via
		// lenient enum parsing, so the catch-all in MigrateLog drops the body.
		old := v0.Log{
			Key:  uuid.New(),
			Name: "unparseable",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": "not-a-number"},
				},
			},
		}
		out := MustSucceed(v1.MigrateLog(ctx, old))
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Name).To(Equal("unparseable"))
		Expect(out.Channels).To(BeEmpty())
	})

	Describe("from testdata fixtures", func() {
		It("Should fully migrate a well-formed v1 body", func(ctx SpecContext) {
			out := MustSucceed(v1.MigrateLog(
				ctx, loadV55("../testdata/import_v1.json"),
			))
			Expect(out.Name).To(Equal("Test Log V1"))
			Expect(out.Channels).To(HaveLen(2))
			Expect(out.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(out.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationScientific))
			Expect(out.Channels[0].Precision).To(Equal(int32(2)))
			Expect(out.Channels[0].Alias).To(Equal("temp"))
			Expect(out.Channels[1].Channel).To(Equal(channel.Key(5)))
			Expect(out.Channels[1].Color).To(Equal(color.Color{}))
			Expect(out.TimestampPrecision).To(Equal(int32(1)))
			Expect(out.HideChannelNames).To(BeFalse())
			Expect(out.HideReceiptTimestamp).To(BeTrue())
		})

		It("Should default a malformed color hex to the zero color", func(ctx SpecContext) {
			out := MustSucceed(v1.MigrateLog(
				ctx, loadV55("../testdata/import_invalid_color.json"),
			))
			Expect(out.Name).To(Equal("Invalid Color"))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(out.Channels[0].Color).To(Equal(color.Color{}))
		})

		It("Should lift a legacy v0 body forward through the chain", func(ctx SpecContext) {
			out := MustSucceed(v1.MigrateLog(
				ctx, loadV55("../testdata/import_v0.json"),
			))
			Expect(out.Name).To(Equal("Test Log V0"))
			Expect(out.Channels).To(HaveLen(3))
			Expect(out.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(out.Channels[1].Channel).To(Equal(channel.Key(2)))
			Expect(out.Channels[2].Channel).To(Equal(channel.Key(3)))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		})

		DescribeTable("Should keep Key and Name but yield no channels for an undecodable body",
			func(ctx SpecContext, path, name string) {
				old := loadV55(path)
				out := MustSucceed(v1.MigrateLog(ctx, old))
				Expect(out.Key).To(Equal(old.Key))
				Expect(out.Name).To(Equal(name))
				Expect(out.Channels).To(BeEmpty())
			},
			Entry("channels stored as a non-array", "../testdata/import_bad_data.json", "Bad Data"),
			Entry("unsupported version stamp", "../testdata/import_bad_version.json", "Bad Version"),
		)
	})
})
