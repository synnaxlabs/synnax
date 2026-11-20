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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// oldDelete runs the delete of the route before it went through impact.Apply, in a
// transaction that it then aborts, and returns its error.
func oldDelete(
	ctx SpecContext,
	db *gorp.DB,
	svc *channel.Service,
	req apichannel.DeleteRequest,
) error {
	GinkgoHelper()
	tx := db.OpenTx()
	defer func() { Expect(tx.Close()).To(Succeed()) }()
	w := svc.NewWriter(tx)
	if len(req.Keys) > 0 {
		if err := w.DeleteMany(ctx, req.Keys, false); err != nil {
			return err
		}
	}
	if len(req.Names) > 0 {
		return w.DeleteManyByNames(ctx, req.Names, false)
	}
	return nil
}

var _ = Describe("Delete breaker", func() {
	createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch.Name = UniqueChannelName()
		if ch.DataType == "" {
			ch.DataType, ch.Virtual = telem.Float32T, true
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createTask := func(ctx SpecContext, chs ...channel.Key) task.Task {
		GinkgoHelper()
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		entries := make([]any, len(chs))
		for i, ch := range chs {
			entries[i] = map[string]any{
				"type":    "coil",
				"key":     UniqueChannelName(),
				"address": i,
				"channel": ch,
			}
		}
		tsk := task.Task{
			Name:   "modbus_read",
			Type:   "modbus_read",
			Config: msgpack.EncodedJSON{"device": dev.Key, "channels": entries},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
				return err
			}
			return layer.Task.NewWriter(tx).Create(ctx, &tsk)
		})).To(Succeed())
		return tsk
	}

	It("Should refuse an index named by key when its data channel is named by name, "+
		"as it did before", func(ctx SpecContext) {
		index := createChannel(
			ctx,
			channel.Channel{IsIndex: true, DataType: telem.TimestampT},
		)
		data := createChannel(ctx, channel.Channel{
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		})
		req := apichannel.DeleteRequest{
			Keys:  channel.Keys{index.Key()},
			Names: []string{data.Name},
		}
		want := oldDelete(ctx, db, channelSvc, req)
		Expect(want).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("channel", index.Key()),
			Holder:    gorp.NewTarget("channel", data.Key()),
			Reference: "index",
		}))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := apiSvc.Delete(ctx, tx, req)
			return err
		})).To(MatchError(want))
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(index.Key(), data.Key())).
			Count(ctx, db)).To(Equal(2))
	})

	It("Should name the channel that it named before when a task holds two deleted "+
		"channels", func(ctx SpecContext) {
		a := createChannel(ctx, channel.Channel{})
		b := createChannel(ctx, channel.Channel{})
		createTask(ctx, a.Key(), b.Key())
		req := apichannel.DeleteRequest{Keys: channel.Keys{b.Key(), a.Key()}}
		want := oldDelete(ctx, db, channelSvc, req)
		Expect(want).To(HaveOccurred())
		Expect(deleteChannels(AuthedCtx(ctx, author), req)).Error().
			To(MatchError(want))
	})

	Describe("License limit", Ordered, func() {
		var (
			limited   *service.Layer
			limitedDB *gorp.DB
			svc       *apichannel.Service
			over      channel.Channel
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			node := testutil.NewNode(ctx)
			limitedDB = node.DB
			signer := svctestutil.NewSigner()
			lic := svctestutil.NewLicense()
			limited = MustOpen(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
				License: license.ServiceConfig{
					Key:     signer.Sign(lic),
					Anchors: signer.Anchors,
				},
			}))
			svc = MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
				DB:      limitedDB,
				RBAC:    limited.RBAC,
				Channel: limited.Channel,
				Status:  limited.Status,
				Impact:  limited.Impact,
			}))
			for range 2 {
				over = channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(limitedDB.WithTx(ctx, func(tx gorp.Tx) error {
					return limited.Channel.NewWriter(tx).Create(ctx, &over)
				})).To(Succeed())
			}
			// A newer license with one channel fewer than the Core holds puts the last
			// channel over the limit, as a renewal with a lower cap does.
			smaller := svctestutil.NewLicense()
			smaller.Iat = lic.Iat + 1
			smaller.Channels = limited.Channel.CountLimited() - 1
			MustSucceed(limited.License.Activate(ctx, signer.Sign(smaller)))
		})

		It("Should delete a channel over the license limit by name as it did before",
			func(ctx SpecContext) {
				req := apichannel.DeleteRequest{Names: []string{over.Name}}
				Expect(oldDelete(ctx, limitedDB, limited.Channel, req)).To(Succeed())
				Expect(limitedDB.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := svc.Delete(ctx, tx, req)
					return err
				})).To(Succeed())
				Expect(MustSucceed(limited.Channel.NewRetrieve().
					Where(channel.MatchKeys(over.Key())).
					Exists(ctx, limitedDB))).To(BeFalse())
			})
	})
})
