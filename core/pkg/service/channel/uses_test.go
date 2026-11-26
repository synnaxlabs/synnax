// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type taggedHolder struct {
	Key     string
	Channel channel.Key
}

func (h taggedHolder) GorpKey() string { return h.Key }

var _ = Describe("Uses", func() {
	// usesWithTag returns the uses of a channel that one record holds through a
	// reference with the given tag.
	usesWithTag := func(ctx SpecContext, tag uint8) ([][]channel.Use, error) {
		// The holder has no generated codec, so the store encodes with msgpack.
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		cfg, _ := serviceConfig(ctx, testutil.NewNode(ctx), db)
		freshSvc := MustOpen(channel.OpenService(ctx, cfg))
		holders := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, taggedHolder]{
			DB:   db,
			Name: "tagged_holder",
			References: []gorp.Reference[string, taggedHolder]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, taggedHolder, channel.Key]{
						Name:     "channel",
						Table:    "channel",
						OnDelete: gorp.Weak,
						Kind:     gorp.Use,
						Tag:      tag,
						Targets: func(h *taggedHolder) []channel.Key {
							return []channel.Key{h.Channel}
						},
						Detach: func(h *taggedHolder, _ channel.Key) { h.Channel = 0 },
					},
				),
			},
		}))
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return freshSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(holders.NewCreate().
			Entry(&taggedHolder{Key: "h", Channel: ch.Key()}).
			Exec(ctx, db)).To(Succeed())
		return freshSvc.Uses(ctx, db, []channel.Key{ch.Key()})
	}

	It("Should leave out a reference with no direction", func(ctx SpecContext) {
		Expect(MustSucceed(usesWithTag(ctx, 0))).To(Equal([][]channel.Use{{}}))
	})

	It("Should fail on a reference tag that names no direction", func(ctx SpecContext) {
		Expect(usesWithTag(ctx, 3)).Error().To(MatchError(
			ContainSubstring("reference channel of tagged_holder has unknown tag 3"),
		))
	})
})
