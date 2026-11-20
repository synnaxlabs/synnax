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
	"bytes"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// decodeHookCodec runs its hook once, before the first decode of a refDevice after the
// hook is stored.
type decodeHookCodec struct {
	encoding.Codec
	hook atomic.Pointer[func()]
}

func (c *decodeHookCodec) Decode(data []byte, value any) error {
	if _, ok := value.(*refDevice); ok {
		if h := c.hook.Swap(nil); h != nil {
			(*h)()
		}
	}
	return c.Codec.Decode(data, value)
}

// refDeviceRack writes the table of refDevice as another entry type.
type refDeviceRack struct {
	Key  string
	Rack uint32
}

func (d refDeviceRack) GorpKey() string      { return d.Key }
func (refDeviceRack) CustomTypeName() string { return "refDevice" }

var _ = Describe("Reused links", func() {
	It(
		"Should record the links of the written entry when the index loads mid-write",
		func(ctx SpecContext) {
			kvs := &blockingDB{DB: memkv.New(), release: make(chan struct{})}
			pre := gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
			Expect(gorp.NewCreate[uint32, refRack]().
				Entries(&[]refRack{{Key: 1}, {Key: 2}}).
				Exec(ctx, pre)).To(Succeed())
			Expect(gorp.NewCreate[string, refDevice]().
				Entry(&refDevice{Key: "d", Rack: 1}).
				Exec(ctx, pre)).To(Succeed())
			codec := &decodeHookCodec{Codec: msgpack.Codec}
			db := DeferClose(gorp.Wrap(kvs, gorp.WithCodec(codec)))
			StartRun(db.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   db,
				Name: "label",
			}))
			refs := newDeviceRefs()
			devices := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         db,
				Name:       "device",
				References: refs.all(),
			}))
			tx := DeferClose(db.OpenTx())
			// A raw write that moves d to rack 2, which the change list does not see.
			keys := key.NewCodec[string](key.Prefix[refDevice]())
			raw := MustSucceed(msgpack.Codec.Encode(refDevice{Key: "d", Rack: 2}))
			Expect(tx.Set(ctx, bytes.Clone(keys.Encode("d")), raw)).To(Succeed())
			// The index finishes its load while the write reads the stored record.
			hook := func() {
				close(kvs.release)
				Expect(refs.rack.Get(ctx, nil, 1)).To(ConsistOf("d"))
				Expect(refs.parent.Get(ctx, nil, "p")).To(BeEmpty())
				Expect(refs.labels.Get(ctx, nil, "l")).To(BeEmpty())
			}
			codec.hook.Store(&hook)
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "d", Rack: 1}).
				Exec(ctx, tx)).To(Succeed())
			Expect(codec.hook.Load()).To(BeNil())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
				HaveField("Kind", gorp.ChangeUpdate),
				HaveField("Added", []gorp.Link{rackLink(1)}),
				HaveField("Removed", []gorp.Link{rackLink(2)}),
				HaveField("Links", []gorp.Link{rackLink(1)}),
			)))
		},
	)

	It(
		"Should record the links of the written entry after a write as another type",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
				DB:   db,
				Name: "rack",
			}))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
				DB:   db,
				Name: "label",
			}))
			devices := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refDevice]{
				DB:         db,
				Name:       "device",
				References: newDeviceRefs().all(),
			}))
			Expect(racks.NewCreate().
				Entries(&[]refRack{{Key: 1}, {Key: 2}}).
				Exec(ctx, db)).To(Succeed())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "d", Rack: 1}).
				Exec(ctx, db)).To(Succeed())
			tx := DeferClose(db.OpenTx())
			Expect(gorp.NewCreate[string, refDeviceRack]().
				Entry(&refDeviceRack{Key: "d", Rack: 2}).
				Exec(ctx, tx)).To(Succeed())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "d", Rack: 1}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
				HaveField("Kind", gorp.ChangeUpdate),
				HaveField("Links", []gorp.Link{rackLink(1)}),
			)))
		},
	)
})
