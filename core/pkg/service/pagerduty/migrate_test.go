// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("SecretMigration", func() {
	var (
		db      *gorp.DB
		secrets *secret.Service
		svc     *pd.Service
		// names holds the task names that the lookup returns.
		names map[uuid.UUID]string
		// asked holds the keys of each lookup call.
		asked  [][]uuid.UUID
		lookup pd.TaskNames
	)
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		names, asked = map[uuid.UUID]string{}, nil
		lookup = func(
			_ context.Context,
			_ gorp.Tx,
			keys []uuid.UUID,
		) (map[uuid.UUID]string, error) {
			asked = append(asked, keys)
			return names, nil
		}
	})

	// store writes a v1 config that the task named name owns.
	store := func(ctx SpecContext, name, routingKey string) uuid.UUID {
		GinkgoHelper()
		old := v1.TaskConfig{Key: uuid.New(), RoutingKey: routingKey}
		Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).Set(ctx, old)).
			To(Succeed())
		names[old.Key] = name
		return old.Key
	}
	// open opens the service, which lifts the stored configs and stages their keys.
	open := func(ctx SpecContext) {
		GinkgoHelper()
		MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		secrets = secrettestutil.OpenService(ctx, db)
		svc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{DB: db, Secret: secrets}))
	}
	run := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.SecretMigration(lookup).Run(ctx, tx, alamos.Instrumentation{})
		})
	}

	// expectStaged asserts whether the plain routing key of the config is staged.
	expectStaged := func(key uuid.UUID, staged bool) {
		GinkgoHelper()
		kvKey := v2.PlainRoutingKeyKVKey(key)
		if !staged {
			Expect(db.Get(kvKey)).Error().To(MatchError(query.ErrNotFound))
			return
		}
		b, closer := MustSucceed2(db.Get(kvKey))
		Expect(string(b)).To(Equal(plainKey))
		Expect(closer.Close()).To(Succeed())
	}

	It("Should move each staged routing key into a secret named after its task", func(
		ctx SpecContext,
	) {
		withKey := store(ctx, "Pump alerts", plainKey)
		withoutKey := store(ctx, "Draft", "")
		open(ctx)
		expectStaged(withKey, true)
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(Equal([][]uuid.UUID{{withKey}}))
		expectSecret(
			ctx,
			db,
			secrets,
			routingKeyOf(MustSucceed(svc.Alert.Read(ctx, db, withKey))),
			"Pump alerts routing key",
		)
		expectStaged(withKey, false)
		Expect(MustSucceed(svc.Alert.Read(ctx, db, withoutKey))).
			ToNot(HaveKey("routing_key"))
	})

	It("Should change nothing when it runs again", func(ctx SpecContext) {
		key := store(ctx, "Pump alerts", plainKey)
		open(ctx)
		Expect(run(ctx)).To(Succeed())
		first := routingKeyOf(MustSucceed(svc.Alert.Read(ctx, db, key)))
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(HaveLen(1))
		Expect(routingKeyOf(MustSucceed(svc.Alert.Read(ctx, db, key)))).
			To(Equal(first))
		Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(1))
	})

	It("Should do nothing without a staged routing key", func(ctx SpecContext) {
		store(ctx, "Draft", "")
		open(ctx)
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(0))
	})

	It("Should delete the staged routing key of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := store(ctx, "Snapshot", plainKey)
		open(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Alert.Delete(ctx, tx, key)
		})).To(Succeed())
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		expectStaged(key, false)
		Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(0))
	})

	It("Should keep the staged routing key when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := store(ctx, "Pump alerts", plainKey)
		open(ctx)
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("tasks of PagerDuty configs: no task"))
		expectStaged(key, true)
		Expect(MustSucceed(svc.Alert.Read(ctx, db, key))).ToNot(HaveKey("routing_key"))
		Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(0))
	})
})
