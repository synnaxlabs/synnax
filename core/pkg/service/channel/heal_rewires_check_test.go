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

var _ = Describe("Heal rewires check", func() {
	It("Should keep a tombstone of a channel that holds rewires when nothing holds it",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			cfg, _ := serviceConfig(ctx, testutil.NewNode(ctx), db)
			cfg.Retargets = func(gorp.Dependent) bool { return false }
			svc := MustOpen(channel.OpenService(ctx, cfg))
			keepers := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, healKeeper]{
					DB:   db,
					Name: "keeper",
					References: weakChannelReference(func(h *healKeeper) *channel.Key {
						return &h.Channel
					}),
				}),
			)
			Expect(svc.Heal(ctx)).To(Succeed())
			write := func(f func(w channel.Writer) error) {
				GinkgoHelper()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return f(svc.NewWriter(tx))
				})).To(Succeed())
			}
			name := UniqueChannelName()
			dead := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
			write(func(w channel.Writer) error { return w.Create(ctx, &dead) })
			Expect(keepers.NewCreate().
				Entry(&healKeeper{Key: name, Channel: dead.Key()}).
				Exec(ctx, db)).To(Succeed())
			write(
				func(w channel.Writer) error { return w.Delete(ctx, dead.Key(), false) },
			)
			// The new channel takes the tombstone's list, and the keeper stays on the
			// dead
			// key, so nothing holds the new channel.
			ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
			write(func(w channel.Writer) error { return w.Create(ctx, &ch) })
			var stored channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Rewires).ToNot(BeEmpty())
			write(
				func(w channel.Writer) error { return w.Delete(ctx, ch.Key(), false) },
			)
			var ts []channel.Tombstone
			Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
				Entries(&ts).
				Exec(ctx, db)).To(Succeed())
			Expect(ts).To(ContainElement(SatisfyAll(
				HaveField("Key", ch.Key()),
				HaveField("Rewires", Not(BeEmpty())),
			)))
		},
	)
})
