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
	"io"
	"iter"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	ontologycdc "github.com/synnaxlabs/synnax/pkg/distribution/ontology/signals"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

type changeService struct {
	observe.Observer[iter.Seq[ontology.Change]]
}

const changeType ontology.Type = "change"

func newChangeID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: changeType}
}

var _ ontology.Service = (*changeService)(nil)

func (s *changeService) Type() ontology.Type { return changeType }

func (s *changeService) Schema() zyn.Schema {
	return zyn.Object(map[string]zyn.Schema{"key": zyn.String()})
}

func (s *changeService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values([]ontology.Resource{}), xio.NopCloser, nil
}

func (s *changeService) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	return ontology.NewResource(
		s.Schema(),
		newChangeID(key),
		"",
		map[string]any{"key": key},
	), nil
}

var _ = Describe("Signals", Ordered, func() {
	var (
		builder *mock.Cluster
		ctx     = context.Background()
		dist    mock.Node
		svc     *changeService
	)
	BeforeAll(func() {
		builder = mock.NewCluster()
		dist = builder.Provision(ctx)
		svc = &changeService{Observer: observe.New[iter.Seq[ontology.Change]]()}
		dist.Ontology.RegisterService(svc)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
	})
	Describe("DecodeIDs", func() {
		It("Should decode a series of IDs", func() {
			encoded := ontologycdc.EncodeIDs([]ontology.ID{newChangeID("one"), newChangeID("two")})
			decoded := MustSucceed(ontologycdc.DecodeIDs(encoded))
			Expect(decoded).To(Equal([]ontology.ID{newChangeID("one"), newChangeID("two")}))
		})
	})
	Describe("Resource Changes", func() {
		It("Should correctly propagate resource changes to the ontology", func() {
			var resCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_ontology_resource_set").Entry(&resCh).Exec(ctx, nil)).To(Succeed())
			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: channel.Keys{resCh.Key()},
			}))
			requests, responses := confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(5 * time.Millisecond)
			closeStreamer := signal.NewHardShutdown(sCtx, cancel)
			key := "hello"
			svc.NotifyGenerator(ctx, func() iter.Seq[ontology.Change] {
				return slices.Values([]ontology.Change{
					{
						Variant: change.Set,
						Key:     newChangeID(key),
						Value: ontology.NewResource(
							svc.Schema(),
							newChangeID(key),
							"empty",
							map[string]any{"key": key},
						),
					},
				})
			})
			var res framer.StreamerResponse
			Eventually(responses.Outlet()).Should(Receive(&res))
			s := res.Frame.SeriesAt(0)
			Expect(s.Len()).To(Equal(int64(1)))
			for s := range s.Samples() {
				r := ontology.Resource{}
				Expect((&binary.JSONCodec{}).Decode(ctx, s, &r)).To(Succeed())
				Expect(r.ID).To(Equal(newChangeID(key)))
			}
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			Expect(closeStreamer.Close()).To(Succeed())
		})
		It("Should correctly propagate resource deletes to the ontology", func() {
			var resCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_ontology_resource_delete").Entry(&resCh).Exec(ctx, nil)).To(Succeed())
			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: channel.Keys{resCh.Key()},
			}))
			requests, responses := confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(5 * time.Millisecond)
			closeStreamer := signal.NewHardShutdown(sCtx, cancel)
			key := "hello"
			svc.NotifyGenerator(ctx, func() iter.Seq[ontology.Change] {
				return slices.Values([]ontology.Change{
					{
						Variant: change.Delete,
						Key:     newChangeID(key),
					},
				})
			})
			var res framer.StreamerResponse
			Eventually(responses.Outlet()).Should(Receive(&res))
			ids := MustSucceed(ontologycdc.DecodeIDs(res.Frame.SeriesAt(0).Data))
			// There's a condition here where we might receive the channel creation
			// signal, so we just do a length assertion.
			Expect(len(ids)).To(BeNumerically(">", 0))
			Expect(len(ids[0].Type)).To(BeNumerically(">", 0))
			Expect(len(ids[0].Key)).To(BeNumerically(">", 0))
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			Expect(closeStreamer.Close()).To(Succeed())
		})
	})
	It("Should correctly propagate relationship set to the ontology", func() {
		var resCh channel.Channel
		Expect(dist.Channel.NewRetrieve().WhereNames("sy_ontology_relationship_set").Entry(&resCh).Exec(ctx, nil)).To(Succeed())
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{resCh.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		time.Sleep(10 * time.Millisecond)
		closeStreamer := signal.NewHardShutdown(sCtx, cancel)
		defer func() {
			GinkgoRecover()
			Expect(closeStreamer.Close()).To(Succeed())
		}()

		w := dist.Ontology.NewWriter(nil)
		firstResource := newChangeID("abc")
		secondResource := newChangeID("def")
		Expect(w.DefineResource(ctx, firstResource)).To(Succeed())
		Expect(w.DefineResource(ctx, secondResource)).To(Succeed())
		Expect(w.DefineRelationship(ctx, firstResource, ontology.ParentOf, secondResource)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 10*time.Second).Should(Receive(&res))
		relationships := MustSucceed(ontologycdc.DecodeRelationships(res.Frame.SeriesAt(0).Data))
		// There's a condition here where we might receive the channel creation
		// signal, so we just do a length assertion.
		Expect(len(relationships)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].Type)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].From.Key)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].To.Key)).To(BeNumerically(">", 0))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
	})
	It("Should correctly propagate a relationship delete to the ontology", func() {
		var resCh channel.Channel
		By("Correctly creating the deletion channel.")
		Expect(dist.Channel.NewRetrieve().WhereNames("sy_ontology_relationship_delete").Entry(&resCh).Exec(ctx, nil)).To(Succeed())
		By("Opening a streamer on the deletion channel")
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{resCh.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		time.Sleep(5 * time.Millisecond)
		closeStreamer := signal.NewHardShutdown(sCtx, cancel)
		defer func() {
			GinkgoRecover()
			Expect(closeStreamer.Close()).To(Succeed())
		}()

		w := dist.Ontology.NewWriter(nil)
		firstResource := newChangeID("abc")
		secondResource := newChangeID("def")
		Expect(w.DefineResource(ctx, firstResource)).To(Succeed())
		Expect(w.DefineResource(ctx, secondResource)).To(Succeed())
		By("Creating the relationship")
		Expect(w.DefineRelationship(ctx, firstResource, ontology.ParentOf, secondResource)).To(Succeed())
		By("Deleting the relationship")
		Expect(w.DeleteRelationship(ctx, firstResource, ontology.ParentOf, secondResource)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet()).Should(Receive(&res))
		By("Decoding the relationships")
		relationships := MustSucceed(ontologycdc.DecodeRelationships(res.Frame.SeriesAt(0).Data))
		// There's a condition here where we might receive the channel creation
		// signal, so we just do a length assertion.
		Expect(len(relationships)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].Type)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].From.Key)).To(BeNumerically(">", 0))
		Expect(len(relationships[0].To.Key)).To(BeNumerically(">", 0))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
	})
})
