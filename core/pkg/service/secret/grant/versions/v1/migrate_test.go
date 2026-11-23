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
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/secret/grant/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/secret/grant/versions/v1"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v4"
	"github.com/synnaxlabs/x/encoding/msgpack"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should give a stored grant an empty command", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Grant]{DB: db}))
		old := v0.Grant{
			Key:       uuid.New(),
			Subject:   resource.ID{Type: resource.TypeRack, Key: "65537"},
			Secrets:   []secret.Key{uuid.New()},
			ExpiresAt: telem.TimeStamp(5),
		}
		Expect(gorp.NewCreate[v0.Key, v0.Grant]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Grant]{
			DB: db, Migrations: []gorp.Migration{v1.Migration},
		}))
		var got v1.Grant
		Expect(gorp.NewRetrieve[v1.Key, v1.Grant]().
			Where(gorp.MatchKeys[v1.Key, v1.Grant](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.Grant{
			Key:       old.Key,
			Subject:   old.Subject,
			Secrets:   old.Secrets,
			Command:   task.Command{Args: msgpack.EncodedJSON{}},
			ExpiresAt: old.ExpiresAt,
		}))
	})
})
