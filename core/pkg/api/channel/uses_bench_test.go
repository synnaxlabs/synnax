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
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkRetrieveUses measures the retrieve route for a page of 100 channels by key,
// with and without their uses. Each channel sits in three schematics, and the subject
// can retrieve channels and schematics.
func BenchmarkRetrieveUses(b *testing.B) {
	gomega.RegisterTestingT(b)
	node := disttestutil.OpenNode(b.Context())
	layer, err := svctestutil.OpenLayer(b.Context(), node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	defer func() {
		if err := layer.Close(); err != nil {
			b.Errorf("failed to close the layer: %v", err)
		}
		if err := node.Close(); err != nil {
			b.Errorf("failed to close the node: %v", err)
		}
	}()
	retrieve := benchRetrieve(b, node.DB, layer)
	subject := benchSubject(b, node.DB, layer)
	keys := createUsedChannels(b, node.DB, layer)
	run := func(b *testing.B, uses bool) {
		b.Helper()
		b.ReportAllocs()
		for range b.N {
			res, err := retrieve(subject, apichannel.RetrieveRequest{
				Where:       &channel.Where{Key: &filter.Set[channel.Key]{In: keys}},
				IncludeUses: uses,
			})
			if err != nil {
				b.Fatalf("failed to retrieve: %v", err)
			}
			if len(res.Channels) != len(keys) {
				b.Fatalf("expected %d channels, got %d", len(keys), len(res.Channels))
			}
		}
	}
	b.Run("page", func(b *testing.B) { run(b, false) })
	b.Run("page_uses", func(b *testing.B) { run(b, true) })
}

// benchRetrieve returns the retrieve route as the API layer binds it.
func benchRetrieve(
	b *testing.B,
	db *gorp.DB,
	layer *service.Layer,
) freighter.UnaryHandler[apichannel.RetrieveRequest, apichannel.RetrieveResponse] {
	b.Helper()
	svc, err := apichannel.NewService(apichannel.ServiceConfig{
		DB:      db,
		RBAC:    layer.RBAC,
		Channel: layer.Channel,
		Status:  layer.Status,
		Impact:  layer.Impact,
	})
	if err != nil {
		b.Fatalf("failed to open the API service: %v", err)
	}
	binder, err := bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: layer.RBAC,
		Held: apiauth.HeldTables(),
	})
	if err != nil {
		b.Fatalf("failed to open the binder: %v", err)
	}
	return bind.Read(binder, svc.Retrieve)
}

// benchSubject returns a context whose subject can retrieve channels and schematics.
func benchSubject(b *testing.B, db *gorp.DB, layer *service.Layer) freighter.Context {
	b.Helper()
	u, err := rbactestutil.CreateUser(
		b.Context(), db, layer.RBAC, layer.User, user.User{Name: "bench"},
	)
	if err != nil {
		b.Fatalf("failed to create the user: %v", err)
	}
	if _, err = rbactestutil.Grant(
		b.Context(),
		db,
		layer.RBAC,
		layer.User,
		u.Key,
		[]access.Action{access.ActionRetrieve},
		resource.ID{Type: resource.TypeChannel},
		resource.ID{Type: resource.TypeSchematic},
	); err != nil {
		b.Fatalf("failed to grant retrieve: %v", err)
	}
	return AuthedCtx(b.Context(), u)
}

// createUsedChannels creates 100 channels and 30 schematics of 10 value symbols each,
// so three schematics hold each channel, and returns the keys of the channels.
func createUsedChannels(b *testing.B, db *gorp.DB, layer *service.Layer) channel.Keys {
	b.Helper()
	ctx := b.Context()
	channels := make([]channel.Channel, 100)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:     fmt.Sprintf("bench_used_%d", i),
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	proj := project.Project{Name: "Bench"}
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
			return err
		}
		return layer.Channel.NewWriter(tx).CreateMany(ctx, &channels)
	}); err != nil {
		b.Fatalf("failed to create the channels: %v", err)
	}
	keys := channel.KeysFromChannels(channels)
	for i := range 30 {
		sch := schematic.Schematic{Name: fmt.Sprintf("bench_%d", i), Project: &proj.Key}
		for j := range 10 {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = keys[(i*10+j)%len(keys)]
			sch.Nodes = append(sch.Nodes, schematic.Node{
				Key:    fmt.Sprintf("value_%d", j),
				Config: schematic.NodeConfig{Variant: value},
			})
		}
		if err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		}); err != nil {
			b.Fatalf("failed to create a schematic: %v", err)
		}
	}
	return keys
}
