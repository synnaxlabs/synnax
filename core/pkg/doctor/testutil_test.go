// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor_test

import (
	"bytes"
	"context"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	aspenmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	. "github.com/synnaxlabs/x/testutil"
)

// core is a Core's stores and services, opened against a real data directory the way a
// Core opens them. Specs write through it, close it, then run the doctor over the
// directory.
type core struct {
	// Dirname is the data directory the stores live in.
	Dirname string
	// Layer is the distribution layer, holding the gorp DB and the cluster.
	*distribution.Layer
	// Storage is the storage layer underneath.
	Storage *storage.Layer
	// Channels writes channels to both stores.
	Channels *channel.Service
	// Ontology holds the resource graph.
	Ontology *ontology.Ontology
	// Group creates resource groups.
	Group *group.Service
	// closer shuts the services down before the layers they read through.
	closer xio.MultiCloser
}

// Close shuts down every service and layer, releasing the lock the doctor needs.
func (c *core) Close() error {
	return errors.Join(c.closer.Close(), c.Layer.Close(), c.Storage.Close())
}

// createCore opens a Core's stores and services against a fresh data directory. The
// directory is removed when the spec ends. The caller closes the core before running
// the doctor: an open Core holds the key-value store's lock.
func createCore(ctx context.Context) *core {
	GinkgoHelper()
	dir := MustSucceed(os.MkdirTemp("", "doctor-test-*"))
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	storageLayer := MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
		Dirname: dir,
	}))
	addr := address.Address("localhost:0")
	dist := MustSucceed(distribution.OpenLayer(ctx, distribution.LayerConfig{
		Storage:          storageLayer,
		Transport:        distmock.NewNetwork().New(addr, 1),
		AspenTransport:   aspenmock.NewNetwork().NewTransport(),
		AdvertiseAddress: addr,
		AspenOptions: []aspen.Option{
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		},
	}))
	c := &core{Dirname: dir, Layer: dist, Storage: storageLayer}
	otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: dist.DB}))
	index := MustSucceed(search.OpenIndex())
	groupSvc := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       dist.DB,
		Ontology: otg,
		Search:   index,
	}))
	labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   index,
	}))
	statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
		Search:   index,
	}))
	c.Channels = MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostProvider: dist.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       index,
		Status:       statusSvc,
	}))
	c.Ontology, c.Group = otg, groupSvc
	c.closer = xio.MultiCloser{
		c.Channels,
		statusSvc,
		labelSvc,
		groupSvc,
		otg,
		index,
	}
	return c
}

// createChannel writes one channel through the service, so both stores see it.
func createChannel(ctx context.Context, c *core, ch channel.Channel) channel.Channel {
	GinkgoHelper()
	Expect(c.Channels.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

// run closes the core and inspects its data directory.
func run(ctx context.Context, c *core, cfgs ...doctor.Config) doctor.Report {
	GinkgoHelper()
	Expect(c.Close()).To(Succeed())
	cfg := doctor.Config{Dirname: c.Dirname}
	return MustSucceed(doctor.Run(ctx, append([]doctor.Config{cfg}, cfgs...)...))
}

// findings returns every finding the report holds for the given check.
func findings(r doctor.Report, check doctor.Check) []doctor.Finding {
	matched := make([]doctor.Finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Check == check {
			matched = append(matched, f)
		}
	}
	return matched
}

var _ io.Closer = (*core)(nil)

// write stores entries straight through gorp, bypassing the service that normally
// maintains them. Specs use it to plant the broken references the doctor looks for.
func write[K gorp.Key, E gorp.Entry[K]](ctx context.Context, c *core, entries ...E) {
	GinkgoHelper()
	Expect(gorp.WrapWriter[K, E](c.DB).Set(ctx, entries...)).To(Succeed())
}

// messages returns the message of every finding for the given check.
func messages(r doctor.Report, check doctor.Check) []string {
	found := findings(r, check)
	out := make([]string, len(found))
	for i, f := range found {
		out[i] = f.Message
	}
	return out
}

// readEntry returns the raw value stored under the given key.
func readEntry(ctx context.Context, c *core, key string) []byte {
	GinkgoHelper()
	value, closer := MustSucceed2(c.Storage.KV.Get(ctx, []byte(key)))
	out := bytes.Clone(value)
	Expect(closer.Close()).To(Succeed())
	return out
}

// deleteEntry removes one entry through gorp, leaving whatever referenced it behind.
func deleteEntry[K gorp.Key, E gorp.Entry[K]](ctx context.Context, c *core, key K) {
	GinkgoHelper()
	Expect(gorp.WrapWriter[K, E](c.DB).Delete(ctx, key)).To(Succeed())
}
