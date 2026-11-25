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
	"encoding/json/v2"
	"fmt"
	"math/rand/v2"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	channelpb "github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Description breaker", func() {
	ch := apichannel.Channel{
		Key:         7,
		Name:        "pt_01",
		DataType:    telem.Float32T,
		Virtual:     true,
		Description: "Druck am Einlass é\U0001F600",
	}

	It("Should keep the description through a proto round trip on field 14", func() {
		pb := MustSucceed(channelpb.ChannelToPB(ch))
		b := MustSucceed(proto.Marshal(pb))
		var found bool
		for len(b) > 0 {
			num, typ, n := protowire.ConsumeTag(b)
			Expect(n).To(BeNumerically(">", 0))
			b = b[n:]
			if num == 14 {
				Expect(typ).To(Equal(protowire.BytesType))
				v, m := protowire.ConsumeBytes(b)
				Expect(string(v)).To(Equal(ch.Description))
				found = true
				b = b[m:]
				continue
			}
			m := protowire.ConsumeFieldValue(num, typ, b)
			Expect(m).To(BeNumerically(">=", 0))
			b = b[m:]
		}
		Expect(found).To(BeTrue())
		var decoded channelpb.Channel
		Expect(proto.Unmarshal(MustSucceed(proto.Marshal(pb)), &decoded)).To(Succeed())
		back := MustSucceed(channelpb.ChannelFromPB(&decoded))
		Expect(back.Description).To(Equal(ch.Description))
	})

	It("Should keep the description through JSON and msgpack round trips", func() {
		var fromJSON apichannel.Channel
		Expect(json.Unmarshal(MustSucceed(json.Marshal(ch)), &fromJSON)).To(Succeed())
		Expect(fromJSON.Description).To(Equal(ch.Description))
		var fromMsgpack apichannel.Channel
		Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(ch)), &fromMsgpack)).
			To(Succeed())
		Expect(fromMsgpack.Description).To(Equal(ch.Description))
	})

	It("Should read a channel with no description member as an empty one", func() {
		var decoded apichannel.Channel
		Expect(json.Unmarshal(
			[]byte(`{"key":1,"name":"a","data_type":"float32"}`),
			&decoded,
		)).To(Succeed())
		Expect(decoded.Description).To(BeEmpty())
	})

	It("Should refuse a create with a description over the cap", func(ctx SpecContext) {
		_, err := createChannels(
			AuthedCtx(ctx, author),
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:        UniqueChannelName(),
				DataType:    telem.Float32T,
				Virtual:     true,
				Description: strings.Repeat("é", 1025),
			}}},
		)
		Expect(err).To(MatchError(validate.ErrValidation))
	})

	It("Should set a description through dispatch and find it by search", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "describer-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{
				access.ActionCreate,
				access.ActionUpdate,
				access.ActionRetrieve,
			},
			resource.ID{Type: resource.TypeChannel},
		)
		created := MustSucceed(createChannels(
			AuthedCtx(ctx, u),
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}}},
		)).Channels[0]
		word := fmt.Sprintf("kw%016x", rand.Uint64())
		MustSucceed(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
			Key: created.Key,
			Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
				Path:  []string{"description"},
				Value: "flow " + word,
			})},
		}))
		got := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Key: &filter.Set[channel.Key]{Eq: &created.Key},
			}},
		)).Channels
		Expect(got[0].Description).To(Equal("flow " + word))
		found := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{SearchTerm: word},
		)).Channels
		Expect(found).To(ContainElement(HaveField("Key", created.Key)))
	})
})
