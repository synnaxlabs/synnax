// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/types/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/types/v3"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// loadV55 reads a stored log body fixture and wraps it in the v0 snapshot shape that
// the gorp boot migration consumes. The fixture's promoted key and name become the
// envelope-level Key and Name; the whole object is also handed through as the raw Data
// blob, so any envelope-only keys (version, type) are simply ignored by the lenient v1
// parse the way an unrecognized persisted field would be.
func loadV55(path string) v2.Log {
	raw := MustSucceed(os.ReadFile(path))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	key := MustSucceed(uuid.Parse(m["key"].(string)))
	name, _ := m["name"].(string)
	return v2.Log{Key: key, Name: name, Data: m}
}

var _ = Describe("MigrateLog", func() {
	It("Should lift the v0 envelope fields", func(ctx SpecContext) {
		old := v2.Log{
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
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Name).To(Equal("my-log"))
		Expect(out.TimestampPrecision).To(Equal(int32(2)))
		Expect(out.HideChannelNames).To(BeTrue())
		Expect(out.HideReceiptTimestamp).To(BeFalse())
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Channel).To(BeEquivalentTo(42))
		Expect(out.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationScientific))
		Expect(out.Channels[0].Precision).To(Equal(int32(3)))
		Expect(out.Channels[0].Alias).To(Equal("temp"))
	})

	It("Should preserve per-channel timestamp config that the v1 schema discards", func(ctx SpecContext) {
		old := v2.Log{
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
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatISO))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
	})

	It("Should default the per-channel timestamp when the source omits it", func(ctx SpecContext) {
		old := v2.Log{
			Key:  uuid.New(),
			Name: "no-ts",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": 1},
				},
			},
		}
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatPreciseDate))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
	})

	It("Should drop UI-only fields the console persisted alongside the typed body", func(ctx SpecContext) {
		// The console used to send `setData = { ...state, key: undefined }`, which
		// included its toolbar state and persisted-state version. These must not appear
		// on the typed Log.
		old := v2.Log{
			Key:  uuid.New(),
			Name: "with-noise",
			Data: map[string]any{
				"version":  "1.0.0",
				"toolbar":  map[string]any{"activeTab": "channels"},
				"channels": []any{},
			},
		}
		Expect(v3.MigrateLog(ctx, old)).Error().ToNot(HaveOccurred())
	})

	It("Should be a no-op for an empty Data blob", func(ctx SpecContext) {
		old := v2.Log{Key: uuid.New(), Name: "empty"}
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Name).To(Equal("empty"))
		Expect(out.Channels).To(BeEmpty())
	})

	It("Should default an unknown notation value instead of failing", func(ctx SpecContext) {
		old := v2.Log{
			Key:  uuid.New(),
			Name: "bad-notation",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": 1, "notation": "unsupported"},
				},
			},
		}
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
	})

	It("Should default every typed enum in a channel that carries multiple bad fields", func(ctx SpecContext) {
		old := v2.Log{
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
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Channel).To(BeEquivalentTo(7))
		Expect(out.Channels[0].Color).To(Equal(color.Color{}))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatPreciseDate))
		Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
	})

	It("Should drop the body and keep Key+Name when the data blob is unparseable", func(ctx SpecContext) {
		// channel is required + must coerce to uint32; a string here can't recover via
		// lenient enum parsing, so the catch-all in MigrateLog drops the body.
		old := v2.Log{
			Key:  uuid.New(),
			Name: "unparseable",
			Data: map[string]any{
				"version": "1.0.0",
				"channels": []any{
					map[string]any{"channel": "not-a-number"},
				},
			},
		}
		out := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Name).To(Equal("unparseable"))
		Expect(out.Channels).To(BeEmpty())
	})

	Describe("from testdata fixtures", func() {
		It("Should fully migrate a well-formed v1 body", func(ctx SpecContext) {
			out := MustSucceed(v3.MigrateLog(
				ctx, loadV55("../testdata/import_v1.json"),
			))
			Expect(out.Name).To(Equal("Test Log V1"))
			Expect(out.Channels).To(HaveLen(2))
			Expect(out.Channels[0].Channel).To(BeEquivalentTo(1))
			Expect(out.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationScientific))
			Expect(out.Channels[0].Precision).To(Equal(int32(2)))
			Expect(out.Channels[0].Alias).To(Equal("temp"))
			Expect(out.Channels[1].Channel).To(BeEquivalentTo(5))
			Expect(out.Channels[1].Color).To(Equal(color.Color{}))
			Expect(out.TimestampPrecision).To(Equal(int32(1)))
			Expect(out.HideChannelNames).To(BeFalse())
			Expect(out.HideReceiptTimestamp).To(BeTrue())
		})

		It("Should default a malformed color hex to the zero color", func(ctx SpecContext) {
			out := MustSucceed(v3.MigrateLog(
				ctx, loadV55("../testdata/import_invalid_color.json"),
			))
			Expect(out.Name).To(Equal("Invalid Color"))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Channel).To(BeEquivalentTo(1))
			Expect(out.Channels[0].Color).To(Equal(color.Color{}))
		})

		It("Should lift a legacy v0 body forward through the chain", func(ctx SpecContext) {
			out := MustSucceed(v3.MigrateLog(
				ctx, loadV55("../testdata/import_v0.json"),
			))
			Expect(out.Name).To(Equal("Test Log V0"))
			Expect(out.Channels).To(HaveLen(3))
			Expect(out.Channels[0].Channel).To(BeEquivalentTo(1))
			Expect(out.Channels[1].Channel).To(BeEquivalentTo(2))
			Expect(out.Channels[2].Channel).To(BeEquivalentTo(3))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		})

		DescribeTable("Should keep Key and Name but yield no channels for an undecodable body",
			func(ctx SpecContext, path, name string) {
				old := loadV55(path)
				out := MustSucceed(v3.MigrateLog(ctx, old))
				Expect(out.Key).To(Equal(old.Key))
				Expect(out.Name).To(Equal(name))
				Expect(out.Channels).To(BeEmpty())
			},
			Entry("channels stored as a non-array", "../testdata/import_bad_data.json", "Bad Data"),
			Entry("unsupported version stamp", "../testdata/import_bad_version.json", "Bad Version"),
		)
	})

	Describe("storage integration", func() {
		It("Should re-encode and lift stored logs through the chain", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := v2.Log{
				Key:  uuid.New(),
				Name: "chain-log",
				Data: map[string]any{
					"version": "1.0.0",
					"channels": []any{
						map[string]any{"channel": 42, "color": "#ff0000"},
					},
					"showChannelNames":     false,
					"showReceiptTimestamp": true,
				},
			}
			MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Log]{DB: db}))
			Expect(gorp.NewCreate[v2.Key, v2.Log]().Entry(&seed).Exec(ctx, db)).To(Succeed())

			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Log",
				Migrations: v3.Migrations,
			})).To(Succeed())

			var got v3.Log
			Expect(gorp.NewRetrieve[v3.Key, v3.Log]().
				Where(gorp.MatchKeys[v3.Key, v3.Log](seed.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal("chain-log"))
			Expect(got.Channels).To(HaveLen(1))
			Expect(got.Channels[0].Channel).To(BeEquivalentTo(42))
			Expect(got.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(got.HideChannelNames).To(BeTrue())
			Expect(got.HideReceiptTimestamp).To(BeFalse())
		})
	})
})
