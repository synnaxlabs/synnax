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
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Signals", Ordered, func() {
	var (
		layer     *service.Layer
		responses confluence.Outlet[framer.StreamerResponse]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		layer = MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
	})
	BeforeEach(func(ctx SpecContext) {
		streamer := MustSucceed(layer.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{
				layer.Certificate.SetChannelKey(),
				layer.Certificate.DeleteChannelKey(),
			},
			SendOpenAck: true,
		}))
		var requests confluence.Inlet[framer.StreamerRequest]
		requests, responses = confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		closeStreamer := signal.NewHardShutdown(sCtx, cancel)
		DeferCleanup(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(closeStreamer.Close()).To(Succeed())
		})
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		Eventually(responses.Outlet()).Should(Receive())
	})
	// receive decodes the records in the next frame, which must hold only the channel
	// with the given key.
	receive := func(
		key channel.Key,
		recordOf func([]byte) (resource.ID, error),
	) []resource.ID {
		GinkgoHelper()
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(ConsistOf(key))
		var records []resource.ID
		for sample := range res.Frame.SeriesAt(0).Samples() {
			records = append(records, MustSucceed(recordOf(sample)))
		}
		return records
	}
	create := func(ctx SpecContext) certificate.Certificate {
		GinkgoHelper()
		c := certificate.Certificate{Variant: certificate.FileCertificate{
			Base:            certificate.Base{Name: "Plant"},
			CertificatePath: "/etc/plant.pem",
		}}
		Expect(layer.Certificate.NewWriter(nil).Create(ctx, &c, nil)).To(Succeed())
		return c
	}

	It(
		"Should publish each created certificate on the set channel",
		func(ctx SpecContext) {
			c := create(ctx)
			Expect(receive(layer.Certificate.SetChannelKey(), certificate.SetRecord)).
				To(Equal([]resource.ID{c.ResourceID()}))
		},
	)
	It(
		"Should publish the key of each deleted certificate on the delete channel",
		func(ctx SpecContext) {
			c := create(ctx)
			receive(layer.Certificate.SetChannelKey(), certificate.SetRecord)
			Expect(layer.Certificate.NewWriter(nil).Delete(ctx, c.Base().Key)).
				To(Succeed())
			Expect(receive(
				layer.Certificate.DeleteChannelKey(),
				certificate.DeleteRecord,
			)).To(Equal([]resource.ID{c.ResourceID()}))
		},
	)
})

var _ = Describe("SetRecord", func() {
	It("Should return the resource ID of the certificate in a sample", func() {
		c := certificate.Certificate{Variant: certificate.FileCertificate{
			Base:            certificate.Base{Key: uuid.New(), Name: "Plant"},
			CertificatePath: "/etc/plant.pem",
		}}
		Expect(certificate.SetRecord(MustSucceed(json.Marshal(c)))).
			To(Equal(c.ResourceID()))
	})
	It("Should reject a sample that is not a certificate", func() {
		Expect(certificate.SetRecord([]byte("not json"))).Error().
			To(MatchError(ContainSubstring("invalid character")))
	})
})

var _ = Describe("DeleteRecord", func() {
	It("Should reject a sample that is not a key", func() {
		Expect(certificate.DeleteRecord([]byte{1, 2})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(
				ContainSubstring("certificate delete sample is 2 bytes, not 16"),
			),
		))
	})
})

var _ = Describe("ServiceConfig", func() {
	It("Should require a channel service when signals are set", func() {
		cfg := certificate.DefaultServiceConfig.Override(certificate.ServiceConfig{
			Signals: &signals.Provider{},
		})
		Expect(cfg.Validate()).To(MatchError(
			ContainSubstring("channel: required when signals is set"),
		))
	})
})
