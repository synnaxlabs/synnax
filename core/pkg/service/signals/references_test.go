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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// refHolder uses channels, sits in labels, and places its output on a channel, as a
// calculation does.
type refHolder struct {
	Key      string
	Note     string
	Channels []channel.Key
	Labels   []uuid.UUID
	Output   channel.Key
}

func (h refHolder) GorpKey() string { return h.Key }

func (h refHolder) EncodeOrc(w *orc.Writer) error {
	w.String(h.Key)
	w.String(h.Note)
	w.Uint32(uint32(len(h.Channels)))
	for _, k := range h.Channels {
		w.Uint32(uint32(k))
	}
	w.Uint32(uint32(len(h.Labels)))
	for _, l := range h.Labels {
		w.Write(l[:])
	}
	w.Uint32(uint32(h.Output))
	return nil
}

func (h *refHolder) DecodeOrc(r *orc.Reader) (err error) {
	if h.Key, err = r.String(); err != nil {
		return err
	}
	if h.Note, err = r.String(); err != nil {
		return err
	}
	n, err := r.Uint32()
	if err != nil {
		return err
	}
	h.Channels = make([]channel.Key, n)
	for i := range h.Channels {
		k, err := r.Uint32()
		if err != nil {
			return err
		}
		h.Channels[i] = channel.Key(k)
	}
	if n, err = r.Uint32(); err != nil {
		return err
	}
	h.Labels = make([]uuid.UUID, n)
	for i := range h.Labels {
		if _, err = r.Read(h.Labels[i][:]); err != nil {
			return err
		}
	}
	out, err := r.Uint32()
	h.Output = channel.Key(out)
	return err
}

func refHolderReferences() []gorp.Reference[string, refHolder] {
	return []gorp.Reference[string, refHolder]{
		gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refHolder, channel.Key]{
			Name:     "channels",
			Table:    "channel",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets:  func(h *refHolder) []channel.Key { return h.Channels },
		}),
		gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refHolder, uuid.UUID]{
			Name:     "labels",
			Table:    "label",
			OnDelete: gorp.Cascade,
			Kind:     gorp.Place,
			Targets:  func(h *refHolder) []uuid.UUID { return h.Labels },
		}),
		gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refHolder, channel.Key]{
			Name:     "output",
			Table:    "channel",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Place,
			Targets: func(h *refHolder) []channel.Key {
				if h.Output == 0 {
					return nil
				}
				return []channel.Key{h.Output}
			},
		}),
	}
}

// refSample is one reference change: a target and the holders that changed it.
type refSample struct {
	target  string
	holders []string
}

// holderOf stands each test holder for a schematic, so that its table names a type.
func holderOf(holder gorp.Target) resource.ID {
	if holder.Table == "signals_test_holder" {
		return resource.ID{Type: resource.TypeSchematic, Key: holder.Key}
	}
	return resource.ID{Type: resource.Type(holder.Table), Key: holder.Key}
}

