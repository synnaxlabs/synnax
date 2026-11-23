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
	"context"
	"fmt"
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
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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

// createCalculatedChannels creates raw virtual channels and calculated channels that
// read the first of them. It returns the keys of all of them, the raw channels first.
func createCalculatedChannels(
	ctx context.Context,
	layer *service.Layer,
	db *gorp.DB,
	raw, calculated int,
) channel.Keys {
	bases := make([]channel.Channel, raw)
	for i := range bases {
		bases[i] = channel.Channel{
			Name:     channeltestutil.UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	calculations := make([]calctestutil.Calculation, calculated)
	for i := range calculations {
		calculations[i] = calctestutil.Calculation{
			Name:       channeltestutil.UniqueChannelName(),
			DataType:   telem.Float32T,
			Expression: "return " + bases[0].Name + " * 2",
		}
	}
	gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).CreateMany(ctx, &bases)
	})).To(gomega.Succeed())
	creator := calctestutil.Creator{
		DB:          db,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
	}
	calcs := xtestutil.MustSucceed(creator.CreateMany(ctx, calculations))
	calcs = append(calcs, xtestutil.MustSucceed(creator.Indexes(ctx, calcs))...)
	return channel.KeysFromChannels(slices.Concat(bases, calcs))
}

// subjectContext creates a user that may retrieve keys and returns a Freighter context
// that carries the user as its subject.
func subjectContext(
	ctx context.Context,
	layer *service.Layer,
	db *gorp.DB,
	keys channel.Keys,
) freighter.Context {
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
	return fCtx
}

// BenchmarkStreamKeyChange measures a streamer request that resends the same keys and
// waits for its ack. The keys are raw virtual channels and the calculated channels that
// read the first of them.
func BenchmarkStreamKeyChange(b *testing.B) {
	layer, db := openServiceLayer(b)
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
			keys := createCalculatedChannels(ctx, layer, db, c.raw, c.calculated)
			fCtx := subjectContext(ctx, layer, db, keys)
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

// BenchmarkCalculationChange measures an edit of the expression of a calculation and
// its
// delivery to each streamer of a calculation change channel, as a subject that reads
// the calculation.
func BenchmarkCalculationChange(b *testing.B) {
	layer, db := openServiceLayer(b)
	api := xtestutil.MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
		DB:          db,
		Framer:      layer.Framer,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
		RBAC:        layer.RBAC,
		Errors:      errors.NewCodec(),
		Records:     layer.Signals.Records(),
	}))
	for _, name := range []string{"sy_calculation_set", "sy_calculation_action_set"} {
		for _, subscribers := range []int{1, 10, 100} {
			b.Run(
				fmt.Sprintf("%s/subscribers=%d", name, subscribers),
				func(b *testing.B) {
					benchCalculationChange(b, layer, db, api, name, subscribers)
				},
			)
		}
	}
}

type calculationStream = freighter.ClientStream[
	apiframer.StreamerRequest,
	apiframer.StreamerResponse,
]

func benchCalculationChange(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
	api *apiframer.Service,
	name string,
	subscribers int,
) {
	b.Helper()
	ctx := b.Context()
	input := channel.Channel{
		Name:     channeltestutil.UniqueChannelName(),
		DataType: telem.Float32T,
		Virtual:  true,
	}
	gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).Create(ctx, &input)
	})).To(gomega.Succeed())
	creator := calctestutil.Creator{
		DB:          db,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
	}
	output := xtestutil.MustSucceed(creator.Create(ctx, calctestutil.Calculation{
		Name:       channeltestutil.UniqueChannelName(),
		Expression: "return " + input.Name + " * 2",
	}))
	c := xtestutil.MustSucceed(creator.CalculationOf(ctx, output.Key()))
	clients := openCalculationStreams(b, layer, db, api, name, c, subscribers)
	n := 0
	b.ReportAllocs()
	for b.Loop() {
		n++
		if err := db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := layer.Calculation.NewWriter(tx).Apply(
				ctx,
				c.Key,
				calculation.NewSetAction(calculation.SetPayload{
					Path:  []string{"expression"},
					Value: fmt.Sprintf("return %s * %d", input.Name, n),
				}),
			)
			return err
		}); err != nil {
			b.Fatal(err)
		}
		for _, client := range clients {
			res, err := client.Receive()
			if err != nil {
				b.Fatal(err)
			}
			if res.Frame.Empty() {
				b.Fatal("expected a change")
			}
		}
	}
}

// openCalculationStreams opens count streams of the change channel with the given name,
// as a user that reads c, and closes them when b ends.
func openCalculationStreams(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
	api *apiframer.Service,
	name string,
	c calculation.Calculation,
	count int,
) []calculationStream {
	b.Helper()
	ctx := b.Context()
	var ch channel.Channel
	gomega.Expect(layer.Channel.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&ch).
		Exec(ctx, db)).To(gomega.Succeed())
	u := createBenchUser(b, layer, db, slices.Concat(
		svcframer.ResourceIDs(channel.Keys{ch.Key()}),
		[]resource.ID{calculation.ResourceID(c.Key)},
		layer.Calculation.Channels(c),
	)...)
	// The streams outlive b.Context, which ends before the cleanup closes them.
	streamCtx, cancel := context.WithCancel(context.Background())
	fCtx := freighter.Context{Context: streamCtx, Params: freighter.Params{}}
	fCtx.Set("Subject", u.ResourceID())
	clients := make([]calculationStream, count)
	done := make(chan error, count)
	for i := range clients {
		client, server := freightertestutil.NewStreams[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		](streamCtx)
		clients[i] = client
		go func() { done <- api.Stream(fCtx, server) }()
		gomega.Expect(client.Send(apiframer.StreamerRequest{
			Keys: channel.Keys{ch.Key()},
		})).To(gomega.Succeed())
		gomega.Expect(xtestutil.MustSucceed(client.Receive()).Frame.Empty()).
			To(gomega.BeTrue())
	}
	b.Cleanup(func() {
		for _, client := range clients {
			gomega.Expect(client.CloseSend()).To(gomega.Succeed())
		}
		for range clients {
			gomega.Eventually(done).WithTimeout(5 * time.Second).
				Should(gomega.Receive(gomega.Succeed()))
		}
		cancel()
	})
	return clients
}

// createBenchUser creates a user that can retrieve the granted resources.
func createBenchUser(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
	granted ...resource.ID,
) user.User {
	b.Helper()
	ctx := b.Context()
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
		granted...,
	))
	return u
}
