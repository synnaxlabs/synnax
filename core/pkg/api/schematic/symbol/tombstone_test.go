// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// displayHolder holds a channel through a weak link, as a schematic symbol does.
type displayHolder struct {
	Key     string
	Channel channel.Key
}

func (h displayHolder) GorpKey() string { return h.Key }

func (h displayHolder) EncodeOrc(w *orc.Writer) error {
	w.String(h.Key)
	w.Uint32(uint32(h.Channel))
	return nil
}

func (h *displayHolder) DecodeOrc(r *orc.Reader) (err error) {
	if h.Key, err = r.String(); err != nil {
		return err
	}
	k, err := r.Uint32()
	h.Channel = channel.Key(k)
	return err
}

var _ = Describe("DeleteGroup of a channel folder", func() {
	It("Should remove the tombstones of the folder first", func(ctx SpecContext) {
		holders := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, displayHolder]{
			DB:   db,
			Name: "symbol_test_display",
			References: []gorp.Reference[string, displayHolder]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, displayHolder, channel.Key]{
						Name:     "channel",
						Table:    "channel",
						OnDelete: gorp.Weak,
						Kind:     gorp.Use,
						Tag:      1,
						Targets: func(h *displayHolder) []channel.Key {
							return []channel.Key{h.Channel}
						},
						Detach: func(h *displayHolder, _ channel.Key) { h.Channel = 0 },
					},
				),
			},
		}))
		folder := group.Group{Name: "measurements", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &folder)
		})).To(Succeed())
		ch := channel.Channel{
			Name:     "tombstoned",
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &folder.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(holders.NewCreate().
			Entry(&displayHolder{Key: "display", Channel: ch.Key()}).
			Exec(ctx, db)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		u := createUser(ctx)
		grantDeleteOn(ctx, u.ResourceID(), folder.ResourceID())
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, folder.ResourceID())
		MustSucceed(deleteGroup(
			AuthedCtx(ctx, u),
			apisymbol.DeleteGroupRequest{Key: folder.Key},
		))
		Expect(groupSvc.NewRetrieve().
			Where(group.MatchKeys(folder.Key)).
			Exists(ctx, db)).To(BeFalse())
	})
})
