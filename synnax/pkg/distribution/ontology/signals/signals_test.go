// Copyright 2023 Synnax Labs, Inc.
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	ontologycdc "github.com/synnaxlabs/synnax/pkg/distribution/ontology/signals"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"time"
)

type changeService struct {
	observe.Observer[iter.Nexter[schema.Change]]
}

const changeType ontology.Type = "change"

func newChangeID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: changeType}
}

var _ ontology.Service = (*changeService)(nil)

func (s *changeService) Schema() *ontology.Schema {
	return &ontology.Schema{
		Type: changeType,
		Fields: map[string]schema.Field{
			"key": {Type: schema.String},
		},
	}
}

func (s *changeService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser[ontology.Resource](iter.All[ontology.Resource](nil)), nil
}

func (s *changeService) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	e := schema.NewResource(s.Schema(), newChangeID(key), "empty")
	schema.Set(e, "key", key)
	return e, nil
}

var _ = Describe("Signals", Ordered, func() {
	var (
		builder *mock.Builder
		ctx     = context.Background()
		dist    distribution.Distribution
		svc     *changeService
	)
	BeforeAll(func() {
		builder = mock.NewBuilder()
		dist = builder.New(ctx)
		svc = &changeService{Observer: observe.New[iter.Nexter[schema.Change]]()}
		dist.Ontology.RegisterService(svc)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
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
				Start: telem.Now(),
				Keys:  channel.Keys{resCh.Key()},
			}))
			requests, responses := confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(5 * time.Millisecond)
			closeStreamer := signal.NewShutdown(sCtx, cancel)
			key := "hello"
			svc.NotifyGenerator(ctx, func() iter.Nexter[schema.Change] {
				return iter.All[schema.Change]([]schema.Change{
					{
						Variant: change.Set,
						Key:     newChangeID(key),
						Value:   schema.NewResource(svc.Schema(), newChangeID(key), "empty"),
					},
				})
			})
			var res framer.StreamerResponse
			Eventually(responses.Outlet()).Should(Receive(&res))
			ids := MustSucceed(ontologycdc.DecodeIDs(res.Frame.Series[0].Data))
			// There's a condition here where we might receive the channel creation
			// signal, so we just do a length assertion.
			Expect(len(ids)).To(BeNumerically(">", 0))
			Expect(len(ids[0].Type)).To(BeNumerically(">", 0))
			Expect(len(ids[0].Key)).To(BeNumerically(">", 0))
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			Expect(closeStreamer.Close()).To(Succeed())
		})
		It("Should correctly propagate resource deletes to the ontology", func() {
			var resCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_ontology_resource_delete").Entry(&resCh).Exec(ctx, nil)).To(Succeed())
			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Start: telem.Now(),
				Keys:  channel.Keys{resCh.Key()},
			}))
			requests, responses := confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(5 * time.Millisecond)
			closeStreamer := signal.NewShutdown(sCtx, cancel)
			key := "hello"
			svc.NotifyGenerator(ctx, func() iter.Nexter[schema.Change] {
				return iter.All[schema.Change]([]schema.Change{
					{
						Variant: change.Delete,
						Key:     newChangeID(key),
					},
				})
			})
			var res framer.StreamerResponse
			Eventually(responses.Outlet()).Should(Receive(&res))
			ids := MustSucceed(ontologycdc.DecodeIDs(res.Frame.Series[0].Data))
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
			Start: telem.Now(),
			Keys:  channel.Keys{resCh.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		time.Sleep(10 * time.Millisecond)
		closeStreamer := signal.NewShutdown(sCtx, cancel)
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
		relationships := MustSucceed(ontologycdc.DecodeRelationships(res.Frame.Series[0].Data))
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
			Start: telem.Now(),
			Keys:  channel.Keys{resCh.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		time.Sleep(5 * time.Millisecond)
		closeStreamer := signal.NewShutdown(sCtx, cancel)
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
		relationships := MustSucceed(ontologycdc.DecodeRelationships(res.Frame.Series[0].Data))
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
