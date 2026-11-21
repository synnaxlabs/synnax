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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NameErrors", func() {
	var ch channel.Channel
	BeforeEach(func(ctx SpecContext) {
		ch = channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
	})
	channelErr := func(key channel.Key) error {
		return cesium.ChannelError{
			Key: uint32(key),
			Err: errors.Wrap(validate.ErrValidation, "bad write"),
		}
	}

	It(
		"Should name a channel by its record and keep the sentinels of the error",
		func(ctx SpecContext) {
			err := errors.Wrap(channelErr(ch.Key()), "write failed")
			named := svc.NameErrors(ctx, err, nil)
			Expect(named.Error()).To(Equal(fmt.Sprintf(
				"write failed: channel %q: bad write: validation error",
				ch.Name,
			)))
			Expect(named).To(MatchError(validate.ErrValidation))
			var ce cesium.ChannelError
			Expect(errors.As(named, &ce)).To(BeTrue())
			Expect(ce.Key).To(Equal(uint32(ch.Key())))
			Expect(errors.MessageWithout(named, validate.ErrValidation)).To(Equal(
				fmt.Sprintf("write failed: channel %q: bad write", ch.Name),
			))
		},
	)

	It("Should name a channel of a distribution channel error", func(ctx SpecContext) {
		err := distchannel.Error{Key: ch.Key(), Err: errors.New("bad write")}
		Expect(svc.NameErrors(ctx, err, nil).Error()).To(Equal(
			fmt.Sprintf("channel %q: bad write", ch.Name),
		))
	})

	It("Should name a channel that has no record as deleted", func(ctx SpecContext) {
		key := distchannel.NewKey(1, 64_000)
		Expect(svc.NameErrors(ctx, channelErr(key), nil).Error()).To(Equal(
			fmt.Sprintf("deleted channel <%d>: bad write: validation error", key),
		))
	})

	It("Should keep the key of a channel that the caller cannot see", func(
		ctx SpecContext,
	) {
		err := channelErr(ch.Key())
		hidden := func(channel.Key) bool { return false }
		Expect(svc.NameErrors(ctx, err, hidden).Error()).To(Equal(err.Error()))
	})

	It("Should name the channels of every member of a join", func(ctx SpecContext) {
		missing := distchannel.NewKey(1, 64_001)
		err := errors.Join(channelErr(ch.Key()), channelErr(missing))
		named := svc.NameErrors(ctx, err, nil)
		Expect(named.Error()).To(ContainSubstring("channel %q: bad write", ch.Name))
		Expect(named.Error()).To(ContainSubstring("deleted channel <%d>", missing))
		Expect(named).To(MatchError(validate.ErrValidation))
	})

	It("Should name every key that Cesium prints in a message", func(ctx SpecContext) {
		err := errors.Wrapf(
			validate.ErrValidation,
			"index <%d> is missing a series for channel <%d>",
			ch.Key(),
			ch.Key(),
		)
		named := svc.NameErrors(ctx, err, nil)
		Expect(errors.MessageWithout(named, validate.ErrValidation)).To(Equal(
			fmt.Sprintf(
				"index %q is missing a series for channel %q",
				ch.Name,
				ch.Name,
			),
		))
		Expect(named).To(MatchError(validate.ErrValidation))
	})

	It("Should name a channel that the Core printed with a bracketed name", func(
		ctx SpecContext,
	) {
		err := errors.Wrapf(
			validate.ErrValidation,
			"channel [old]<%d>: expected data type float32",
			ch.Key(),
		)
		Expect(svc.NameErrors(ctx, err, nil).Error()).To(Equal(fmt.Sprintf(
			"channel %q: expected data type float32: validation error",
			ch.Name,
		)))
	})

	It("Should return an error with no channel error unchanged", func(ctx SpecContext) {
		err := errors.New("no channel here")
		Expect(svc.NameErrors(ctx, err, nil)).To(BeIdenticalTo(err))
	})
})
