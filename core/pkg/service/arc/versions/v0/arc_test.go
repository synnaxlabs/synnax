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
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v0"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("StatusDetails", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.StatusDetails{Running: true}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should decode legacy uppercase Go field name", func() {
			legacy := struct{ Running bool }{Running: true}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should handle false value correctly for both formats", func() {
			original := v0.StatusDetails{Running: false}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeFalse())
		})
	})
})

var _ = Describe("Mode", func() {
	Describe("IsValid", func() {
		DescribeTable("Should report whether the mode is defined",
			func(m v0.Mode, valid bool) {
				Expect(m.IsValid()).To(Equal(valid))
			},
			Entry("text", v0.ModeText, true),
			Entry("graph", v0.ModeGraph, true),
			Entry("unknown", v0.Mode("bogus"), false),
			Entry("empty", v0.Mode(""), false),
		)
	})
})

var _ = Describe("Arc", func() {
	Describe("GorpKey", func() {
		It("Should return the Arc's key", func() {
			k := uuid.New()
			Expect(v0.Arc{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Arc{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a Arc row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(xmsgpack.Codec)))
			e := v0.Arc{Key: uuid.New(), Name: "Autosequence"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Arc",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Arc]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.Arc
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.Arc](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
