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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Pre-v0.54 task key normalization", func() {
	It(
		"Should open the service and re-key a task stored under the legacy key prefix",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB: db,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: db,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:      db,
				HostKey: 1,
				Status:  stat,
				DriverRole: MustSucceed(
					racktestutil.CreateDriverRole(ctx, db, roleSvc),
				),
			}))
			testRack := &rack.Rack{Name: "Ancient Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, testRack)
			})).To(Succeed())

			legacyKey := v0.Key(uint64(testRack.Key)<<32 | 42)
			legacyRow := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Task",
				uint64(legacyKey),
				v0.Task{
					Key:  legacyKey,
					Name: "Ancient Task",
					Type: pagerduty.AlertTaskType,
					Config: msgpack.EncodedJSON{
						"routing_key": "rk-ancient",
					},
				},
			)

			secrets := secrettestutil.OpenService(ctx, db)
			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB:     db,
				Secret: secrets,
			}))
			configs := MustSucceed(config.NewRegistry(pd.Stores()...))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:      db,
				Rack:    rackSvc,
				Status:  stat,
				ImEx:    imex.NewService(),
				Configs: configs,
			}))

			By("Carrying the task through the full chain to a UUID key")
			var migrated task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Ancient Task")).
				Entry(&migrated).
				Exec(ctx, db)).To(Succeed())
			Expect(migrated.Key).ToNot(Equal(uuid.Nil()))
			Expect(migrated.Rack).To(Equal(testRack.Key))
			Expect(migrated.Config["routing_key"]).To(BeAssignableToTypeOf(""))
			routingKey := uuid.MustParse(migrated.Config["routing_key"].(string))
			Expect(secrets.Reveal(ctx, db, routingKey)).
				To(HaveKeyWithValue(routingKey, []byte("rk-ancient")))
			var routingSecret secret.Secret
			Expect(secrets.NewRetrieve().
				Where(secret.MatchKeys(routingKey)).
				Entry(&routingSecret).
				Exec(ctx, db)).To(Succeed())
			Expect(routingSecret.Base().Name).To(Equal("Ancient Task routing key"))

			By("Deleting the legacy-prefix row")
			Expect(db.Get(ctx, legacyRow)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
