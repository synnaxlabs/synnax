// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config_test

import (
	"context"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	arcversions "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions"
	arclegacy "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// testType is the task type the suite registers its store under. The record type
// is the Arc task config, picked because it has both schema defaults and validation.
const testType = "arc_task"

var _ = Describe("Service", func() {
	var svc *config.Service[arctask.Config]
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:            db,
				Type:          testType,
				Ladder:        arcversions.ConfigLadder,
				SetEntryKey:   (*arctask.Config).SetKey,
				ValidateEntry: (*arctask.Config).Validate,
			},
		))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					Type:        testType,
					SetEntryKey: (*arctask.Config).SetKey,
				},
			)).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the type", func(ctx SpecContext) {
			Expect(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:          db,
					SetEntryKey: (*arctask.Config).SetKey,
				},
			)).Error().To(MatchError(ContainSubstring("type: required")))
		})

		It(
			"Should reject a config missing the set entry key hook",
			func(ctx SpecContext) {
				Expect(config.OpenService(
					ctx,
					config.ServiceConfig[arctask.Config]{
						DB:     db,
						Type:   testType,
						Ladder: arcversions.ConfigLadder,
					},
				)).Error().To(MatchError(ContainSubstring("set_entry_key: must be non-nil")))
			},
		)

		It("Should reject a config missing the ladder", func(ctx SpecContext) {
			Expect(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:          db,
					Type:        testType,
					SetEntryKey: (*arctask.Config).SetKey,
				},
			)).Error().To(MatchError(ContainSubstring(
				"ladder.decode: must be non-nil",
			)))
		})
	})

	Describe("Type", func() {
		It("Should report the configured task type", func() {
			Expect(svc.Type()).To(Equal(testType))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded config under the given key", func(
			ctx SpecContext,
		) {
			key, arcKey := uuid.New(), uuid.New()
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     arcKey.String(),
				"performance": "high",
			}))
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["performance"]).To(Equal("high"))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := uuid.New()
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["performance"]).To(Equal("auto"))
		})

		It("Should keep an explicit empty field instead of defaulting it", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "",
			})).Error().To(MatchError(ContainSubstring("invalid performance")))
		})

		It("Should return the stored record as Read returns it", func(ctx SpecContext) {
			key := uuid.New()
			stored := MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(stored["performance"]).To(Equal("auto"))
			Expect(stored).To(Equal(MustSucceed(svc.Read(ctx, nil, key))))
		})

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "high",
			}))
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "low",
			}))
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["performance"]).To(Equal("low"))
		})

		It("Should return a validation error when the data does not decode", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key": 123,
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "BOGUS",
			})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
		})
	})

	Describe("WriteStrict", func() {
		It("Should store a config whose members the record declares", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.WriteStrict(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "high",
			}))
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["performance"]).To(Equal("high"))
		})

		It("Should return the stored record as Read returns it", func(ctx SpecContext) {
			key := uuid.New()
			stored := MustSucceed(svc.WriteStrict(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(stored["performance"]).To(Equal("auto"))
			Expect(stored).To(Equal(MustSucceed(svc.Read(ctx, nil, key))))
		})

		It("Should reject a member the record does not declare", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.WriteStrict(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performnace": "high",
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring(`unknown object member name "performnace"`),
				),
			))
			Expect(svc.Read(ctx, nil, key)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.WriteStrict(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "BOGUS",
			})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
		})
	})

	Describe("Write", func() {
		It("Should drop a member the record does not declare", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
				"extra":   true,
			}))
			Expect(MustSucceed(svc.Read(ctx, nil, key))).ToNot(HaveKey("extra"))
		})

		Context("With a resolve hook", func() {
			openResolving := func(
				ctx SpecContext,
				resolve func(context.Context, gorp.Tx, *arctask.Config) error,
			) *config.Service[arctask.Config] {
				GinkgoHelper()
				return MustOpen(config.OpenService(
					ctx,
					config.ServiceConfig[arctask.Config]{
						DB:            db,
						Type:          testType,
						Ladder:        arcversions.ConfigLadder,
						SetEntryKey:   (*arctask.Config).SetKey,
						ResolveEntry:  resolve,
						ValidateEntry: (*arctask.Config).Validate,
					},
				))
			}

			It("Should store the entry the hook resolves, before validating it", func(
				ctx SpecContext,
			) {
				resolving := openResolving(ctx, func(
					_ context.Context,
					tx gorp.Tx,
					c *arctask.Config,
				) error {
					Expect(tx).ToNot(BeNil())
					c.Performance = "low"
					return nil
				})
				key := uuid.New()
				MustSucceed(resolving.Write(ctx, nil, key, msgpack.EncodedJSON{
					"arc_key":     uuid.New().String(),
					"performance": "BOGUS",
				}))
				data := MustSucceed(resolving.Read(ctx, nil, key))
				Expect(data["performance"]).To(Equal("low"))
			})

			It("Should return the hook's error and store nothing", func(
				ctx SpecContext,
			) {
				resolving := openResolving(ctx, func(
					context.Context,
					gorp.Tx,
					*arctask.Config,
				) error {
					return errors.New("resolve failed")
				})
				key := uuid.New()
				Expect(resolving.Write(ctx, nil, key, msgpack.EncodedJSON{
					"arc_key": uuid.New().String(),
				})).Error().To(MatchError("resolve failed"))
				Expect(resolving.Read(ctx, nil, key)).Error().
					To(MatchError(query.ErrNotFound))
			})
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Read(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := uuid.New()
			MustSucceed(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
		})

		It("Should remove multiple records in one call", func(ctx SpecContext) {
			k1, k2 := uuid.New(), uuid.New()
			for _, k := range []uuid.UUID{k1, k2} {
				MustSucceed(svc.Write(ctx, nil, k, msgpack.EncodedJSON{
					"arc_key": uuid.New().String(),
				}))
			}
			Expect(svc.Delete(ctx, nil, k1, k2)).To(Succeed())
			Expect(svc.Read(ctx, nil, k1)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Read(ctx, nil, k2)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			from, to := uuid.New(), uuid.New()
			arcKey := uuid.New()
			MustSucceed(svc.Write(ctx, nil, from, msgpack.EncodedJSON{
				"arc_key": arcKey.String(),
			}))
			Expect(svc.Copy(ctx, nil, from, to)).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, to))
			Expect(data["key"]).To(Equal(to.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			original := MustSucceed(svc.Read(ctx, nil, from))
			Expect(original["key"]).To(Equal(from.String()))
		})

		It("Should return not found when the source is missing", func(
			ctx SpecContext,
		) {
			Expect(svc.Copy(ctx, nil, uuid.New(), uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Normalize", func() {
		var versioned *config.Service[arctask.Config]
		BeforeEach(func(ctx SpecContext) {
			versioned = MustOpen(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:            db,
					Type:          "versioned_test",
					Ladder:        arcversions.ConfigLadder,
					LegacyVersion: 1,
					SetEntryKey:   (*arctask.Config).SetKey,
					Legacy: &legacy.Rewrite{
						Post: func(cfg msgpack.EncodedJSON) {
							legacy.RenameKey(cfg, "module", "arc_key")
						},
					},
				},
			))
		})

		It("Should reject a version above the ladder's latest", func(ctx SpecContext) {
			Expect(versioned.Normalize(
				ctx, arcversions.Latest+1, msgpack.EncodedJSON{},
			)).Error().To(MatchError(ContainSubstring(fmt.Sprintf(
				"version: versioned_test version %d is newer than this Core supports "+
					"(latest: %d): validation error",
				arcversions.Latest+1,
				arcversions.Latest,
			))))
		})

		It("Should decode current-version data into the current shape", func(
			ctx SpecContext,
		) {
			arcKey := uuid.New()
			out := MustSucceed(versioned.Normalize(
				ctx,
				arcversions.Latest,
				msgpack.EncodedJSON{"arc_key": arcKey.String(), "unknown": 1},
			))
			Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
			Expect(out).ToNot(HaveKey("unknown"))
		})

		It(
			"Should run the legacy rewrite on a legacy version, then decode at the "+
				"ladder's first version",
			func(ctx SpecContext) {
				arcKey := uuid.New()
				out := MustSucceed(versioned.Normalize(ctx, 0, msgpack.EncodedJSON{
					"module":     arcKey.String(),
					"dataSaving": true,
				}))
				Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
				Expect(out).To(HaveKeyWithValue("data_saving_disabled", false))
				Expect(out).ToNot(HaveKey("module"))
			},
		)

		It("Should apply era normalization alone when no legacy rewrite is set", func(
			ctx SpecContext,
		) {
			eraOnly := MustOpen(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:          db,
					Type:        "era_only_test",
					Ladder:      arcversions.ConfigLadder,
					SetEntryKey: (*arctask.Config).SetKey,
				},
			))
			arcKey := uuid.New()
			out := MustSucceed(eraOnly.Normalize(ctx, 0, msgpack.EncodedJSON{
				"arcKey":     arcKey.String(),
				"dataSaving": false,
			}))
			Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
			Expect(out).To(HaveKeyWithValue("data_saving_disabled", true))
		})

		It("Should apply the store's defaults to the normalized config", func(
			ctx SpecContext,
		) {
			out := MustSucceed(svc.Normalize(
				ctx,
				arcversions.Latest,
				msgpack.EncodedJSON{"arc_key": uuid.New().String()},
			))
			Expect(out).To(HaveKeyWithValue("performance", "auto"))
		})

		DescribeTable("Should default only the members a lifted config omits",
			func(ctx SpecContext, in msgpack.EncodedJSON, performance string) {
				// The lift builds the entry in Go, as a ladder migration step does.
				lifted := MustOpen(config.OpenService(
					ctx,
					config.ServiceConfig[arctask.Config]{
						DB:          db,
						Type:        "lifted_test",
						SetEntryKey: (*arctask.Config).SetKey,
						Ladder: imex.Ladder[arctask.Config]{
							First:  1,
							Latest: 2,
							Decode: func(
								_ context.Context,
								_ imex.Version,
								data msgpack.EncodedJSON,
							) (arctask.Config, error) {
								p, _ := data["performance"].(string)
								return arctask.Config{
									ArcKey: uuid.MustParse(
										data["arc_key"].(string),
									),
									Performance: arctask.Performance(p),
								}, nil
							},
						},
					},
				))
				out := MustSucceed(lifted.Normalize(ctx, 1, in))
				Expect(out).To(HaveKeyWithValue("performance", performance))
			},
			Entry(
				"an omitted member takes its default",
				msgpack.EncodedJSON{"arc_key": uuid.New().String()},
				"auto",
			),
			Entry(
				"an explicit zero stays",
				msgpack.EncodedJSON{"arc_key": uuid.New().String(), "performance": ""},
				"",
			),
		)

		DescribeTable("Should default the omitted members of a lifted nested record",
			func(ctx SpecContext, in msgpack.EncodedJSON, valuePointer, format string) {
				lifted := MustOpen(config.OpenService(
					ctx,
					config.ServiceConfig[kafka.WriteConfig]{
						DB:          db,
						Type:        "lifted_record_test",
						SetEntryKey: (*kafka.WriteConfig).SetKey,
						Ladder: imex.Ladder[kafka.WriteConfig]{
							First:  1,
							Latest: 2,
							Decode: func(
								_ context.Context,
								_ imex.Version,
								data msgpack.EncodedJSON,
							) (kafka.WriteConfig, error) {
								record, _ := data["record"].(map[string]any)
								pointer, _ := record["value_pointer"].(string)
								timeFormat, _ := record["time_format"].(string)
								return kafka.WriteConfig{
									Topic: "t",
									Record: kafka.Record{
										ValuePointer: pointer,
										TimeFormat:   http.TimeFormat(timeFormat),
									},
								}, nil
							},
						},
					},
				))
				out := MustSucceed(lifted.Normalize(ctx, 1, in))
				Expect(out["record"]).To(SatisfyAll(
					HaveKeyWithValue("value_pointer", valuePointer),
					HaveKeyWithValue("time_format", format),
				))
			},
			Entry(
				"an omitted record takes its defaults",
				msgpack.EncodedJSON{},
				"/value",
				string(http.TimeFormatUnixNs),
			),
			Entry(
				"an explicit zero stays beside an omitted member",
				msgpack.EncodedJSON{"record": map[string]any{"value_pointer": ""}},
				"",
				string(http.TimeFormatUnixNs),
			),
		)

		It("Should default the omitted members of each lifted channel", func(
			ctx SpecContext,
		) {
			lifted := MustOpen(config.OpenService(
				ctx,
				config.ServiceConfig[opcua.ReadConfig]{
					DB:          db,
					Type:        "lifted_channels_test",
					SetEntryKey: (*opcua.ReadConfig).SetKey,
					Ladder: imex.Ladder[opcua.ReadConfig]{
						First:  1,
						Latest: 2,
						Decode: func(
							_ context.Context,
							_ imex.Version,
							data msgpack.EncodedJSON,
						) (opcua.ReadConfig, error) {
							var cfg opcua.ReadConfig
							for _, raw := range data["channels"].([]any) {
								ch := raw.(map[string]any)
								key, _ := ch["key"].(string)
								dataType, _ := ch["data_type"].(string)
								var rc opcua.ReadChannel
								rc.Key = key
								rc.DataType = telem.DataType(dataType)
								cfg.Channels = append(cfg.Channels, rc)
							}
							return cfg, nil
						},
					},
				},
			))
			out := MustSucceed(lifted.Normalize(ctx, 1, msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"key": "omitted"},
					map[string]any{"key": "explicit", "data_type": ""},
				},
			}))
			Expect(out["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "float32"),
				HaveKeyWithValue("data_type", ""),
			))
		})

		It(
			"Should give a legacy Arc config without an execution mode the default",
			func(
				ctx SpecContext,
			) {
				arcLegacy := MustOpen(config.OpenService(
					ctx,
					config.ServiceConfig[arctask.Config]{
						DB:            db,
						Type:          "arc_legacy_test",
						Ladder:        arcversions.ConfigLadder,
						LegacyVersion: arclegacy.LastVersion,
						Legacy:        &arclegacy.Config,
						SetEntryKey:   (*arctask.Config).SetKey,
					},
				))
				out := MustSucceed(arcLegacy.Normalize(ctx, 0, msgpack.EncodedJSON{
					"arc_key":        uuid.New().String(),
					"execution_mode": "",
				}))
				Expect(out).To(HaveKeyWithValue("performance", "auto"))
			},
		)

		It("Should keep an explicit zero in a latest-version config", func(
			ctx SpecContext,
		) {
			out := MustSucceed(
				svc.Normalize(ctx, arcversions.Latest, msgpack.EncodedJSON{
					"arc_key":     uuid.New().String(),
					"performance": "",
				}),
			)
			Expect(out).To(HaveKeyWithValue("performance", ""))
		})

		It("Should run the legacy rewrite on a version below the ladder's first", func(
			ctx SpecContext,
		) {
			arcKey := uuid.New()
			out := MustSucceed(versioned.Normalize(
				ctx,
				arcversions.ConfigLadder.First-1,
				msgpack.EncodedJSON{"module": arcKey.String()},
			))
			Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
		})

		It("Should reject data that does not decode at its version", func(
			ctx SpecContext,
		) {
			Expect(versioned.Normalize(
				ctx, arcversions.Latest, msgpack.EncodedJSON{"arc_key": 12},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding versioned_test config")),
			))
		})
	})

	Describe("Version", func() {
		It("Should report the ladder's latest version", func() {
			Expect(svc.Version()).To(Equal(arcversions.Latest))
		})
	})
})

var _ = Describe("Registry", func() {
	var store config.Store
	BeforeEach(func(ctx SpecContext) {
		store = MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:          db,
				Type:        testType,
				Ladder:      arcversions.ConfigLadder,
				SetEntryKey: (*arctask.Config).SetKey,
			},
		))
	})

	Describe("NewRegistry", func() {
		It("Should route each store by its type", func() {
			reg := MustSucceed(config.NewRegistry(store))
			Expect(reg.IsZero()).To(BeFalse())
			Expect(MustBeOk(reg.Store(testType))).To(BeIdenticalTo(store))
			Expect(reg.Types()).To(ConsistOf(testType))
		})

		It("Should reject two stores declaring the same type", func() {
			Expect(config.NewRegistry(store, store)).Error().
				To(MatchError(ContainSubstring("registered twice")))
		})
	})

	Describe("Store", func() {
		It("Should return false for an unclaimed type", func() {
			reg := MustSucceed(config.NewRegistry(store))
			_, ok := reg.Store("unclaimed")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("IsZero", func() {
		It("Should report true for a never-constructed registry", func() {
			Expect(config.Registry{}.IsZero()).To(BeTrue())
		})
	})
})
