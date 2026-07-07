// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures the distribution-layer channel allocator.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// HostResolver resolves node keys to network addresses for cluster routing.
	//
	// [REQUIRED]
	HostResolver node.HostResolver
	// KVReadWriter is the key-value store used to back the local-key counters.
	//
	// [REQUIRED]
	KVReadWriter kv.ReadWriter
	// TS is the storage-layer time-series database where channel storage is created.
	//
	// [REQUIRED]
	TS *ts.DB
	// Transport routes allocation, rename, and delete operations to leaseholders.
	//
	// [REQUIRED]
	Transport Transport
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "kv_read_writer", c.KVReadWriter)
	validate.NotNil(v, "ts", c.TS)
	validate.NotNil(v, "transport", c.Transport)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.KVReadWriter = override.Nil(c.KVReadWriter, other.KVReadWriter)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	return c
}

// Service is the distribution-layer channel allocator. It owns the cluster-wide
// machinery for assigning local channel keys and creating, renaming, and deleting the
// underlying storage channels, routing each operation to the channel's leaseholder.
type Service struct {
	cfg           ServiceConfig
	leasedCounter *counter
	freeCounter   *counter
	renameRouter  proxy.BatchFactory[renameBatchEntry]
	deleteRouter  proxy.BatchFactory[Key]
	// freeStorageMu serializes transient free-channel registrations in the local
	// storage engine so concurrent writer opens do not race on creation.
	freeStorageMu sync.Mutex
}

// NewService opens the distribution-layer channel service, which is responsible for
// assigning local keys and routing create, rename, and delete operations to the
// appropriate nodes.
func NewService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:          cfg,
		deleteRouter: proxy.BatchFactory[Key](cfg.HostResolver.HostKey()),
		renameRouter: proxy.BatchFactory[renameBatchEntry](cfg.HostResolver.HostKey()),
	}
	leasedCounterKey := []byte(
		cfg.HostResolver.HostKey().String() + ".distribution.channel.leasedCounter",
	)
	if s.leasedCounter, err = newCounter(
		ctx, cfg.KVReadWriter, leasedCounterKey,
	); err != nil {
		return nil, err
	}
	if cfg.HostResolver.HostKey() == node.KeyBootstrapper {
		freeCounterKey := []byte(
			cfg.HostResolver.HostKey().String() + ".distribution.channel.counter.free",
		)
		if s.freeCounter, err = newCounter(
			ctx, cfg.KVReadWriter, freeCounterKey,
		); err != nil {
			return nil, err
		}
	}
	cfg.Transport.CreateServer().BindHandler(s.createHandler)
	cfg.Transport.DeleteServer().BindHandler(s.deleteHandler)
	cfg.Transport.RenameServer().BindHandler(s.renameHandler)
	return s, nil
}
