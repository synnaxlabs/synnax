// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/apikey/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/auth/apikey/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should give a stored key no expiry", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.APIKey]{DB: db}))
		old := v0.APIKey{
			Key:     uuid.New(),
			Subject: resource.ID{Type: resource.TypeRack, Key: "65537"},
			Hash:    []byte{1, 2, 3},
			Name:    "driver",
			Created: telem.TimeStamp(5),
		}
		Expect(gorp.NewCreate[v0.Key, v0.APIKey]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.APIKey]{
			DB: db, Migrations: []gorp.Migration{v1.Migration},
		}))
		var got v1.APIKey
		Expect(gorp.NewRetrieve[v1.Key, v1.APIKey]().
			Where(gorp.MatchKeys[v1.Key, v1.APIKey](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.APIKey{
			Key:     old.Key,
			Subject: old.Subject,
			Hash:    old.Hash,
			Name:    old.Name,
			Created: old.Created,
		}))
	})
})
