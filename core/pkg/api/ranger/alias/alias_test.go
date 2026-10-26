// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apialias "github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/x/telem"
)

func createChannel(ctx SpecContext) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: telem.Float32T,
		Virtual:  true,
	}
	Expect(svc.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

var _ = Describe("Search", func() {
	var (
		rng           ranger.Range
		inlet, outlet channel.Channel
		req           apialias.SearchRequest
	)
	BeforeEach(func(ctx SpecContext) {
		rng = ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &rng)).To(Succeed())
		inlet, outlet = createChannel(ctx), createChannel(ctx)
		w := svc.Alias.NewWriter(nil)
		Expect(w.Set(ctx, rng.Key, inlet.Key(), "turbopump inlet")).To(Succeed())
		Expect(w.Set(ctx, rng.Key, outlet.Key(), "turbopump")).To(Succeed())
		req = apialias.SearchRequest{Range: rng.Key, Term: "turbopump"}
	})

	It(
		"Should return the channels whose alias matches, closest match first",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantRetrieveOn(ctx, u.OntologyID(), alias.OntologyIDs(
				rng.Key,
				[]channel.Key{inlet.Key(), outlet.Key()},
			)...)
			Expect(apiSvc.Search(AuthedCtx(ctx, u), req)).To(Equal(
				apialias.SearchResponse{
					Channels: []channel.Key{outlet.Key(), inlet.Key()},
				},
			))
		},
	)

	It("Should reject the request when retrieve is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(apiSvc.Search(AuthedCtx(ctx, u), req)).
			Error().To(MatchError(access.ErrDenied))
	})
})
