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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// testType is the task type the suite registers its store under. The record type
// is the Arc task config, picked because it has both schema defaults and validation.
const testType = "arc_task"

var _ = Describe("Service", func() {
	var (
		svc   *config.Service[arctask.Config]
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:            db,
				Type:          testType,
				Ladder:        arcversions.ConfigLadder,
				SetEntryKey:   (*arctask.Config).SetKey,
				PatchEntry:    (*arctask.Config).Patch,
				HashEntry:     (*arctask.Config).HashContent,
				ValidateEntry: (*arctask.Config).Validate,
			},
		))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: []config.Store{svc},
		}))
	})
	createTask := func(ctx SpecContext) uuid.UUID {
		GinkgoHelper()
		return MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
			"arc_key": uuid.New().String(),
		}))
	}

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					Type:        testType,
					SetEntryKey: (*arctask.Config).SetKey,
					PatchEntry:  (*arctask.Config).Patch,
					HashEntry:   (*arctask.Config).HashContent,
				},
			)).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the type", func(ctx SpecContext) {
			Expect(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:          db,
					SetEntryKey: (*arctask.Config).SetKey,
					PatchEntry:  (*arctask.Config).Patch,
					HashEntry:   (*arctask.Config).HashContent,
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
					PatchEntry:  (*arctask.Config).Patch,
					HashEntry:   (*arctask.Config).HashContent,
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

	Describe("Prepare", func() {
		It("Should store a decoded config under the given key", func(
			ctx SpecContext,
		) {
			arcKey := uuid.New()
			key := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key":     arcKey.String(),
				"performance": "high",
			}))
			data := MustSucceed(svc.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["performance"]).To(Equal("high"))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			data := MustSucceed(svc.Read(ctx, db, key))
			Expect(data["performance"]).To(Equal("auto"))
		})

		It("Should keep an explicit empty field instead of defaulting it", func(
			ctx SpecContext,
		) {
			Expect(svc.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "",
			})).Error().To(MatchError(ContainSubstring("invalid performance")))
		})

		It("Should return the record as Read returns it once stored", func(
			ctx SpecContext,
		) {
			key := createTask(ctx)
			rec := MustSucceed(svc.Prepare(ctx, db, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(MustSucceed(rec.Data())["performance"]).To(Equal("auto"))
			Expect(rec.Write(ctx, db)).To(Succeed())
			Expect(
				MustSucceed(rec.Data()),
			).To(Equal(MustSucceed(svc.Read(ctx, db, key))))
		})

		It("Should not write the record before Write", func(ctx SpecContext) {
			key := uuid.New()
			MustSucceed(svc.Prepare(ctx, db, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(svc.Read(ctx, db, key)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should not store a record whose key is not a task's key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			rec := MustSucceed(svc.Prepare(ctx, db, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(rec.Write(ctx, db)).
				To(MatchError(ContainSubstring("key: task %s does not exist", key)))
			Expect(svc.Read(ctx, db, key)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "high",
			}))
			Expect(MustSucceed(svc.Prepare(ctx, db, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "low",
			})).Write(ctx, db)).To(Succeed())
			data := MustSucceed(svc.Read(ctx, db, key))
			Expect(data["performance"]).To(Equal("low"))
		})

		It("Should return a validation error when the data does not decode", func(
			ctx SpecContext,
		) {
			Expect(svc.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
				"arc_key": 123,
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "BOGUS",
			})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
		})
	})

	Describe("Edit", func() {
		set := func(path []string, value any) patch.Cursor {
			GinkgoHelper()
			return MustSucceed(patch.Begin(patch.Set{
				Path:  append([]string{"config"}, path...),
				Value: value,
			})).Next()
		}

		It("Should apply the actions to the stored record", func(ctx SpecContext) {
			key := createTask(ctx)
			rec, changed := MustSucceed2(svc.Edit(
				ctx,
				db,
				key,
				[]patch.Cursor{set([]string{"performance"}, "high")},
			))
			Expect(changed).To(BeFalse())
			Expect(rec.Write(ctx, db)).To(Succeed())
			data := MustSucceed(svc.Read(ctx, db, key))
			Expect(data["performance"]).To(Equal("high"))
			Expect(MustSucceed(rec.Data())).To(Equal(data))
		})

		It("Should keep an explicit zero that an action sets", func(ctx SpecContext) {
			key := createTask(ctx)
			_, _, err := svc.Edit(
				ctx,
				db,
				key,
				[]patch.Cursor{set([]string{"performance"}, "")},
			)
			Expect(err).To(MatchError(ContainSubstring("invalid performance: ")))
		})

		It("Should report a change when a value lacks a member", func(
			ctx SpecContext,
		) {
			key := createTask(ctx)
			arc := uuid.New()
			rec, changed := MustSucceed2(svc.Edit(ctx, db, key, []patch.Cursor{
				set(nil, map[string]any{"arc_key": arc.String()}),
			}))
			Expect(changed).To(BeTrue())
			Expect(MustSucceed(rec.Data())).To(SatisfyAll(
				HaveKeyWithValue("arc_key", arc.String()),
				HaveKeyWithValue("performance", "auto"),
			))
		})

		It("Should report a change when an action changes the key", func(
			ctx SpecContext,
		) {
			key := createTask(ctx)
			rec, changed := MustSucceed2(svc.Edit(ctx, db, key, []patch.Cursor{
				set([]string{"key"}, uuid.New().String()),
			}))
			Expect(changed).To(BeTrue())
			Expect(MustSucceed(rec.Data())).To(HaveKeyWithValue("key", key.String()))
		})

		It("Should hash the record as Hash does once written", func(ctx SpecContext) {
			key := createTask(ctx)
			rec := MustSucceed(svc.Prepare(ctx, db, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "high",
			}))
			Expect(rec.Write(ctx, db)).To(Succeed())
			Expect(rec.Hash()).To(HaveLen(16))
			Expect(MustSucceed(svc.Hash(ctx, db, key))).To(Equal(rec.Hash()))
		})

		It("Should hash records with the same content under other keys equally", func(
			ctx SpecContext,
		) {
			arc := uuid.New().String()
			first := MustSucceed(
				svc.Prepare(ctx, db, createTask(ctx), msgpack.EncodedJSON{
					"arc_key": arc,
				}),
			)
			second := MustSucceed(
				svc.Prepare(ctx, db, createTask(ctx), msgpack.EncodedJSON{
					"arc_key": arc,
				}),
			)
			third := MustSucceed(
				svc.Prepare(ctx, db, createTask(ctx), msgpack.EncodedJSON{
					"arc_key":     arc,
					"performance": "high",
				}),
			)
			Expect(first.Hash()).To(Equal(second.Hash()))
			Expect(first.Hash()).ToNot(Equal(third.Hash()))
		})

		It("Should reject a path to a member the record does not declare", func(
			ctx SpecContext,
		) {
			Expect(svc.Edit(ctx, db, createTask(ctx), []patch.Cursor{
				set([]string{"performnace"}, "high"),
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the path names no editable field")),
			))
		})

		It("Should reject a value holding a member the record does not declare", func(
			ctx SpecContext,
		) {
			Expect(svc.Edit(ctx, db, createTask(ctx), []patch.Cursor{
				set(nil, map[string]any{
					"arc_key":     uuid.New().String(),
					"performnace": "high",
				}),
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring(`unknown object member name "performnace"`),
				),
			))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.Edit(ctx, db, createTask(ctx), []patch.Cursor{
				set([]string{"performance"}, "BOGUS"),
			})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
		})

		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Edit(ctx, db, uuid.New(), []patch.Cursor{
				set([]string{"performance"}, "high"),
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Validate", func() {
		It("Should drop a member the record does not declare", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
				"extra":   true,
			}))
			Expect(MustSucceed(svc.Read(ctx, db, key))).ToNot(HaveKey("extra"))
		})

		Context("With a resolve hook", func() {
			// openResolving opens the store on a DB of its own, as two stores of one
			// type cannot share a DB.
			openResolving := func(
				ctx SpecContext,
				resolve func(context.Context, gorp.Snapshot, *arctask.Config) error,
			) (*config.Service[arctask.Config], *tasktestutil.Tasks, *gorp.DB) {
				GinkgoHelper()
				db := DeferClose(gorp.Wrap(memkv.New()))
				resolving := MustOpen(config.OpenService(
					ctx,
					config.ServiceConfig[arctask.Config]{
						DB:            db,
						Type:          testType,
						Ladder:        arcversions.ConfigLadder,
						SetEntryKey:   (*arctask.Config).SetKey,
						PatchEntry:    (*arctask.Config).Patch,
						HashEntry:     (*arctask.Config).HashContent,
						ResolveEntry:  resolve,
						ValidateEntry: (*arctask.Config).Validate,
					},
				))
				return resolving, MustOpen(tasktestutil.OpenTasks(
					ctx,
					tasktestutil.TasksConfig{
						DB:     db,
						Stores: []config.Store{resolving},
					},
				)), db
			}

			It("Should store the entry the hook resolves, before validating it", func(
				ctx SpecContext,
			) {
				resolving, tasks, db := openResolving(ctx, func(
					_ context.Context,
					snap gorp.Snapshot,
					c *arctask.Config,
				) error {
					Expect(snap).ToNot(BeNil())
					c.Performance = "low"
					return nil
				})
				key := MustSucceed(tasks.Create(ctx, resolving, msgpack.EncodedJSON{
					"arc_key":     uuid.New().String(),
					"performance": "BOGUS",
				}))
				data := MustSucceed(resolving.Read(ctx, db, key))
				Expect(data["performance"]).To(Equal("low"))
			})

			It("Should return the hook's error and store nothing", func(
				ctx SpecContext,
			) {
				resolving, _, db := openResolving(ctx, func(
					context.Context,
					gorp.Snapshot,
					*arctask.Config,
				) error {
					return errors.New("resolve failed")
				})
				key := uuid.New()
				Expect(resolving.Prepare(ctx, db, key, msgpack.EncodedJSON{
					"arc_key": uuid.New().String(),
				})).Error().To(MatchError("resolve failed"))
				Expect(resolving.Read(ctx, db, key)).Error().
					To(MatchError(query.ErrNotFound))
			})
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Read(ctx, db, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			}))
			Expect(svc.Delete(ctx, db, key)).To(Succeed())
			Expect(svc.Read(ctx, db, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Delete(ctx, db, key)).To(Succeed())
		})

		It("Should remove multiple records in one call", func(ctx SpecContext) {
			k1, k2 := createTask(ctx), createTask(ctx)
			Expect(svc.Delete(ctx, db, k1, k2)).To(Succeed())
			Expect(svc.Read(ctx, db, k1)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Read(ctx, db, k2)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			arcKey := uuid.New()
			from := MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key": arcKey.String(),
			}))
			to := createTask(ctx)
			Expect(svc.Copy(ctx, db, from, to)).To(Succeed())
			data := MustSucceed(svc.Read(ctx, db, to))
			Expect(data["key"]).To(Equal(to.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			original := MustSucceed(svc.Read(ctx, db, from))
			Expect(original["key"]).To(Equal(from.String()))
		})

		It("Should return not found when the source is missing", func(
			ctx SpecContext,
		) {
			Expect(svc.Copy(ctx, db, uuid.New(), uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Normalize", func() {
		var versioned *config.Service[arctask.Config]
		BeforeEach(func(ctx SpecContext) {
			versioned = MustOpen(config.OpenService(
				ctx,
				config.ServiceConfig[arctask.Config]{
					DB:            DeferClose(gorp.Wrap(memkv.New())),
					Type:          "versioned_test",
					Ladder:        arcversions.ConfigLadder,
					LegacyVersion: 1,
					SetEntryKey:   (*arctask.Config).SetKey,
					PatchEntry:    (*arctask.Config).Patch,
					HashEntry:     (*arctask.Config).HashContent,
					Legacy: &legacy.Rewrite{
						Post: func(cfg msgpack.EncodedJSON) {
							legacy.RenameKey(cfg, "module", "arc_key")
						},
					},
				},
			))
		})

		It("Should reject a version above the ladder's latest", func(ctx SpecContext) {
			Expect(
				versioned.Normalize(
					ctx,
					db,
					arcversions.Latest+1,
					msgpack.EncodedJSON{},
				),
			).Error().
				To(MatchError(ContainSubstring(fmt.Sprintf(
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
			out := MustSucceed(versioned.Normalize(ctx, db,
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
				out := MustSucceed(versioned.Normalize(ctx, db, 0, msgpack.EncodedJSON{
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
					DB:          DeferClose(gorp.Wrap(memkv.New())),
					Type:        "era_only_test",
					Ladder:      arcversions.ConfigLadder,
					SetEntryKey: (*arctask.Config).SetKey,
					PatchEntry:  (*arctask.Config).Patch,
					HashEntry:   (*arctask.Config).HashContent,
				},
			))
			arcKey := uuid.New()
			out := MustSucceed(eraOnly.Normalize(ctx, db, 0, msgpack.EncodedJSON{
				"arcKey":     arcKey.String(),
				"dataSaving": false,
			}))
			Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
			Expect(out).To(HaveKeyWithValue("data_saving_disabled", true))
		})

		It("Should apply the store's defaults to the normalized config", func(
			ctx SpecContext,
		) {
			out := MustSucceed(svc.Normalize(ctx, db,
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
						DB:          DeferClose(gorp.Wrap(memkv.New())),
						Type:        "lifted_test",
						SetEntryKey: (*arctask.Config).SetKey,
						PatchEntry:  (*arctask.Config).Patch,
						HashEntry:   (*arctask.Config).HashContent,
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
				out := MustSucceed(lifted.Normalize(ctx, db, 1, in))
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
						DB:          DeferClose(gorp.Wrap(memkv.New())),
						Type:        "lifted_record_test",
						SetEntryKey: (*kafka.WriteConfig).SetKey,
						PatchEntry:  (*kafka.WriteConfig).Patch,
						HashEntry:   (*kafka.WriteConfig).HashContent,
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
				out := MustSucceed(lifted.Normalize(ctx, db, 1, in))
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
					DB:          DeferClose(gorp.Wrap(memkv.New())),
					Type:        "lifted_channels_test",
					SetEntryKey: (*opcua.ReadConfig).SetKey,
					PatchEntry:  (*opcua.ReadConfig).Patch,
					HashEntry:   (*opcua.ReadConfig).HashContent,
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
			out := MustSucceed(lifted.Normalize(ctx, db, 1, msgpack.EncodedJSON{
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
						DB:            DeferClose(gorp.Wrap(memkv.New())),
						Type:          "arc_legacy_test",
						Ladder:        arcversions.ConfigLadder,
						LegacyVersion: arclegacy.LastVersion,
						Legacy:        &arclegacy.Config,
						SetEntryKey:   (*arctask.Config).SetKey,
						PatchEntry:    (*arctask.Config).Patch,
						HashEntry:     (*arctask.Config).HashContent,
					},
				))
				out := MustSucceed(arcLegacy.Normalize(ctx, db, 0, msgpack.EncodedJSON{
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
				svc.Normalize(ctx, db, arcversions.Latest, msgpack.EncodedJSON{
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
			out := MustSucceed(versioned.Normalize(ctx, db,
				arcversions.ConfigLadder.First-1,
				msgpack.EncodedJSON{"module": arcKey.String()},
			))
			Expect(out).To(HaveKeyWithValue("arc_key", arcKey.String()))
		})

		It("Should reject data that does not decode at its version", func(
			ctx SpecContext,
		) {
			Expect(
				versioned.Normalize(
					ctx,
					db,
					arcversions.Latest,
					msgpack.EncodedJSON{"arc_key": 12},
				),
			).Error().
				To(SatisfyAll(
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
				PatchEntry:  (*arctask.Config).Patch,
				HashEntry:   (*arctask.Config).HashContent,
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

// target is a record that the detach reference of the "With a detach reference" specs
// points at.
type target struct{ Key uuid.UUID }

func (t target) GorpKey() uuid.UUID { return t.Key }

const targetTable = "config_target"

var _ = Describe("TableName", func() {
	It("Should name the table after the task type", func() {
		Expect(config.TableName("modbus_read")).To(Equal("modbus_read_config"))
	})
})

var _ = Describe("With a detach reference", func() {
	var (
		ldb     *gorp.DB
		targets *gorp.Table[uuid.UUID, target]
	)
	reference := func() gorp.Reference[uuid.UUID, arctask.Config] {
		return gorp.NewKeyReference(
			gorp.KeyReferenceConfig[uuid.UUID, arctask.Config, uuid.UUID]{
				Name:     "arc_key",
				Table:    targetTable,
				OnDelete: gorp.Detach,
				Kind:     gorp.Use,
				Targets: func(c *arctask.Config) []uuid.UUID {
					if c.ArcKey == uuid.Nil() {
						return nil
					}
					return []uuid.UUID{c.ArcKey}
				},
				Detach: func(c *arctask.Config, _ uuid.UUID) { c.ArcKey = uuid.Nil() },
			},
		)
	}
	openService := func(ctx SpecContext) *config.Service[arctask.Config] {
		GinkgoHelper()
		return MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:          ldb,
				Type:        testType,
				Ladder:      arcversions.ConfigLadder,
				SetEntryKey: (*arctask.Config).SetKey,
				PatchEntry:  (*arctask.Config).Patch,
				HashEntry:   (*arctask.Config).HashContent,
				References: []gorp.Reference[uuid.UUID, arctask.Config]{
					reference(),
				},
			},
		))
	}
	createTarget := func(ctx SpecContext) target {
		GinkgoHelper()
		t := target{Key: uuid.New()}
		Expect(targets.NewCreate().Entry(&t).Exec(ctx, ldb)).To(Succeed())
		return t
	}
	BeforeEach(func(ctx SpecContext) {
		ldb = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		targets = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, target]{
			DB:   ldb,
			Name: targetTable,
		}))
	})

	Describe("Normalize", func() {
		normalize := func(ctx SpecContext, arcKey uuid.UUID) msgpack.EncodedJSON {
			GinkgoHelper()
			return MustSucceed(openService(ctx).Normalize(
				ctx,
				ldb,
				arcversions.Latest,
				msgpack.EncodedJSON{"arc_key": arcKey.String()},
			))
		}

		It(
			"Should zero a reference whose target does not exist",
			func(ctx SpecContext) {
				Expect(
					normalize(ctx, uuid.New())["arc_key"],
				).To(Equal(uuid.Nil().String()))
			},
		)

		It("Should keep a reference whose target exists", func(ctx SpecContext) {
			t := createTarget(ctx)
			Expect(normalize(ctx, t.Key)["arc_key"]).To(Equal(t.Key.String()))
		})
	})

	Describe("OnDetach", func() {
		var (
			svc   *config.Service[arctask.Config]
			tasks *tasktestutil.Tasks
			t     target
			key   uuid.UUID
		)
		BeforeEach(func(ctx SpecContext) {
			svc = openService(ctx)
			tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
				DB:     ldb,
				Stores: []config.Store{svc},
			}))
			t = createTarget(ctx)
			key = MustSucceed(tasks.Create(ctx, svc, msgpack.EncodedJSON{
				"arc_key": t.Key.String(),
			}))
		})
		deleteTarget := func(ctx SpecContext, tx gorp.Tx) error {
			return targets.NewDelete().
				Where(gorp.MatchKeys[uuid.UUID, target](t.Key)).
				Exec(ctx, tx)
		}
		configHash := func(ctx SpecContext, tx gorp.Tx) string {
			GinkgoHelper()
			var res task.Task
			Expect(tasks.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			return res.ConfigHash
		}

		It(
			"Should run the hook in the delete's transaction after the detach",
			func(ctx SpecContext) {
				var (
					detached []uuid.UUID
					arcKeys  []any
				)
				svc.OnDetach(func(ctx context.Context, tx gorp.Tx, k uuid.UUID) error {
					detached = append(detached, k)
					arcKeys = append(
						arcKeys,
						MustSucceed(svc.Read(ctx, tx, k))["arc_key"],
					)
					return nil
				})
				Expect(deleteTarget(ctx, ldb)).To(Succeed())
				Expect(detached).To(Equal([]uuid.UUID{key}))
				Expect(arcKeys).To(Equal([]any{uuid.Nil().String()}))
			},
		)

		It("Should fail the delete when the hook fails", func(ctx SpecContext) {
			svc.OnDetach(func(context.Context, gorp.Tx, uuid.UUID) error {
				return errors.New("hook failed")
			})
			Expect(deleteTarget(ctx, ldb)).To(MatchError("hook failed"))
			Expect(targets.NewRetrieve().
				Where(gorp.MatchKeys[uuid.UUID, target](t.Key)).
				Exists(ctx, ldb)).To(BeTrue())
		})

		It(
			"Should let the task service store the new config hash in the delete's "+
				"transaction",
			func(ctx SpecContext) {
				before := configHash(ctx, ldb)
				tx := DeferClose(ldb.OpenTx())
				Expect(deleteTarget(ctx, tx)).To(Succeed())
				Expect(configHash(ctx, tx)).ToNot(Equal(before))
				Expect(configHash(ctx, ldb)).To(Equal(before))
			},
		)
	})

	Describe("OpenService", func() {
		It(
			"Should drop missing targets from stored records when it opens",
			func(ctx SpecContext) {
				raw := MustOpen(
					gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, arctask.Config]{
						DB:   ldb,
						Name: config.TableName(testType),
					}),
				)
				stored := arctask.Config{ArcKey: uuid.New()}
				stored.SetKey(uuid.New())
				Expect(raw.NewCreate().Entry(&stored).Exec(ctx, ldb)).To(Succeed())
				Expect(raw.Close()).To(Succeed())
				svc := openService(ctx)
				Expect(MustSucceed(svc.Read(ctx, ldb, stored.GorpKey()))["arc_key"]).
					To(Equal(uuid.Nil().String()))
			},
		)
	})
})

var _ = Describe("TaskOf", func() {
	It("Should return the task whose config record a target names", func() {
		key := uuid.New()
		id := MustBeOk(config.TaskOf(gorp.NewTarget(config.TableName(testType), key)))
		Expect(id).To(Equal(task.ResourceID(key)))
	})

	It("Should return false for a table that stores no config records", func() {
		_, ok := config.TaskOf(gorp.NewTarget("task", uuid.New()))
		Expect(ok).To(BeFalse())
	})
})
