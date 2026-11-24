// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Upgrade", func() {
	It("Should key ICDs by the type name that v0 used", func() {
		Expect(types.Name[icd.ICD]()).To(Equal(types.Name[v0.ICD]()))
	})

	It("Should read a stored v0 ICD with no framing", func(ctx SpecContext) {
		upgradeDB := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(upgradeDB.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.ICD]{DB: upgradeDB}))
		field := v0.BinaryField{
			Key:             uuid.New(),
			Name:            "speed",
			Scale:           1,
			StartBit:        4,
			BitLength:       16,
			ByteOrder:       v0.ByteOrderBigEndian,
			Kind:            payload.KindFloat,
			MultiplexValues: []int32{2},
		}
		old := v0.ICD{
			Key:  uuid.New(),
			Name: "engine",
			Messages: []v0.Message{{
				Key:  uuid.New(),
				Name: "status",
				Payload: v0.Payload{Variant: v0.BinaryPayload{
					Length: new(uint16(4)),
					Fields: []v0.BinaryField{field},
				}},
			}},
		}
		Expect(gorp.NewCreate[v0.Key, v0.ICD]().Entry(&old).Exec(ctx, upgradeDB)).
			To(Succeed())
		upgradeSvc := MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:   upgradeDB,
			ImEx: imex.NewService(),
		}))
		var got icd.ICD
		Expect(upgradeSvc.NewRetrieve().
			Where(icd.MatchKeys(old.Key)).
			Entry(&got).
			Exec(ctx, upgradeDB)).To(Succeed())
		Expect(got).To(Equal(icd.ICD{
			Key:  old.Key,
			Name: "engine",
			Messages: []icd.Message{{
				Key:  old.Messages[0].Key,
				Name: "status",
				Payload: icd.Payload{Variant: icd.BinaryPayload{
					Length: new(uint16(4)),
					Fields: []icd.BinaryField{{Variant: icd.FloatBinaryField{
						BinaryBaseField: icd.BinaryBaseField{
							BaseField:       field.BaseField,
							StartBit:        4,
							BitLength:       16,
							ByteOrder:       icd.ByteOrderBigEndian,
							MultiplexValues: []int32{2},
						},
					}}},
				}},
			}},
		}))
	})
})
