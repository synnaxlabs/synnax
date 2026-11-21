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
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep each message and give none a cycle time", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.ICD]{DB: db}))
		pld := v0.Payload{Variant: v0.BinaryPayload{
			Length: new(uint16(2)),
			Fields: []v0.BinaryField{{
				Key:       uuid.New(),
				Name:      "speed",
				BitLength: 16,
				ByteOrder: v0.ByteOrderLittleEndian,
				Kind:      payload.KindUnsigned,
			}},
		}}
		old := v0.ICD{
			Key:      uuid.New(),
			Name:     "engine",
			Messages: []v0.Message{{Key: uuid.New(), Name: "status", Payload: pld}},
		}
		Expect(gorp.NewCreate[v0.Key, v0.ICD]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.ICD]{
			DB: db, Migrations: []gorp.Migration{v1.Migration},
		}))
		var got v1.ICD
		Expect(gorp.NewRetrieve[v1.Key, v1.ICD]().
			Where(gorp.MatchKeys[v1.Key, v1.ICD](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.ICD{
			Key:  old.Key,
			Name: "engine",
			Messages: []v1.Message{
				{Key: old.Messages[0].Key, Name: "status", Payload: pld},
			},
		}))
	})
})

var _ = Describe("MigrateICD", func() {
	It("Should keep the key, the name, and each message", func(ctx SpecContext) {
		old := v0.ICD{
			Key:      uuid.New(),
			Name:     "engine",
			Messages: []v0.Message{{Key: uuid.New(), Name: "status"}},
		}
		Expect(MustSucceed(v1.MigrateICD(ctx, old))).To(Equal(v1.ICD{
			Key:      old.Key,
			Name:     "engine",
			Messages: []v1.Message{{Key: old.Messages[0].Key, Name: "status"}},
		}))
	})
})

var _ = Describe("MigrateMessage", func() {
	It("Should keep each part and give no cycle time", func(ctx SpecContext) {
		old := v0.Message{
			Key:  uuid.New(),
			Name: "status",
			Payload: v0.Payload{Variant: v0.TextPayload{
				Delimiter: ",",
				Fields:    []v0.TextField{},
			}},
		}
		Expect(MustSucceed(v1.MigrateMessage(ctx, old))).To(Equal(v1.Message{
			Key:     old.Key,
			Name:    "status",
			Payload: old.Payload,
		}))
	})
})
