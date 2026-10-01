// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("KeysByUsername", func() {
	It("Should index every user's key by its username", func(ctx SpecContext) {
		db := gorp.Wrap(DeferClose(memkv.New()))
		alice := v0.User{Key: uuid.New(), Username: "alice"}
		bob := v0.User{Key: uuid.New(), Username: "bob"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[v0.Key, v0.User](tx).Set(ctx, alice, bob)
		})).To(Succeed())
		tx := DeferClose(db.OpenTx())
		Expect(v0.KeysByUsername(ctx, tx)).To(Equal(map[string]v0.Key{
			"alice": alice.Key,
			"bob":   bob.Key,
		}))
	})
})
