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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("User", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.User{
				Key:       uuid.New(),
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Smith",
				RootUser:  true,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.User
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal(original.Key))
			Expect(decoded.Username).To(Equal("alice"))
			Expect(decoded.FirstName).To(Equal("Alice"))
			Expect(decoded.LastName).To(Equal("Smith"))
			Expect(decoded.RootUser).To(BeTrue())
		})

		It("Should decode legacy uppercase msgpack fields", func() {
			key := uuid.New()
			legacy := struct {
				Key      uuid.UUID
				Username string
			}{
				Key:      key,
				Username: "bob",
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.User
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal(key))
			Expect(decoded.Username).To(Equal("bob"))
		})

		It("Should decode mixed legacy and new fields", func() {
			key := uuid.New()
			mixed := map[string]any{
				"Key":        key,
				"username":   "charlie",
				"first_name": "Charlie",
				"last_name":  "Brown",
			}
			data := MustSucceed(msgpack.Marshal(mixed))
			var decoded v0.User
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal(key))
			Expect(decoded.Username).To(Equal("charlie"))
			Expect(decoded.FirstName).To(Equal("Charlie"))
			Expect(decoded.LastName).To(Equal("Brown"))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the user's key", func() {
			k := uuid.New()
			Expect(v0.User{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.User{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the user ontology identifier", func() {
			k := uuid.New()
			Expect(v0.User{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeUser, Key: k.String(),
			}))
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a User row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(xmsgpack.Codec)))
			e := v0.User{Key: uuid.New(), Username: "ada"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"User",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.User]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.User
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.User](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
