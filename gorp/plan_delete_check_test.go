// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// pdCopy keeps a copy of the channel it names when that channel is deleted.
type pdCopy struct {
	Key     string
	Channel uint32
}

func (c pdCopy) GorpKey() string { return c.Key }

var _ = Describe("PlanDelete checks", func() {
	It("Should list a holder that inlines a target among the detaches", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		channels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, pdChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		copies := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdCopy]{
			DB:   db,
			Name: "copy",
			References: []gorp.Reference[string, pdCopy]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, pdCopy, uint32]{
					Name:     "channel",
					Table:    "channel",
					OnDelete: gorp.Inline,
					Kind:     gorp.Use,
					Targets: func(c *pdCopy) []uint32 {
						if c.Channel == 0 {
							return nil
						}
						return []uint32{c.Channel}
					},
					Inline: func(context.Context, gorp.Tx, *pdCopy, uint32) error {
						return nil
					},
				}),
			},
		}))
		Expect(channels.NewCreate().Entry(&pdChannel{Key: 1}).Exec(ctx, db)).
			To(Succeed())
		Expect(copies.NewCreate().Entry(&pdCopy{Key: "c", Channel: 1}).Exec(ctx, db)).
			To(Succeed())
		effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
		Expect(effects.Detaches).To(Equal([]gorp.DeleteEffect{{
			Dependent: gorp.Dependent{
				Holder:    gorp.Target{Table: "copy", Key: "c"},
				Reference: "channel",
				Kind:      gorp.Use,
			},
			Target: gorp.NewTarget("channel", uint32(1)),
		}}))
		Expect(effects.Cascades).To(BeEmpty())
		Expect(effects.Restricts).To(BeEmpty())
		Expect(effects.Weak).To(BeEmpty())
	})
})
