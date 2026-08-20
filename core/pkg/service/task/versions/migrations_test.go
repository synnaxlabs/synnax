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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
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

var _ = Describe("Pre-v0.54 task key normalization", func() {
	It(
		"Should open the service and re-key a task stored under the legacy key prefix",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    g,
				Label:    labelSvc,
				Search:   searchIdx,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:           db,
				Ontology:     otg,
				Group:        g,
				HostProvider: mock.NewStaticHostProvider(1),
				Status:       stat,
				Search:       searchIdx,
			}))
			testRack := &rack.Rack{Name: "Ancient Rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

			legacyKey := v0.Key(uint64(testRack.Key)<<32 | 42)
			legacyRow := setPreV54Row(ctx, kvDB, "Task", uint64(legacyKey), v0.Task{
				Key:  legacyKey,
				Name: "Ancient Task",
				Type: pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-ancient",
				},
			})

			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
			configs := MustSucceed(config.NewRegistry(pd.Stores()...))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackSvc,
				Status:   stat,
				Search:   searchIdx,
				ImEx:     imex.NewService(),
				Configs:  configs,
			}))

			By("Carrying the task through the full chain to a UUID key")
			var migrated task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Ancient Task")).
				Entry(&migrated).
				Exec(ctx, nil)).To(Succeed())
			Expect(migrated.Key).ToNot(Equal(uuid.Nil))
			Expect(migrated.Rack).To(Equal(testRack.Key))
			Expect(migrated.Config).To(HaveKeyWithValue("routing_key", "rk-ancient"))

			By("Deleting the legacy-prefix row")
			Expect(db.Get(ctx, legacyRow)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