var _ = Describe("PublishReferenceChanges", Ordered, func() {
	var holders *gorp.Table[string, refHolder]
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		holders = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, refHolder]{
			DB:         db,
			Name:       "signals_test_holder",
			References: refHolderReferences(),
		}))
		DeferCleanup(func() { Expect(holders.Close()).To(Succeed()) })
	})

	var (
		sigs    *signals.Provider
		receive func() []refSample
		changes func() []string
	)
	BeforeEach(func(ctx SpecContext) {
		sigs = MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(Succeed())
		MustOpen(sigs.Start(ctx))
		ch := retrieveChannel(ctx, "sy_reference_change")
		Expect(ch.DataType).To(Equal(telem.StringT))
		hook := make(chan []signals.ReferenceChange, 16)
		DeferCleanup(sigs.ReferenceChanges().OnChange(
			func(_ context.Context, changes []signals.ReferenceChange) {
				select {
				case hook <- changes:
				default:
				}
			},
		))
		// The streamer reads the channel as a task does.
		responses := openStreamer(ctx, ch.Key())
		receive = func() []refSample {
			GinkgoHelper()
			raw := receiveSamples(responses)
			targets := make([]string, len(raw))
			for i, r := range raw {
				targets[i] = string(r)
			}
			var changes []signals.ReferenceChange
			Eventually(hook).Should(Receive(&changes))
			samples := make([]refSample, len(changes))
			for i, c := range changes {
				samples[i].target = c.Target.String()
				for _, h := range c.Holders {
					samples[i].holders = append(samples[i].holders, h.String())
				}
			}
			// The channel holds each target bare, in the order of the hook.
			Expect(targets).To(Equal(lo.Map(samples, func(s refSample, _ int) string {
				return s.target
			})))
			return samples
		}
		changes = func() []string {
			GinkgoHelper()
			samples := receive()
			targets := make([]string, len(samples))
			for i, s := range samples {
				targets[i] = s.target
			}
			return targets
		}
	})

	createChannel := func(ctx SpecContext) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{Name: nextName(), DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch.Key()
	}
	createLabel := func(ctx SpecContext) uuid.UUID {
		GinkgoHelper()
		l := label.Label{Key: uuid.New(), Name: nextName()}
		Expect(gorp.NewCreate[uuid.UUID, label.Label]().Entry(&l).Exec(ctx, db)).
			To(Succeed())
		return l.Key
	}
	set := func(ctx SpecContext, hs ...refHolder) {
		GinkgoHelper()
		Expect(holders.NewCreate().Entries(&hs).Exec(ctx, db)).To(Succeed())
	}
	channelID := func(k channel.Key) string {
		return resource.ID{Type: resource.TypeChannel, Key: gorp.FormatKey(k)}.String()
	}
	labelID := func(k uuid.UUID) string {
		return resource.ID{Type: resource.TypeLabel, Key: gorp.FormatKey(k)}.String()
	}

	It("Should write the target of each added link once", func(ctx SpecContext) {
		a, b, out := createChannel(ctx), createChannel(ctx), createChannel(ctx)
		l := createLabel(ctx)
		set(ctx, refHolder{
			Key:      nextName(),
			Channels: []channel.Key{a, b},
			Labels:   []uuid.UUID{l},
			Output:   out,
		})
		Expect(changes()).To(ConsistOf(
			channelID(a),
			channelID(b),
			channelID(out),
			labelID(l),
		))
	})

	It("Should write nothing for a write that keeps its targets", func(
		ctx SpecContext,
	) {
		a, b := createChannel(ctx), createChannel(ctx)
		h := refHolder{Key: nextName(), Channels: []channel.Key{a}}
		set(ctx, h)
		Expect(changes()).To(ConsistOf(channelID(a)))
		h.Note = "kept"
		set(ctx, h)
		set(ctx, refHolder{Key: nextName(), Channels: []channel.Key{b}})
		Expect(changes()).To(ConsistOf(channelID(b)))
	})

	It("Should write the removed targets of a delete", func(ctx SpecContext) {
		a := createChannel(ctx)
		h := refHolder{Key: nextName(), Channels: []channel.Key{a}}
		set(ctx, h)
		Expect(changes()).To(ConsistOf(channelID(a)))
		Expect(holders.NewDelete().
			Where(gorp.MatchKeys[string, refHolder](h.Key)).
			Exec(ctx, db)).To(Succeed())
		Expect(changes()).To(ConsistOf(channelID(a)))
	})

	It("Should write the targets of the holders that a cascade deletes", func(
		ctx SpecContext,
	) {
		a, l := createChannel(ctx), createLabel(ctx)
		set(ctx, refHolder{
			Key:      nextName(),
			Channels: []channel.Key{a},
			Labels:   []uuid.UUID{l},
		})
		Expect(changes()).To(ConsistOf(channelID(a), labelID(l)))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return labelSvc.NewWriter(tx).Delete(ctx, l)
		})).To(Succeed())
		Expect(changes()).To(ConsistOf(channelID(a), labelID(l)))
	})

	It("Should write a target that two holders of one commit add once", func(
		ctx SpecContext,
	) {
		a := createChannel(ctx)
		first, second := nextName(), nextName()
		set(
			ctx,
			refHolder{Key: first, Channels: []channel.Key{a}},
			refHolder{Key: second, Channels: []channel.Key{a}},
		)
		holderID := func(key string) string {
			holder := gorp.Target{Table: "signals_test_holder", Key: key}
			return holderOf(holder).String()
		}
		Expect(receive()).To(Equal([]refSample{{
			target:  channelID(a),
			holders: []string{holderID(first), holderID(second)},
		}}))
	})

	It("Should write the old and the new target of a move", func(ctx SpecContext) {
		from, to := createLabel(ctx), createLabel(ctx)
		h := refHolder{Key: nextName(), Labels: []uuid.UUID{from}}
		set(ctx, h)
		Expect(changes()).To(ConsistOf(labelID(from)))
		h.Labels = []uuid.UUID{to}
		set(ctx, h)
		Expect(changes()).To(ConsistOf(labelID(from), labelID(to)))
	})

	It("Should write the channel of a calculation's output", func(ctx SpecContext) {
		out := createChannel(ctx)
		set(ctx, refHolder{Key: nextName(), Output: out})
		Expect(changes()).To(ConsistOf(channelID(out)))
	})

	It("Should write only the bare target of a change to the channel", func(
		ctx SpecContext,
	) {
		a := createChannel(ctx)
		set(ctx, refHolder{Key: nextName(), Channels: []channel.Key{a}})
		samples := receive()
		Expect(samples).To(HaveLen(1))
		Expect(samples[0].target).To(Equal(channelID(a)))
		Expect(samples[0].holders).To(HaveLen(1))
		ch := retrieveChannel(ctx, "sy_reference_change")
		Expect(sigs.ReferenceChanges().Key).To(Equal(ch.Key()))
		Expect(sigs.Records()).ToNot(HaveKey(ch.Key()))
	})
})

