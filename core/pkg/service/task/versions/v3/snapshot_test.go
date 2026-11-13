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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("NewSnapshotMigration", func() {
	var (
		db        *gorp.DB
		statusSvc *status.Service
		rt        *racktask.Service
		logs      *observer.ObservedLogs
		ins       alamos.Instrumentation
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Task]{
			DB:   db,
			Name: string(resource.TypeTask),
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		rt = MustOpen(racktask.OpenService(ctx, racktask.ServiceConfig{DB: db}))
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		ins = alamos.New("test", alamos.WithLogger(logger))
	})

	// create stores t with a status and, when t has the rack status type, a config
	// record.
	create := func(ctx SpecContext, t v2.Task) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).
			To(Succeed())
		id := resource.ID{Type: resource.TypeTask, Key: t.Key.String()}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if t.Type == rt.Status.Type() {
				rec, err := rt.Status.Prepare(ctx, tx, t.Key, msgpack.EncodedJSON{})
				if err != nil {
					return err
				}
				if err = rec.Write(ctx, tx); err != nil {
					return err
				}
			}
			return statusSvc.NewMigrationWriter(tx).Set(ctx, status.Status[any]{
				Key:     uuid.New().String(),
				Owner:   &id,
				Name:    t.Name,
				Variant: status.VariantSuccess,
				Time:    telem.Now(),
			})
		})).To(Succeed())
	}

	run := func(ctx SpecContext) error {
		GinkgoHelper()
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       string(resource.TypeTask),
			Migrations: []gorp.Migration{
				v3.NewSnapshotMigration(v3.MigrationConfig{
					Status:  statusSvc,
					Configs: MustSucceed(config.NewRegistry(rt.Stores()...)),
				}),
			},
		})
	}

	hasStatus := func(ctx SpecContext, key v2.Key) bool {
		GinkgoHelper()
		id := resource.ID{Type: resource.TypeTask, Key: key.String()}
		err := statusSvc.NewRetrieve[any]().
			Where(status.MatchOwners[any](id)).
			Entry(new(status.Status[any])).
			Exec(ctx, db)
		Expect(err).To(Or(Succeed(), MatchError(query.ErrNotFound)))
		return err == nil
	}

	It(
		"Should delete a snapshot task with its config record and status",
		func(ctx SpecContext) {
			snapshot := v2.Task{
				Key:      uuid.New(),
				Name:     "Snapshot",
				Type:     "rack_status",
				Snapshot: true,
			}
			create(ctx, snapshot)
			Expect(run(ctx)).To(Succeed())
			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](snapshot.Key)).
				Exists(ctx, db)).To(BeFalse())
			Expect(rt.Status.Read(ctx, db, snapshot.Key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(hasStatus(ctx, snapshot.Key)).To(BeFalse())
			Expect(logs.FilterMessage("deleted a snapshot task").All()).To(ConsistOf(
				WithTransform(
					func(e observer.LoggedEntry) map[string]any { return e.ContextMap() },
					Equal(map[string]any{
						"key":  snapshot.Key.String(),
						"name": "Snapshot",
						"type": "rack_status",
					}),
				),
			))
		},
	)

	It("Should keep a task that is not a snapshot", func(ctx SpecContext) {
		live := v2.Task{Key: uuid.New(), Name: "Live", Type: "rack_status"}
		create(ctx, live)
		Expect(run(ctx)).To(Succeed())
		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](live.Key)).
			Exists(ctx, db)).To(BeTrue())
		Expect(rt.Status.Read(ctx, db, live.Key)).
			To(HaveKeyWithValue("key", live.Key.String()))
		Expect(hasStatus(ctx, live.Key)).To(BeTrue())
		Expect(logs.All()).To(BeEmpty())
	})

	It(
		"Should return an error when a snapshot has a type with no config store",
		func(ctx SpecContext) {
			key := uuid.New()
			create(ctx, v2.Task{Key: key, Name: "Bogus", Type: "bogus", Snapshot: true})
			Expect(run(ctx)).To(MatchError(ContainSubstring(
				`unknown type "bogus" of snapshot task ` + key.String(),
			)))
		},
	)
})
