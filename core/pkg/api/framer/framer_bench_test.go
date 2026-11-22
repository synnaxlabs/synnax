// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// openServiceLayer opens a single-node Core service layer that closes with b, and
// returns it with the node's Gorp DB.
func openServiceLayer(b *testing.B) (*service.Layer, *gorp.DB) {
	b.Helper()
	gomega.RegisterTestingT(b)
	node := testutil.OpenNode(b.Context())
	b.Cleanup(func() {
		if err := node.Close(); err != nil {
			b.Error(err)
		}
	})
	sec := xtestutil.MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
	}))
	layer := xtestutil.MustSucceed(service.OpenLayer(b.Context(), service.LayerConfig{
		Distribution:    node.Layer,
		Security:        sec,
		Storage:         node.Storage,
		RootCredentials: auth.Credentials{Username: "bench-root", Password: "p"},
		SecretsKey:      secret.NewEncryptionKey(),
	}))
	b.Cleanup(func() {
		if err := layer.Close(); err != nil {
			b.Error(err)
		}
	})
	return layer, node.DB
}

// BenchmarkStreamKeyChange measures a streamer request that resends the same keys and
// waits for its ack. The keys are raw virtual channels and the calculated channels that
// read the first of them.
func BenchmarkStreamKeyChange(b *testing.B) {
	layer, db := openServiceLayer(b)
	creator := calctestutil.Creator{
		DB:          db,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
	}
	api := xtestutil.MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
		DB:          db,
		Framer:      layer.Framer,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
		RBAC:        layer.RBAC,
		Errors:      errors.NewCodec(),
	}))
	cases := []struct {
		name            string
		raw, calculated int
	}{
		{"calculated=0", 1, 0},
		{"calculated=1", 1, 1},
		{"calculated=10", 1, 10},
		{"calculated=100", 1, 100},
		{"raw=100", 100, 0},
		{"raw=1000", 1000, 0},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			ctx := b.Context()
			raw := make([]channel.Channel, c.raw)
			for i := range raw {
				raw[i] = channel.Channel{
					Name:     channeltestutil.UniqueChannelName(),
					DataType: telem.Float32T,
					Virtual:  true,
				}
			}
			calculations := make([]calctestutil.Calculation, c.calculated)
			for i := range calculations {
				calculations[i] = calctestutil.Calculation{
					Name:       channeltestutil.UniqueChannelName(),
					DataType:   telem.Float32T,
					Expression: "return " + raw[0].Name + " * 2",
				}
			}
			gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).CreateMany(ctx, &raw)
			})).To(gomega.Succeed())
			calcs := xtestutil.MustSucceed(creator.CreateMany(ctx, calculations))
			calcs = append(calcs, xtestutil.MustSucceed(creator.Indexes(ctx, calcs))...)
			keys := channel.KeysFromChannels(slices.Concat(raw, calcs))
			u := xtestutil.MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				layer.RBAC,
				layer.User,
				user.User{Name: "bench-" + uuid.New().String()},
			))
			xtestutil.MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				layer.RBAC,
				layer.User,
				u.Key,
				[]access.Action{access.ActionRetrieve},
				svcframer.ResourceIDs(keys)...,
			))
			fCtx := freighter.Context{Context: ctx, Params: freighter.Params{}}
			fCtx.Set("Subject", u.ResourceID())
			client, server := freightertestutil.NewStreams[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			](ctx)
			done := make(chan error, 1)
			go func() { done <- api.Stream(fCtx, server) }()
			gomega.Expect(client.Send(apiframer.StreamerRequest{Keys: keys})).
				To(gomega.Succeed())
			gomega.Expect(xtestutil.MustSucceed(client.Receive()).Frame.Empty()).
				To(gomega.BeTrue())
			req := apiframer.StreamerRequest{Keys: keys, AckRequested: true}
			b.ReportAllocs()
			for b.Loop() {
				if err := client.Send(req); err != nil {
					b.Fatal(err)
				}
				res, err := client.Receive()
				if err != nil {
					b.Fatal(err)
				}
				if !res.Ack {
					b.Fatalf("expected an ack, got %+v", res)
				}
			}
			gomega.Expect(client.CloseSend()).To(gomega.Succeed())
			gomega.Eventually(done).WithTimeout(5 * time.Second).
				Should(gomega.Receive(gomega.Succeed()))
		})
	}
}
