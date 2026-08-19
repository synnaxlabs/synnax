// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	"fmt"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicontrol "github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// framerTypeOnly is the type-level ontology ID granting access across all channels.
var framerTypeOnly = ontology.ID{Type: ontology.ResourceTypeFramer}

// channelCounter names each channel distinctly. The mock cluster persists channels
// across specs in the suite.
var channelCounter atomic.Uint64

// createChannel creates a virtual channel with a unique name.
func createChannel(ctx SpecContext) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     fmt.Sprintf("control_%d", channelCounter.Add(1)),
		DataType: telem.Float64T,
		Virtual:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

// openWriter opens a writer that acquires control over keys under the given subject.
func openWriter(
	ctx SpecContext,
	subject string,
	keys ...channel.Key,
) *framer.Writer {
	GinkgoHelper()
	return DeferClose(MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:           keys,
		Start:          telem.SecondTS,
		Sync:           new(true),
		ControlSubject: xcontrol.Subject{Key: subject, Name: subject},
	})))
}

var _ = Describe("Service.Retrieve", func() {
	It("Should return the state of the requested channels", func(ctx SpecContext) {
		ch := createChannel(ctx)
		openWriter(ctx, "philadelphia", ch.Key())
		author := newUser(ctx)
		grantOn(
			ctx,
			author.OntologyID(),
			[]access.Action{access.ActionRetrieve},
			framerTypeOnly,
		)
		res := MustSucceed(apiSvc.Retrieve(
			authedCtx(ctx, author),
			apicontrol.RetrieveRequest{Keys: channel.Keys{ch.Key()}},
		))
		Expect(res.States).To(ConsistOf(apicontrol.State{
			Subject:   xcontrol.Subject{Key: "philadelphia", Name: "philadelphia"},
			Resource:  ch.Key(),
			Authority: xcontrol.AuthorityAbsolute,
		}))
	})

	It("Should return no states for an uncontrolled channel", func(ctx SpecContext) {
		ch := createChannel(ctx)
		author := newUser(ctx)
		grantOn(
			ctx,
			author.OntologyID(),
			[]access.Action{access.ActionRetrieve},
			framerTypeOnly,
		)
		Expect(MustSucceed(apiSvc.Retrieve(
			authedCtx(ctx, author),
			apicontrol.RetrieveRequest{Keys: channel.Keys{ch.Key()}},
		)).States).To(BeEmpty())
	})

	It("Should reject a subject without retrieve access", func(ctx SpecContext) {
		ch := createChannel(ctx)
		openWriter(ctx, "denver", ch.Key())
		Expect(apiSvc.Retrieve(
			authedCtx(ctx, newUser(ctx)),
			apicontrol.RetrieveRequest{Keys: channel.Keys{ch.Key()}},
		)).Error().To(MatchError(access.ErrDenied))
	})
})
