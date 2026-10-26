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
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// testType is the task type the suite registers its store under. The record type
// is the Arc task config, picked because it exercises both the ApplyDefaults and
// Validate hooks.
const testType = "arc_task"

var _ = Describe("Service", func() {
	var svc *config.Service[arctask.Config]
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:                 db,
				Type:               testType,
				Ladder:             arcversions.ConfigLadder,
				SetEntryKey:        (*arctask.Config).SetKey,
				ApplyEntryDefaults: (*arctask.Config).ApplyDefaults,
				ValidateEntry:      (*arctask.Config).Validate,
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
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     arcKey.String(),
				"performance": "high",
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["performance"]).To(Equal("high"))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["performance"]).To(Equal("auto"))
		})

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "high",
			})).To(Succeed())
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "low",
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["performance"]).To(Equal("low"))
		})

		It("Should return a validation error when the data does not decode", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key": 123,
			})).To(MatchError(validate.ErrValidation))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"performance": "BOGUS",
			})).To(MatchError(ContainSubstring("invalid performance: BOGUS")))
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
				Expect(resolving.Write(ctx, nil, key, msgpack.EncodedJSON{
					"arc_key":     uuid.New().String(),
					"performance": "BOGUS",
				})).To(Succeed())
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
				})).To(MatchError("resolve failed"))
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
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
		})

		It("Should remove multiple records in one call", func(ctx SpecContext) {
			k1, k2 := uuid.New(), uuid.New()
			for _, k := range []uuid.UUID{k1, k2} {
				Expect(svc.Write(ctx, nil, k, msgpack.EncodedJSON{
					"arc_key": uuid.New().String(),
				})).To(Succeed())
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
			Expect(svc.Write(ctx, nil, from, msgpack.EncodedJSON{
				"arc_key": arcKey.String(),
			})).To(Succeed())
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
