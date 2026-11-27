// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Paths over a stored folder cycle", func() {
	// A v0.62 Core moved a folder with no check of its own chain, so an upgraded store
	// can hold a chain that comes back to a folder it passed.
	It("Should return the path of a folder whose chain comes back to itself", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		first, second := uuid.New(), uuid.New()
		loop := []group.Group{
			{
				Key:    first,
				Name:   "plant",
				Scope:  resource.TypeChannel,
				Parent: new(group.ResourceID(second)),
			},
			{
				Key:    second,
				Name:   "rig",
				Scope:  resource.TypeChannel,
				Parent: new(group.ResourceID(first)),
			},
		}
		Expect(gorp.WrapWriter[group.Key, group.Group](db).Set(ctx, loop...)).
			To(Succeed())
		done := make(chan error, 1)
		go func() {
			_, err := group.Paths(ctx, db, first)
			done <- err
		}()
		Eventually(done).WithTimeout(time.Second).Should(Receive(Succeed()))
	})
})
