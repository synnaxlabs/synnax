// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
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
				layer.Secret.SetChannelKey(),
				layer.Secret.DeleteChannelKey(),
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
		recordOf func([]byte) (ontology.ID, error),
	) []ontology.ID {
		GinkgoHelper()
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(ConsistOf(key))
		var records []ontology.ID
		for sample := range res.Frame.SeriesAt(0).Samples() {
			records = append(records, MustSucceed(recordOf(sample)))
		}
		return records
	}
	create := func(ctx SpecContext) secret.Secret {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.FileSecret{
			Base: secret.Base{Name: "Key file"},
			Path: "/etc/key.pem",
		}}
		Expect(layer.Secret.NewWriter(nil).Create(ctx, &s, nil)).To(Succeed())
		return s
	}

	It("Should publish each created secret on the set channel", func(ctx SpecContext) {
		s := create(ctx)
		Expect(receive(layer.Secret.SetChannelKey(), secret.SetRecord)).
			To(Equal([]ontology.ID{s.OntologyID()}))
	})
	It(
		"Should publish the key of each deleted secret on the delete channel",
		func(ctx SpecContext) {
			s := create(ctx)
			receive(layer.Secret.SetChannelKey(), secret.SetRecord)
			Expect(layer.Secret.NewWriter(nil).Delete(ctx, s.Base().Key)).
				To(Succeed())
			Expect(receive(layer.Secret.DeleteChannelKey(), secret.DeleteRecord)).
				To(Equal([]ontology.ID{s.OntologyID()}))
		},
	)
})

var _ = Describe("SetRecord", func() {
	It("Should reject a sample that is not a secret", func() {
		Expect(secret.SetRecord([]byte("not json"))).Error().
			To(MatchError(ContainSubstring("invalid character")))
	})
})

var _ = Describe("DeleteRecord", func() {
	It("Should reject a sample that is not a key", func() {
		Expect(secret.DeleteRecord([]byte{1, 2})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("secret delete sample is 2 bytes, not 16")),
		))
	})
})

var _ = Describe("ServiceConfig", func() {
	It("Should require a channel service when signals are set", func() {
		cfg := secret.DefaultServiceConfig.Override(secret.ServiceConfig{
			Signals: &signals.Provider{},
		})
		Expect(cfg.Validate()).To(MatchError(
			ContainSubstring("channel: required when signals is set"),
		))
	})
})
