// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// dirHolder reads some channels and writes others, as a schematic does through the
// state and command channels of its nodes.
type dirHolder struct {
	Key    string
	Reads  []channel.Key
	Writes []channel.Key
}

func (h dirHolder) GorpKey() string { return h.Key }

func (h dirHolder) EncodeOrc(w *orc.Writer) error {
	w.String(h.Key)
	for _, keys := range [2][]channel.Key{h.Reads, h.Writes} {
		w.Uint32(uint32(len(keys)))
		for _, k := range keys {
			w.Uint32(uint32(k))
		}
	}
	return nil
}

func (h *dirHolder) DecodeOrc(r *orc.Reader) (err error) {
	if h.Key, err = r.String(); err != nil {
		return err
	}
	for _, keys := range [2]*[]channel.Key{&h.Reads, &h.Writes} {
		n, err := r.Uint32()
		if err != nil {
			return err
		}
		*keys = make([]channel.Key, n)
		for i := range *keys {
			k, err := r.Uint32()
			if err != nil {
				return err
			}
			(*keys)[i] = channel.Key(k)
		}
	}
	return nil
}

var _ = Describe("PublishReferenceChanges and use directions", Ordered, func() {
	var holders *gorp.Table[string, dirHolder]
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		holders = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, dirHolder]{
			DB:   db,
			Name: "signals_test_dir_holder",
			References: []gorp.Reference[string, dirHolder]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, dirHolder, channel.Key]{
						Name:     "reads",
						Table:    "channel",
						OnDelete: gorp.Restrict,
						Kind:     gorp.Use,
						Tag:      channel.ReadTag,
						Targets:  func(h *dirHolder) []channel.Key { return h.Reads },
					},
				),
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, dirHolder, channel.Key]{
						Name:     "writes",
						Table:    "channel",
						OnDelete: gorp.Restrict,
						Kind:     gorp.Use,
						Tag:      channel.WriteTag,
						Targets:  func(h *dirHolder) []channel.Key { return h.Writes },
					},
				),
			},
		}))
		DeferCleanup(func() { Expect(holders.Close()).To(Succeed()) })
	})

	It("Should name a channel whose uses gain a direction", func(ctx SpecContext) {
		holderOf := func(t gorp.Target) resource.ID {
			if t.Table == "signals_test_dir_holder" {
				return resource.ID{Type: resource.TypeSchematic, Key: t.Key}
			}
			return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
		}
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(Succeed())
		MustOpen(sigs.Start(ctx))
		ch := retrieveChannel(ctx, "sy_reference_change")
		record := sigs.Records()[ch.Key()]
		responses := openStreamer(ctx, ch.Key())
		changes := func() []string {
			GinkgoHelper()
			samples := receiveSamples(responses)
			targets := make([]string, len(samples))
			for i, s := range samples {
				targets[i] = MustSucceed(record.Of(s)).String()
			}
			return targets
		}
		createChannel := func() channel.Key {
			GinkgoHelper()
			c := channel.Channel{
				Name:     nextName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			return c.Key()
		}
		usesOf := func(k channel.Key) []channel.Use {
			GinkgoHelper()
			var uses [][]channel.Use
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				uses, err = channelSvc.Uses(ctx, snap, []channel.Key{k})
				return err
			})).To(Succeed())
			return uses[0]
		}
		channelID := func(k channel.Key) string {
			return resource.ID{
				Type: resource.TypeChannel,
				Key:  gorp.FormatKey(k),
			}.String()
		}
		a, b := createChannel(), createChannel()
		h := dirHolder{Key: nextName(), Reads: []channel.Key{a}}
		Expect(holders.NewCreate().Entry(&h).Exec(ctx, db)).To(Succeed())
		Expect(changes()).To(ConsistOf(channelID(a)))
		holder := gorp.Target{Table: "signals_test_dir_holder", Key: h.Key}
		Expect(usesOf(a)).To(Equal([]channel.Use{
			{Holder: holder, Direction: channel.DirectionRead},
		}))
		// The holder now writes the channel it reads, as a schematic does when a button
		// commands the channel that a value node shows.
		h.Writes = []channel.Key{a}
		Expect(holders.NewCreate().Entry(&h).Exec(ctx, db)).To(Succeed())
		Expect(usesOf(a)).To(Equal([]channel.Use{
			{Holder: holder, Direction: channel.DirectionRead},
			{Holder: holder, Direction: channel.DirectionWrite},
		}))
		// A later commit, so that a missing frame shows as the next frame.
		other := dirHolder{Key: nextName(), Reads: []channel.Key{b}}
		Expect(holders.NewCreate().Entry(&other).Exec(ctx, db)).To(Succeed())
		Expect(changes()).To(ConsistOf(channelID(a)))
	})
	It("Should carry holder keys with colons, digits, and any bytes", func(
		ctx SpecContext,
	) {
		holderOf := func(t gorp.Target) resource.ID {
			if t.Table == "signals_test_dir_holder" {
				return resource.ID{Type: resource.TypeSchematic, Key: t.Key}
			}
			return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
		}
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(Succeed())
		MustOpen(sigs.Start(ctx))
		ch := retrieveChannel(ctx, "sy_reference_change")
		record := sigs.Records()[ch.Key()]
		responses := openStreamer(ctx, ch.Key())
		c := channel.Channel{Name: nextName(), DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		keys := []string{
			"12:schematic:7", "", "ключ✓\x00\n", strings.Repeat("9:", 5000), ":",
		}
		hs := make([]dirHolder, len(keys))
		want := make([]resource.ID, len(keys))
		for i, k := range keys {
			hs[i] = dirHolder{Key: k, Reads: []channel.Key{c.Key()}}
			want[i] = resource.ID{Type: resource.TypeSchematic, Key: k}
		}
		Expect(holders.NewCreate().Entries(&hs).Exec(ctx, db)).To(Succeed())
		samples := receiveSamples(responses)
		Expect(samples).To(HaveLen(1))
		Expect(record.Of(samples[0])).To(Equal(c.ResourceID()))
		of, by, sent, err := record.Changed(samples[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(of).To(Equal(c.ResourceID()))
		Expect(by).To(ConsistOf(want))
		Expect(string(sent)).To(Equal(c.ResourceID().String()))
		malformed := "[signals] - malformed reference change sample"
		_, _, _, err = record.Changed([]byte("10:channel:428:nocolon"))
		Expect(err).To(MatchError(malformed))
		Expect(
			record.Of([]byte("3:abc")),
		).Error().
			To(MatchError(ContainSubstring("abc")))
		Expect(record.Of([]byte("99:channel:1"))).Error().To(MatchError(malformed))
	})
})
