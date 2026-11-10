// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"context"
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Signals", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})
	// receive streams the channel with the given name and returns the samples of the
	// next frame after write runs.
	receive := func(ctx context.Context, name string, write func()) [][]byte {
		GinkgoHelper()
		var ch channel.Channel
		Expect(layer.Channel.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		streamer := MustSucceed(layer.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        channel.Keys{ch.Key()},
			SendOpenAck: true,
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		defer func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(shutdown.Close()).To(Succeed())
		}()
		Eventually(responses.Outlet()).Should(Receive())
		write()
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		var samples [][]byte
		for sample := range res.Frame.SeriesAt(0).Samples() {
			samples = append(samples, sample)
		}
		return samples
	}
	write := func(ctx context.Context, fn func(certificate.Writer) error) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(layer.Certificate.NewWriter(tx))
		})).To(Succeed())
	}
	file := func(name string) certificate.Certificate {
		return certificate.Certificate{Variant: certificate.FileCertificate{
			Base:            certificate.Base{Name: name},
			CertificatePath: "/etc/plant.pem",
		}}
	}
	create := func(ctx context.Context, c *certificate.Certificate) {
		GinkgoHelper()
		write(ctx, func(w certificate.Writer) error { return w.Set(ctx, c, nil) })
	}

	It("Should publish each created certificate", func(ctx SpecContext) {
		c := file("Set")
		samples := receive(ctx, "sy_certificate_set", func() { create(ctx, &c) })
		Expect(samples).To(HaveLen(1))
		var got certificate.Certificate
		Expect(json.Unmarshal(samples[0], &got)).To(Succeed())
		Expect(got).To(Equal(c))
	})

	It("Should broadcast a create action", func(ctx SpecContext) {
		c := file("Action")
		samples := receive(ctx, "sy_certificate_action_set", func() { create(ctx, &c) })
		Expect(samples).To(HaveLen(1))
		var scoped actions.Scoped[certificate.Key, certificate.Action]
		Expect(json.Unmarshal(samples[0], &scoped)).To(Succeed())
		Expect(scoped.Key).To(Equal(c.Base().Key))
		Expect(scoped.Actions).To(HaveLen(1))
		Expect(scoped.Actions[0].Create.Certificate).To(Equal(c))
	})

	It("Should publish the key of each deleted certificate", func(ctx SpecContext) {
		c := file("Delete")
		create(ctx, &c)
		samples := receive(ctx, "sy_certificate_delete", func() {
			write(ctx, func(w certificate.Writer) error {
				return w.Delete(ctx, c.Base().Key)
			})
		})
		Expect(samples).To(HaveLen(1))
		Expect(uuid.UUID(samples[0])).To(Equal(c.Base().Key))
	})
})
