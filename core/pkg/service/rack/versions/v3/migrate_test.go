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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep every field of each rack and leave its version absent", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Rack]{DB: db}))
		old := v2.Rack{
			Key:          v2.Key(0x0001_0002),
			Name:         "Node 1",
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		}
		Expect(gorp.NewCreate[v2.Key, v2.Rack]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Rack",
			Migrations: []gorp.Migration{v3.Migration},
		})).To(Succeed())
		var got v3.Rack
		Expect(gorp.NewRetrieve[v3.Key, v3.Rack]().
			Where(gorp.MatchKeys[v3.Key, v3.Rack](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v3.Rack{
			Key:          old.Key,
			Name:         "Node 1",
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		}))
	})
})
