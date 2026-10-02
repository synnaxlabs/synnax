// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json/v2"
	"os"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// loadV55 reads a stored log body fixture and wraps it in the v0 snapshot shape that
// the gorp boot migration consumes. The fixture's promoted key and name become the
// envelope-level Key and Name; the whole object is also handed through as the raw Data
// blob, so any envelope-only keys (version, type) are simply ignored by the lenient v1
// parse the way an unrecognized persisted field would be.
func loadV55(path string) v0.Log {
	GinkgoHelper()
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
		out := MustSucceed(v2.MigrateLog(ctx, old))
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Name).To(Equal("my-log"))
		Expect(out.TimestampPrecision).To(Equal(int32(2)))
		Expect(out.ChannelNamesHidden).To(BeTrue())
		Expect(out.ReceiptTimestampHidden).To(BeFalse())
		Expect(out.Channels).To(HaveLen(1))
		Expect(out.Channels[0].Channel).To(BeEquivalentTo(42))
		Expect(out.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationScientific))
		Expect(out.Channels[0].Precision).To(Equal(int32(3)))
		Expect(out.Channels[0].Alias).To(Equal("temp"))
	})

	It(
		"Should preserve per-channel timestamp config that the v1 schema discards",
		func(ctx SpecContext) {
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
			out := MustSucceed(v2.MigrateLog(ctx, old))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Timestamp.Format).To(Equal(telem.TimestampFormatISO))
			Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneUTC))
		},
	)

	It(
		"Should default the per-channel timestamp when the source omits it",
		func(ctx SpecContext) {
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
			out := MustSucceed(v2.MigrateLog(ctx, old))
			Expect(out.Channels).To(HaveLen(1))
			Expect(
				out.Channels[0].Timestamp.Format,
			).To(Equal(telem.TimestampFormatPreciseDate))
			Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
		},
	)

	It(
		"Should drop UI-only fields the console persisted alongside the typed body",
		func(ctx SpecContext) {
			// The Console used to send `setData = { ...state, key: undefined }`, which
			// included its toolbar state and persisted-state version. These must not
			// appear on the typed Log.
			old := v0.Log{
				Key:  uuid.New(),
				Name: "with-noise",
				Data: map[string]any{
					"version":  "1.0.0",
					"toolbar":  map[string]any{"activeTab": "channels"},
					"channels": []any{},
				},
			}
			Expect(v2.MigrateLog(ctx, old)).Error().ToNot(HaveOccurred())
		},
	)

	It("Should be a no-op for an empty Data blob", func(ctx SpecContext) {
		old := v0.Log{Key: uuid.New(), Name: "empty"}
		out := MustSucceed(v2.MigrateLog(ctx, old))
		Expect(out.Name).To(Equal("empty"))
		Expect(out.Channels).To(BeEmpty())
	})

	It(
		"Should default an unknown notation value instead of failing",
		func(ctx SpecContext) {
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
			out := MustSucceed(v2.MigrateLog(ctx, old))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		},
	)

	It(
		"Should default every typed enum in a channel that carries multiple bad fields",
		func(ctx SpecContext) {
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
			out := MustSucceed(v2.MigrateLog(ctx, old))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Channel).To(BeEquivalentTo(7))
			Expect(out.Channels[0].Color).To(Equal(color.Color{}))
			Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
			Expect(
				out.Channels[0].Timestamp.Format,
			).To(Equal(telem.TimestampFormatPreciseDate))
			Expect(out.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
		},
	)

	It("Should reject a data blob it cannot decode", func(ctx SpecContext) {
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
		Expect(v2.MigrateLog(ctx, old)).Error().To(MatchError(ContainSubstring(
			"cannot unmarshal JSON string into Go channel.Key",
		)))
	})

	It(
		"Should round down a fractional precision typed in an older Console",
		func(ctx SpecContext) {
			old := v0.Log{
				Key:  uuid.New(),
				Name: "fractional",
				Data: map[string]any{
					"version":            "1.0.0",
					"timestampPrecision": 2.5,
					"showChannelNames":   false,
					"channels": []any{
						map[string]any{
							"channel":   4,
							"alias":     "pres",
							"precision": 2.5,
						},
					},
				},
			}
			out := MustSucceed(v2.MigrateLog(ctx, old))
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Alias).To(Equal("pres"))
			Expect(out.Channels[0].Precision).To(BeEquivalentTo(2))
			Expect(out.TimestampPrecision).To(BeEquivalentTo(2))
			Expect(out.ChannelNamesHidden).To(BeTrue())
		},
	)

	Describe("from testdata fixtures", func() {
		It("Should fully migrate a well-formed v1 body", func(ctx SpecContext) {
			out := MustSucceed(v2.MigrateLog(
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
			Expect(out.ChannelNamesHidden).To(BeFalse())
			Expect(out.ReceiptTimestampHidden).To(BeTrue())
		})

		It(
			"Should default a malformed color hex to the zero color",
			func(ctx SpecContext) {
				out := MustSucceed(v2.MigrateLog(
					ctx, loadV55("../testdata/import_invalid_color.json"),
				))
				Expect(out.Name).To(Equal("Invalid Color"))
				Expect(out.Channels).To(HaveLen(1))
				Expect(out.Channels[0].Channel).To(BeEquivalentTo(1))
				Expect(out.Channels[0].Color).To(Equal(color.Color{}))
			},
		)

		It(
			"Should lift a Console v0 body forward through the chain",
			func(ctx SpecContext) {
				out := MustSucceed(v2.MigrateLog(
					ctx, loadV55("../testdata/import_v0.json"),
				))
				Expect(out.Name).To(Equal("Test Log V0"))
				Expect(out.Channels).To(HaveLen(3))
				Expect(out.Channels[0].Channel).To(BeEquivalentTo(1))
				Expect(out.Channels[1].Channel).To(BeEquivalentTo(2))
				Expect(out.Channels[2].Channel).To(BeEquivalentTo(3))
				Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
			},
		)

		DescribeTable(
			"Should reject an undecodable body",
			func(ctx SpecContext, path, msg string) {
				Expect(v2.MigrateLog(ctx, loadV55(path))).Error().
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"channels stored as a non-array",
				"../testdata/import_bad_data.json",
				"cannot unmarshal JSON string into Go []v1.ChannelEntry",
			),
			Entry(
				"unsupported version stamp",
				"../testdata/import_bad_version.json",
				"unknown log data version",
			),
		)
	})

	Describe("storage integration", func() {
		It(
			"Should re-encode and lift stored logs through the chain",
			func(ctx SpecContext) {
				db := DeferClose(gorp.Wrap(memkv.New()))
				seed := v0.Log{
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
				MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Log]{DB: db}),
				)
				Expect(
					gorp.NewCreate[v0.Key, v0.Log]().Entry(&seed).Exec(ctx, db),
				).To(Succeed())

				Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
					DB:         db,
					Namespace:  "Log",
					Migrations: []migrate.Migration{v0.Migration, v2.Migration},
				})).To(Succeed())

				var got v2.Log
				Expect(gorp.NewRetrieve[v2.Key, v2.Log]().
					Where(gorp.MatchKeys[v2.Key, v2.Log](seed.Key)).
					Entry(&got).Exec(ctx, db)).To(Succeed())
				Expect(got.Key).To(Equal(seed.Key))
				Expect(got.Name).To(Equal("chain-log"))
				Expect(got.Channels).To(HaveLen(1))
				Expect(got.Channels[0].Channel).To(BeEquivalentTo(42))
				Expect(got.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
				Expect(got.ChannelNamesHidden).To(BeTrue())
				Expect(got.ReceiptTimestampHidden).To(BeFalse())
			},
		)
	})
})

var _ = Describe("Migration", func() {
	It("Should keep a stored log with an undecodable body and log it", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := v0.Log{
			Key:  uuid.New(),
			Name: "unparseable",
			Data: map[string]any{
				"version":  "1.0.0",
				"channels": []any{map[string]any{"channel": "not-a-number"}},
			},
		}
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Log]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Log]().Entry(&seed).Exec(ctx, db)).
			To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "Log",
			Migrations:      []migrate.Migration{v0.Migration, v2.Migration},
		})).To(Succeed())
		var got v2.Log
		Expect(gorp.NewRetrieve[v2.Key, v2.Log]().
			Where(gorp.MatchKeys[v2.Key, v2.Log](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("unparseable"))
		Expect(got.Channels).To(BeEmpty())
		dropped := logs.FilterMessage("dropped a log body that does not decode").All()
		Expect(dropped).To(HaveLen(1))
		Expect(dropped[0].ContextMap()).To(HaveKeyWithValue("log", seed.Key.String()))
	})
})
