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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// Service is the distribution-layer channel allocator. It owns the cluster-wide
// machinery for assigning local channel keys and creating, renaming, and deleting the
// underlying storage channels, routing each operation to the channel's leaseholder. It
// does NOT own channel metadata — the rich channel record, its gorp table, retrieval,
// and ontology integration live in the service layer, which drives this allocator.
type Service struct {
	cfg    ServiceConfig
	closer io.MultiCloser
	// leasedCounter and freeCounter drive local-key assignment for gateway-leased and
	// free-virtual channels respectively. freeCounter is only populated on the
	// bootstrapper node.
	leasedCounter *counter
	freeCounter   *counter
	createRouter  proxy.BatchFactory[Channel]
	renameRouter  proxy.BatchFactory[renameBatchEntry]
	keyRouter     proxy.BatchFactory[Key]
}

// ServiceConfig configures the distribution-layer channel allocator.
type ServiceConfig struct {
	alamos.Instrumentation
	// HostResolver resolves node keys to network addresses for cluster routing.
	HostResolver node.HostResolver
	// ClusterDB backs the local-key counters.
	ClusterDB *gorp.DB
	// TSChannel is the storage-layer time-series database where channel storage is
	// created.
	TSChannel *ts.DB
	// Transport routes allocation, rename, and delete operations to leaseholders.
	Transport Transport
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "cluster_db", c.ClusterDB)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "transport", c.Transport)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	return c
}

var DefaultServiceConfig = ServiceConfig{}

// OpenService opens the distribution-layer channel allocator.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:          cfg,
		createRouter: proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		keyRouter:    proxy.BatchFactory[Key]{Host: cfg.HostResolver.HostKey()},
		renameRouter: proxy.BatchFactory[renameBatchEntry]{Host: cfg.HostResolver.HostKey()},
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	leasedCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.leasedCounter")
	if s.leasedCounter, err = openCounter(ctx, cfg.ClusterDB, leasedCounterKey); !ok(err, nil) {
		return nil, err
	}
	if cfg.HostResolver.HostKey() == node.KeyBootstrapper {
		freeCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.counter.free")
		if s.freeCounter, err = openCounter(ctx, cfg.ClusterDB, freeCounterKey); !ok(err, nil) {
			return nil, err
		}
	}
	cfg.Transport.CreateServer().BindHandler(s.allocateHandler)
	cfg.Transport.DeleteServer().BindHandler(s.deleteHandler)
	cfg.Transport.RenameServer().BindHandler(s.renameHandler)
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

// Create assigns local keys to the provided new channels (those with a zero LocalKey)
// by routing to each channel's leaseholder to draw from that node's key counter, and
// creates the corresponding storage channels there. The returned channels carry their
// assigned keys, in the same order as the input. Free-virtual channels draw their keys
// from the bootstrapper. A channel with an unspecified (zero) leaseholder defaults to
// the host node.
func (s *Service) Create(ctx context.Context, channels []Channel) ([]Channel, error) {
	out := make([]Channel, len(channels))
	indicesByTarget := make(map[node.Key][]int)
	freeIndices := make([]int, 0)
	host := s.cfg.HostResolver.HostKey()
	for i, ch := range channels {
		out[i] = ch
		if out[i].Leaseholder == 0 {
			out[i].Leaseholder = host
		}
		lease := out[i].Lease()
		if lease.IsFree() {
			freeIndices = append(freeIndices, i)
		} else {
			indicesByTarget[lease] = append(indicesByTarget[lease], i)
		}
	}
	for target, indices := range indicesByTarget {
		if target == host {
			if err := s.allocateGateway(ctx, out, indices); err != nil {
				return nil, err
			}
			continue
		}
		if err := s.allocateRemote(ctx, target, out, indices); err != nil {
			return nil, err
		}
	}
	if len(freeIndices) > 0 {
		if host.IsBootstrapper() {
			if err := s.allocateFree(ctx, out, freeIndices); err != nil {
				return nil, err
			}
		} else if err := s.allocateRemote(ctx, node.KeyBootstrapper, out, freeIndices); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Delete deletes the storage channels for the provided keys, routing each key to its
// leaseholder. It does not touch channel metadata.
func (s *Service) Delete(ctx context.Context, keys Keys) error {
	batch := s.keyRouter.Batch(keys)
	for nodeKey, entries := range batch.Peers {
		if err := s.deleteRemote(ctx, nodeKey, entries); err != nil {
			return err
		}
	}
	gateway := append(Keys{}, batch.Gateway...)
	gateway = append(gateway, batch.Free...)
	if len(gateway) == 0 {
		return nil
	}
	return s.cfg.TSChannel.DeleteChannels(gateway.Storage())
}

// Rename renames the storage channels for the provided keys, routing each key to its
// leaseholder. Free-virtual channels have no persistent storage and are skipped.
func (s *Service) Rename(ctx context.Context, keys Keys, names []string) error {
	batch := s.renameRouter.Batch(newRenameBatch(keys, names))
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		if err := s.renameRemote(ctx, nodeKey, keys, names); err != nil {
			return err
		}
	}
	if len(batch.Gateway) == 0 {
		return nil
	}
	keys, names = unzipRenameBatch(batch.Gateway)
	return s.cfg.TSChannel.RenameChannels(ctx, keys.Storage(), names)
}
