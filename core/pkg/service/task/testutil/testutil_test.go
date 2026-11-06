// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Tasks", func() {
	var (
		svc    *arctask.Service
		tasks  *tasktestutil.Tasks
		arcKey arc.Key
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := arc.Arc{Key: uuid.New(), Name: "arc"}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		arcKey = a.Key
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	It("Should create a task whose config lives in the store", func(ctx SpecContext) {
		key := MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key": arcKey.String(),
		}))
		data := MustSucceed(svc.Config.Read(ctx, nil, key))
		Expect(data["arc_key"]).To(Equal(arcKey.String()))
		Expect(data["performance"]).To(Equal("auto"))
	})

	It("Should return the error of a config the store rejects", func(ctx SpecContext) {
		Expect(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key":     uuid.New().String(),
			"performance": "BOGUS",
		})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
	})
})
