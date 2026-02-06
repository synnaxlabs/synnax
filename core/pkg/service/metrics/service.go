// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type ServiceConfig struct {
	// DB is the database used to open a transaction for the service.
	//
	// [REQUIRED]
	DB *gorp.DB
	// HostProvider is for identify the current host for channel naming.
	//
	// [REQUIRED]
	HostProvider cluster.HostProvider
	// Channel is used to create and retrieve metric collection channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer is used to write metrics to the metric channels.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Storage is the storage layer used for disk usage metrics.
	//
	// [REQUIRED]
	Storage *storage.Layer
	// Ontology is used to create relationships between metrics and other entities in
	// the Synnax resource graph.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create a metrics group for the node.
	//
	// [REQUIRED]
	Group *group.Service
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// CollectionInterval sets the interval at which metrics will be collected from the
	// host machine.
	//
	// [OPTIONAL] - Defaults to 2s
	CollectionInterval time.Duration
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for a metrics service.
	DefaultServiceConfig = ServiceConfig{CollectionInterval: 2 * time.Second}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.DB = override.Nil(c.DB, other.DB)
	c.CollectionInterval = override.Numeric(
		c.CollectionInterval,
		other.CollectionInterval,
	)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.Group = override.Nil(c.Group, other.Group)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.metrics")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "host_provider", c.HostProvider)
	validate.NotNil(v, "storage", c.Storage)
	validate.Positive(v, "collection_interval", c.CollectionInterval)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Service is used to collect metrics from the host machine (cpu, memory, disk) and
// write them to channels.
type Service struct {
	shutdown      io.Closer
	stopCollector chan struct{}
	cfg           ServiceConfig
	group         group.Group
}

const (
	writerAddr        address.Address = "writer"
	collectorAddr     address.Address = "collector"
	loggerAddr        address.Address = "logger"
	channelBufferSize                 = 10
)

// OpenService opens a new metric.Service using the provided configuration. See the
// ServiceConfig struct for details on the required configuration values. If OpenService
// returns an error, the service is not safe to use. If OpenService succeeds, it must be
// shut down by calling Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, stopCollector: make(chan struct{})}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx, "Metrics", cfg.Channel.Group().OntologyID(),
	); err != nil {
		return nil, err
	}
	namePrefix := fmt.Sprintf("sy_node_%s_metrics_", cfg.HostProvider.HostKey())
	c := &collector{
		ins:      cfg.Child("collector"),
		interval: cfg.CollectionInterval,
		stop:     s.stopCollector,
	}
	c.idx = channel.Channel{
		Name:     namePrefix + "time",
		DataType: telem.TimeStampT,
		IsIndex:  true,
	}
	var metricsChannels []channel.Channel
	if err := cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		chWriter := cfg.Channel.NewWriter(tx)
		if err := chWriter.Create(
			ctx,
			&c.idx,
			channel.RetrieveIfNameExists(),
			channel.CreateWithoutGroupRelationship(),
		); err != nil {
			return err
		}
		otgWriter := cfg.Ontology.NewWriter(tx)
		if err := s.maybeDefineGroupRelationship(
			ctx,
			tx,
			[]ontology.ID{c.idx.OntologyID()},
		); err != nil {
			return err
		}
		metrics := s.createMetrics(namePrefix, c.idx.LocalKey)
		metricsChannels = lo.Map(metrics, func(m metric, _ int) channel.Channel {
			return m.ch
		})
		if err := chWriter.CreateMany(
			ctx,
			&metricsChannels,
			channel.RetrieveIfNameExists(),
			channel.CreateWithoutGroupRelationship(),
		); err != nil {
			return err
		}
		// delete any existing relationships between the parent Channels group and the
		// metrics channels
		for _, ch := range metricsChannels {
			if err := otgWriter.DeleteRelationship(
				ctx,
				cfg.Channel.Group().OntologyID(),
				ontology.RelationshipTypeParentOf,
				ch.OntologyID(),
			); err != nil {
				return err
			}
		}
		for i, ch := range metricsChannels {
			metrics[i].ch = ch
		}
		c.metrics = metrics
		if err := s.maybeDefineGroupRelationship(
			ctx,
			tx,
			channel.OntologyIDsFromChannels(metricsChannels),
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}
	// Do this in a separate transaction otherwise the Arc analyzer won't parse the
	// calculated channel expressions.
	if err := cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		calculatedChannels := createCalculatedMetrics(namePrefix)
		if err := cfg.Channel.NewWriter(tx).CreateMany(
			ctx,
			&calculatedChannels,
			channel.RetrieveIfNameExists(),
			channel.CreateWithoutGroupRelationship(),
		); err != nil {
			return err
		}
		if err := s.maybeDefineGroupRelationship(
			ctx,
			tx,
			channel.OntologyIDsFromChannels(calculatedChannels),
		); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	w, err := cfg.Framer.NewStreamWriter(
		ctx,
		framer.WriterConfig{
			Keys: append(
				channel.KeysFromChannels(metricsChannels),
				c.idx.Key(),
			),
			Start:                    telem.Now(),
			AutoIndexPersistInterval: telem.Second * 30,
		},
	)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	p := plumber.New()
	plumber.SetSegment[framer.WriterRequest, framer.WriterResponse](p, writerAddr, w)
	plumber.SetSource[framer.WriterRequest](p, collectorAddr, c)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(_ context.Context, response framer.WriterResponse) {
		if response.Err != nil {
			cfg.L.Error("failed to write metrics to node", zap.Error(response.Err))
		}
	})
	plumber.SetSink[framer.WriterResponse](p, loggerAddr, o)
	plumber.MustConnect[framer.WriterRequest](
		p,
		collectorAddr,
		writerAddr,
		channelBufferSize,
	)
	plumber.MustConnect[framer.WriterResponse](
		p,
		writerAddr,
		loggerAddr,
		channelBufferSize,
	)
	s.shutdown = signal.NewGracefulShutdown(sCtx, cancel)
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return s, nil
}

// Close gracefully stops the service, waiting for all internal goroutines to exit.
func (s *Service) Close() error { close(s.stopCollector); return s.shutdown.Close() }

// maybeDefineGroupRelationship checks each channel for an existing parent group
// relationship. If a channel already has a parent group, it is skipped. If a channel
// doesn't have a parent group, it is attached to the service's metrics group.
func (s *Service) maybeDefineGroupRelationship(
	ctx context.Context,
	tx gorp.Tx,
	channelIDs []ontology.ID,
) error {
	otgWriter := s.cfg.Ontology.NewWriter(tx)
	for _, chID := range channelIDs {
		var parents []ontology.Resource
		if err := s.cfg.Ontology.NewRetrieve().
			WhereIDs(chID).
			TraverseTo(ontology.ParentsTraverser).
			WhereTypes(group.OntologyType).
			Entries(&parents).
			Exec(ctx, tx); err != nil {
			return err
		}
		if len(parents) > 0 {
			continue
		}
		if err := otgWriter.DefineRelationship(
			ctx,
			s.group.OntologyID(),
			ontology.RelationshipTypeParentOf,
			chID,
		); err != nil {
			return err
		}
	}
	return nil
}
