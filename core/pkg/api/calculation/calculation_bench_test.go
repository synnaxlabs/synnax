// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	apitestutil "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// createOutputs creates n virtual channels, each the output of a calculation.
func createOutputs(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
	n int,
) []channel.Channel {
	b.Helper()
	ctx := b.Context()
	outputs := make([]channel.Channel, n)
	for i := range outputs {
		outputs[i] = channel.Channel{
			Name:     "bench_" + uuid.New().String()[:8],
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := layer.Channel.NewWriter(tx).CreateMany(ctx, &outputs); err != nil {
			return err
		}
		w := layer.Calculation.NewWriter(tx)
		for _, out := range outputs {
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			if err := w.Create(ctx, &c); err != nil {
				return err
			}
		}
		return nil
	})).To(gomega.Succeed())
	return outputs
}

// createReader creates a user that can retrieve every channel, and every calculation
// when shown.
func createReader(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
	shown bool,
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
	objects := []resource.ID{{Type: resource.TypeChannel}}
	if shown {
		objects = append(objects, resource.ID{Type: resource.TypeCalculation})
	}
	xtestutil.MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		layer.RBAC,
		layer.User,
		u.Key,
		[]access.Action{access.ActionRetrieve},
		objects...,
	))
	return u
}

// openRetrieve opens a single-node Core service layer that closes with b, and returns
// it with its Gorp DB and the calculation retrieve route as the API layer binds it.
func openRetrieve(b *testing.B) (
	*service.Layer,
	*gorp.DB,
	freighter.UnaryHandler[apicalculation.RetrieveRequest, apicalculation.RetrieveResponse],
) {
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
	api := xtestutil.MustSucceed(apicalculation.NewService(apicalculation.ServiceConfig{
		Calculation: layer.Calculation,
		Status:      layer.Status,
		RBAC:        layer.RBAC,
	}))
	binder := xtestutil.MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: layer.RBAC,
		Held: apiauth.HeldTables(),
	}))
	return layer, node.DB, bind.Read(binder, api.Retrieve)
}

// BenchmarkRetrieveByOutput measures a retrieve of the calculations that write n
// channels. Under shown, the subject can retrieve each calculation. Under hidden, it
// can retrieve each output channel but no calculation.
func BenchmarkRetrieveByOutput(b *testing.B) {
	layer, db, retrieve := openRetrieve(b)
	for _, n := range []int{1, 10, 100} {
		for _, shown := range []bool{true, false} {
			name, want := fmt.Sprintf("hidden/n=%d", n), 0
			if shown {
				name, want = fmt.Sprintf("shown/n=%d", n), n
			}
			b.Run(name, func(b *testing.B) {
				outputs := createOutputs(b, layer, db, n)
				authed := apitestutil.AuthedCtx(
					b.Context(),
					createReader(b, layer, db, shown),
				)
				req := apicalculation.RetrieveRequest{Where: &calculation.Where{
					Output: &filter.Set[channel.Key]{
						In: channel.KeysFromChannels(outputs),
					},
				}}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res := xtestutil.MustSucceed(retrieve(authed, req))
					if got := len(res.Calculations); got != want {
						b.Fatalf("got %d calculations, want %d", got, want)
					}
				}
			})
		}
	}
}
