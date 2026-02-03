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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

func fixedOverflowChecker(limit int) channel.IntOverflowChecker {
	return func(count types.Uint20) error {
		if count > types.Uint20(limit) {
			return errors.New("channel limit exceeded")
		}
		return nil
	}
}

var _ = Describe("Limit", Ordered, func() {
	var (
		limit       = 5
		mockCluster *mock.Cluster
		dist        mock.Node
	)
	BeforeEach(func() {
		mockCluster = mock.NewCluster()
		dist = mockCluster.Provision(ctx, distribution.Config{
			TestingIntOverflowCheck: fixedOverflowChecker(limit),
		})
	})
	AfterEach(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	It("Should not allow creating channels over the limit", func() {
		// Create channels up to the limit
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &ch)).To(Succeed())
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		err := dist.Channel.Create(ctx, &overLimitCh)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))
	})

	It("Should allow creating channels after deleting some to stay under the limit", func() {
		// Create channels up to the limit
		channels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &ch)).To(Succeed())
			channels[i] = ch
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		err := dist.Channel.Create(ctx, &overLimitCh)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))

		// Delete one channel
		writer := dist.Channel.NewWriter(nil)
		Expect(writer.Delete(ctx, channels[0].Key(), false)).To(Succeed())

		// Now we should be able to create a new channel
		newCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "NewAfterDelete",
			Leaseholder: 1,
		}
		Expect(dist.Channel.Create(ctx, &newCh)).To(Succeed())

		// Try to create one more channel (should fail again)
		anotherCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "AnotherOverLimit",
			Leaseholder: 1,
		}
		err = dist.Channel.Create(ctx, &anotherCh)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))
	})

	It("Should allow retrieving channels even at the limit", func() {
		// Create channels up to the limit
		createdChannels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &ch)).To(Succeed())
			createdChannels[i] = ch
		}

		// Try to create one more channel over the limit
		overLimitCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "OverLimit",
			Leaseholder: 1,
		}
		err := dist.Channel.Create(ctx, &overLimitCh)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))

		// Retrieve all channels - this should work fine even at the limit
		var retrievedChannels []channel.Channel
		retrieve := dist.Channel.NewRetrieve()
		err = retrieve.Entries(&retrievedChannels).WhereNodeKey(1).Exec(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(retrievedChannels).To(HaveLen(limit + internalChannelCount))

		// Retrieve a specific channel by name
		var singleChannel channel.Channel
		err = retrieve.WhereKeys(createdChannels[0].Key()).Entry(&singleChannel).Exec(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(singleChannel.Name).To(Equal(createdChannels[0].Name))
	})
	It("Should not edit the channel limit if a deletion fails in TS", func() {
		createdChannels := make([]channel.Channel, int(limit))
		for i := range limit {
			ch := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        fmt.Sprintf("LimitTest%d", i),
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &ch)).To(Succeed())
			createdChannels[i] = ch
		}
		writer := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Keys: []channel.Key{createdChannels[0].Key()},
		}))
		Expect(dist.Channel.Delete(ctx, createdChannels[0].Key(), false)).
			To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
		newCh := channel.Channel{
			IsIndex:     true,
			DataType:    telem.TimeStampT,
			Name:        "NewAfterDelete",
			Leaseholder: 1,
		}
		Expect(dist.Channel.Create(ctx, &newCh)).
			To(MatchError(ContainSubstring("channel limit exceeded")))
		Expect(writer.Close()).To(Succeed())
	})
})
