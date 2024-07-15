// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"time"
)

const (
	subscriberSetChannelName    = "subscriber_set"
	subscriberDeleteChannelName = "subscriber_delete"
)

var _ = Describe("Subscriber", func() {
	It("Should correctly receive a change", func() {
		publishObs := observe.New[[]change.Change[[]byte, struct{}]]()
		publishCfg := signals.ObservablePublisherConfig{
			SetChannel:    channel.Channel{Name: subscriberSetChannelName, DataType: telem.UUIDT},
			DeleteChannel: channel.Channel{Name: subscriberDeleteChannelName, DataType: telem.UUIDT},
			Observable:    publishObs,
		}
		publishCloser := MustSucceed(dist.Signals.PublishFromObservable(ctx, publishCfg))
		defer func() {
			GinkgoRecover()
			Expect(publishCloser.Close()).To(Succeed())
		}()
		sCtx, cancel := signal.Isolated()
		subscriber := MustSucceed(dist.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
			SetChannelName:    subscriberSetChannelName,
			DeleteChannelName: subscriberDeleteChannelName,
		}))
		defer func() {
			GinkgoRecover()
			cancel()
			Expect(sCtx.Wait()).To(HaveOccurredAs(context.Canceled))
		}()
		time.Sleep(10 * time.Millisecond)
		uid := uuid.New()
		var oChange []change.Change[[]byte, struct{}]
		subscriber.OnChange(func(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
			oChange = changes
		})
		publishObs.Notify(ctx, []change.Change[[]byte, struct{}]{{
			Variant: change.Set,
			Key:     uid[:],
		}})
		Eventually(func() []change.Change[[]byte, struct{}] {
			return oChange
		}).Should(HaveLen(1))
	})
})