var _ = Describe("PublishReferenceChanges with an unknown target table", func() {
	It("Should fail when a referenced table names no resource type", func(
		ctx SpecContext,
	) {
		other := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(orc.Codec)))
		StartRun(other.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, testUUIDEntry]{
			DB:   other,
			Name: "signals_test_target",
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refHolder]{
			DB:   other,
			Name: "signals_test_holder",
			References: []gorp.Reference[string, refHolder]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, refHolder, channel.Key]{
						Name:     "channels",
						Table:    "signals_test_target",
						OnDelete: gorp.Restrict,
						Kind:     gorp.Use,
						Targets:  func(h *refHolder) []channel.Key { return h.Channels },
					},
				),
			},
		}))
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      other,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(MatchError(
			`[signals] - referenced table "signals_test_target" names no resource type`,
		))
	})
})

var _ = Describe("PublishReferenceChanges beside a table", func() {
	It("Should write the set and the reference change of one commit in one frame", func(
		ctx SpecContext,
	) {
		holders := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[string, refHolder]{
			DB:         db,
			Name:       "signals_test_frame_holder",
			References: refHolderReferences(),
		}))
		DeferCleanup(func() { Expect(holders.Close()).To(Succeed()) })
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		setName := nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[string, refHolder]{
			Changes:    holders.Changes,
			Resource:   testResource[string],
			SetName:    setName,
			DeleteName: nextName(),
		})).To(Succeed())
		Expect(sigs.PublishReferenceChanges(ctx, holderOf)).To(Succeed())
		MustOpen(sigs.Start(ctx))
		set := retrieveChannel(ctx, setName)
		ref := retrieveChannel(ctx, "sy_reference_change")
		responses := openStreamer(ctx, set.Key(), ref.Key())
		c := channel.Channel{Name: nextName(), DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		h := refHolder{Key: nextName(), Channels: []channel.Key{c.Key()}}
		Expect(holders.NewCreate().Entry(&h).Exec(ctx, db)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(func() bool {
			select {
			case res = <-responses.Outlet():
				return !res.Frame.Empty()
			default:
				return false
			}
		}).Should(BeTrue())
		Expect(res.Frame.KeysSlice()).To(ConsistOf(set.Key(), ref.Key()))
	})
})
