// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project/versions"
	v0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// setPreV54Row writes a raw KV row in the key format releases before v0.54 used:
// msgpack(typeName) + msgpack(key), with an msgpack-encoded value.
func setPreV54Row(
	ctx context.Context,
	kvDB kv.DB,
	typeName string,
	key, value any,
) []byte {
	GinkgoHelper()
	prefix := MustSucceed(msgpack.Codec.Encode(ctx, typeName))
	encodedKey := MustSucceed(msgpack.Codec.Encode(ctx, key))
	fullKey := make([]byte, 0, len(prefix)+len(encodedKey))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, encodedKey...)
	Expect(kvDB.Set(ctx, fullKey, MustSucceed(msgpack.Codec.Encode(ctx, value)))).
		To(Succeed())
	return fullKey
}

var _ = Describe("Pre-v0.54 workspace key normalization", func() {
	It(
		"Should lift a workspace stored under the legacy msgpack key prefix",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB))
			wsKey := uuid.New()
			layout := msgpack.EncodedJSON{"mosaic": "tree"}
			legacyRow := setPreV54Row(ctx, kvDB, "Workspace", wsKey, v0.Workspace{
				Key:    wsKey,
				Name:   "Ops",
				Layout: layout,
			})

			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			table := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[versions.Key, versions.Project]{
					DB: db,
					Migrations: versions.NewMigrations(
						versions.MigrationsConfig{Ontology: otg},
					),
				},
			))

			By("Lifting the workspace into a project with identical fields")
			var p versions.Project
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[versions.Key, versions.Project](wsKey)).
				Entry(&p).
				Exec(ctx, db)).To(Succeed())
			Expect(p).To(Equal(versions.Project{
				Key:    wsKey,
				Name:   "Ops",
				Layout: layout,
			}))

			By("Staging the layout blob for the panel composition migration")
			blob, closer := MustSucceed2(
				db.Get(ctx, []byte(v1.LegacyLayoutKVPrefix+wsKey.String())),
			)
			var got msgpack.EncodedJSON
			Expect(json.Unmarshal(blob, &got)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(got).To(Equal(layout))

			By("Deleting the legacy-prefix row")
			Expect(db.Get(ctx, legacyRow)).Error().To(MatchError(query.ErrNotFound))
		},
	)

	It(
		"Should be a no-op on a store with no legacy-prefix rows",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			table := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[versions.Key, versions.Project]{
					DB: db,
					Migrations: versions.NewMigrations(
						versions.MigrationsConfig{Ontology: otg},
					),
				},
			))
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[versions.Key, versions.Project](uuid.New())).
				Exists(ctx, db)).To(BeFalse())
		},
	)
})
