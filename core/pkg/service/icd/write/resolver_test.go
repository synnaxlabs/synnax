// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write_test

import (
	"context"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const modeKey channel.Key = 3

// writeAt returns the write of wm of the ICD with key, at targets.i of a config.
func writeAt(i int, key icd.Key, wm icd.WriteMessage) write.Write {
	return write.Write{
		Path:    []string{"targets", strconv.Itoa(i)},
		ICD:     key,
		Message: wm,
	}
}

var _ = Describe("Resolver", func() {
	var (
		resolver write.Resolver
		// held holds the channels that the channel lookup finds.
		held map[channel.Key]icdv0.Channel
		// asked holds the keys of each channel lookup.
		asked [][]channel.Key
	)
	BeforeEach(func() {
		held = map[channel.Key]icdv0.Channel{
			valueKey: {Name: "value", DataType: telem.Float64T},
		}
		asked = nil
		resolver = write.NewResolver(icdSvc, func(
			_ context.Context,
			_ gorp.Snapshot,
			keys []channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			asked = append(asked, keys)
			return held, nil
		})
	})

	Describe("Check", func() {
		given := plant
		given.Key = uuid.New()
		// check checks writes with given as the only ICD that the caller holds.
		check := func(ctx context.Context, writes ...write.Write) error {
			GinkgoHelper()
			return resolver.Check(
				ctx,
				db,
				writes,
				map[icd.Key]icd.ICD{given.Key: given},
			)
		}

		It("Should accept writes whose messages build", func(ctx SpecContext) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(
					1,
					given.Key,
					writeOf(tagged, channelOf(id.Base().Key, modeKey, nil)),
				),
			)).To(Succeed())
		})

		It("Should return the error of the first write that does not build, at the "+
			"path of its message", func(ctx SpecContext) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(1, given.Key, writeOf(tagged)),
				writeAt(2, given.Key, writeOf(binary)),
			)).To(MatchError("targets.1.message: message Tagged binds no channels"))
		})

		It("Should take the data type of each channel that the lookup finds", func(
			ctx SpecContext,
		) {
			w := writeAt(0, given.Key, writeOf(
				setpoint, bind(temp), channelOf(mode.Base().Key, modeKey, nil),
			))
			Expect(check(ctx, w)).To(Succeed())
			held[modeKey] = icdv0.Channel{Name: "mode", DataType: telem.Float64T}
			Expect(check(ctx, w)).To(MatchError(
				"targets.0.message: message Setpoint: field mode: cannot write float64 " +
					"channels to string fields: validation error",
			))
		})

		It("Should retrieve an ICD that the given ICDs lack", func(ctx SpecContext) {
			stored := icd.ICD{Name: "Stored", Messages: []icd.Message{tagged}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &stored)
			})).To(Succeed())
			Expect(check(ctx, writeAt(0, stored.Key, writeOf(setpoint, bind(temp))))).
				To(MatchError(
					"targets.0.message: message " + setpoint.Key.String() +
						" does not exist in ICD Stored",
				))
		})

		It("Should skip a write whose ICD does not exist", func(ctx SpecContext) {
			Expect(check(ctx, writeAt(0, uuid.New(), writeOf(tagged)))).To(Succeed())
		})

		It("Should look up each channel that a binding names once", func(
			ctx SpecContext,
		) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(1, given.Key, writeOf(
					setpoint,
					bind(temp),
					channelOf(mode.Base().Key, modeKey, nil),
					fixed(limit, icd.FloatWriteValue{Value: 2}),
				)),
				writeAt(2, uuid.New(), writeOf(setpoint, bind(temp))),
			)).To(Succeed())
			Expect(asked).To(HaveExactElements(ConsistOf(valueKey, modeKey)))
		})

		It("Should not look up channels when no binding names one", func(
			ctx SpecContext,
		) {
			Expect(check(ctx, writeAt(0, uuid.New(), writeOf(tagged)))).To(Succeed())
			Expect(asked).To(BeEmpty())
		})

		It("Should return the error of the channel lookup", func(ctx SpecContext) {
			resolver = write.NewResolver(icdSvc, func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]icdv0.Channel, error) {
				return nil, errors.New("no channels")
			})
			Expect(check(ctx, writeAt(0, given.Key, writeOf(setpoint, bind(temp))))).
				To(MatchError("no channels"))
		})
	})

	Describe("Hash", func() {
		var stored icd.ICD
		BeforeEach(func(ctx SpecContext) {
			stored = icd.ICD{Name: "Plant", Messages: []icd.Message{setpoint, tagged}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &stored)
			})).To(Succeed())
		})
		sum := func(ctx context.Context, writes ...write.Write) string {
			GinkgoHelper()
			h := content.NewHash()
			Expect(resolver.Hash(ctx, db, &h, writes)).To(Succeed())
			return h.Sum()
		}
		// setPointer sets the pointer of f, a field of m, to pointer.
		setPointer := func(
			ctx context.Context,
			m icd.Message,
			f icd.JSONField,
			pointer string,
		) {
			GinkgoHelper()
			path := []string{
				"messages", m.Key.String(),
				"payload", "fields", f.Base().Key.String(),
				"pointer",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Apply(ctx, stored.Key, icd.NewSetAction(
					icd.SetPayload{Path: path, Value: pointer},
				))
			})).To(Succeed())
		}

		It("Should give the same sum for the same writes", func(ctx SpecContext) {
			w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
			Expect(sum(ctx, w)).To(Equal(sum(ctx, w)))
		})

		It("Should change when the payload of a message that a write sends changes",
			func(ctx SpecContext) {
				w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
				before := sum(ctx, w)
				setPointer(ctx, setpoint, temp, "/temperature")
				Expect(sum(ctx, w)).ToNot(Equal(before))
			},
		)

		It("Should not change when a message that no write sends changes", func(
			ctx SpecContext,
		) {
			w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
			before := sum(ctx, w)
			setPointer(ctx, tagged, id, "/uuid")
			Expect(sum(ctx, w)).To(Equal(before))
		})

		It("Should mark a write of a message that the ICD lacks", func(
			ctx SpecContext,
		) {
			lacked := writeAt(0, stored.Key, writeOf(binary))
			Expect(sum(ctx, lacked)).ToNot(Equal(sum(ctx)))
		})

		It("Should hash a write of an ICD that does not exist as a message that is "+
			"absent", func(ctx SpecContext) {
			missing := writeAt(0, uuid.New(), writeOf(setpoint, bind(temp)))
			lacked := writeAt(0, stored.Key, writeOf(binary))
			Expect(sum(ctx, missing)).To(Equal(sum(ctx, lacked)))
		})
	})
})
