// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v4"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should lift a stored v3 task to v4", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Task]{
			DB:   db,
			Name: string(resource.TypeTask),
		}))
		old := v3.Task{
			Key:        uuid.New(),
			Rack:       65538,
			Name:       "Pump",
			Type:       "rack_status",
			Config:     msgpack.EncodedJSON{"auto_start": true},
			ConfigHash: "hash",
			Internal:   true,
		}
		Expect(gorp.NewCreate[v3.Key, v3.Task]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  string(resource.TypeTask),
			Migrations: []gorp.Migration{v4.Migration},
		})).To(Succeed())
		var got v4.Task
		Expect(gorp.NewRetrieve[v4.Key, v4.Task]().
			Where(gorp.MatchKeys[v4.Key, v4.Task](old.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v4.Task{
			Key:        old.Key,
			Rack:       old.Rack,
			Name:       "Pump",
			Type:       "rack_status",
			Config:     msgpack.EncodedJSON{"auto_start": true},
			ConfigHash: "hash",
			Internal:   true,
		}))
	})
})
