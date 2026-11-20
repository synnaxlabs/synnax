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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Reused links through another DB", func() {
	It(
		"Should record the links of an entry written through a transaction of another DB",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			// A second DB on the same store, with no tables of its own.
			other := gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
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
			tx := DeferClose(other.OpenTx())
			Expect(devices.NewCreate().
				Entry(&refDevice{Key: "d", Rack: 2}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.ChangesOf(tx)).To(HaveExactElements(SatisfyAll(
				HaveField("Kind", gorp.ChangeUpdate),
				HaveField("Added", []gorp.Link{rackLink(2)}),
				HaveField("Removed", []gorp.Link{rackLink(1)}),
				HaveField("Links", []gorp.Link{rackLink(2)}),
			)))
		},
	)
})
