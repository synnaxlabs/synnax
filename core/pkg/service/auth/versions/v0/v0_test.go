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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("SecureCredentials", func() {
	Describe("GorpKey", func() {
		It("Should return the username", func() {
			Expect(v0.SecureCredentials{Username: "root"}.GorpKey()).To(Equal("root"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.SecureCredentials{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a SecureCredentials row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.SecureCredentials{Username: "ada", Password: []byte("hash")}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"SecureCredentials",
				e.GorpKey(),
				e,
			)
			table := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, v0.SecureCredentials]{
					DB:         db,
					Migrations: []gorp.Migration{v0.NormalizeKeys},
				}),
			)
			var res v0.SecureCredentials
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[string, v0.SecureCredentials](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
