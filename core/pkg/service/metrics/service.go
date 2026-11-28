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
	"runtime/pprof"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lifetime"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
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
	// HostKey is the key of the host node. The metrics channel names include it.
	//
	// [REQUIRED]
	HostKey node.Key
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
	// Group is used to create a metrics group for the node.
	//
	// [REQUIRED]
	Group *group.Service
	// Status reports a channel that the service renames to keep the name of its folder.
	//
	// [REQUIRED]
	Status *status.Service
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
	c.HostKey = override.Numeric(c.HostKey, other.HostKey)
	c.DB = override.Nil(c.DB, other.DB)
	c.CollectionInterval = override.Numeric(
		c.CollectionInterval,
		other.CollectionInterval,
	)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.Group = override.Nil(c.Group, other.Group)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.metrics")
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NonZero("host_key", c.HostKey)
	v.NotNil("storage", c.Storage)
	v.Positive("collection_interval", c.CollectionInterval)
	v.NotNil("group", c.Group)
	v.NotNil("status", c.Status)
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service is used to collect metrics from the host machine (cpu, memory, disk) and
// write them to channels.
type Service struct {
	cfg      ServiceConfig
	group    group.Group
	metrics  []metric
	channels []channel.Channel
	// guard orders Run against Close.
	guard lifetime.Guard
}

const (
	writerAddr        address.Address = "writer"
	collectorAddr     address.Address = "collector"
	loggerAddr        address.Address = "logger"
	channelBufferSize                 = 10
)

// folderName is the name of the top-level channel folder that holds the metrics
// channels.
const folderName = "Metrics"

// openFolder returns the top-level channel folder named folderName, and creates it when
// there is none. A top-level channel that holds the name yields first (yieldName).
func openFolder(ctx context.Context, cfg ServiceConfig) (group.Group, error) {
	g, err := cfg.Group.CreateOrRetrieve(ctx, folderName, resource.TypeChannel)
	if !errors.Is(err, query.ErrUniqueViolation) {
		return g, err
	}
	if err = yieldName(ctx, cfg); err != nil {
		return group.Group{}, err
	}
	return cfg.Group.CreateOrRetrieve(ctx, folderName, resource.TypeChannel)
}

// yieldName renames the top-level channel named folderName to the first free name of
// the form Metrics_2, and writes a warning status that names the new name.
func yieldName(ctx context.Context, cfg ServiceConfig) error {
	return cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var ch channel.Channel
		if err := cfg.Channel.NewRetrieve().
			Where(channel.MatchPaths(folderName)).
			Entry(&ch).
			Exec(ctx, tx); err != nil {
			return err
		}
		w := cfg.Channel.NewWriter(tx)
		for n := 2; ; n++ {
			name := fmt.Sprintf("%s_%d", folderName, n)
			err := w.Rename(ctx, ch.Key(), name, false)
			if errors.Is(err, query.ErrUniqueViolation) {
				continue
			}
			if err != nil {
				return err
			}
			s := status.Status[any]{
				Name:    name,
				Variant: status.VariantWarning,
				Message: "Renamed channel",
				Description: fmt.Sprintf(
					"The Core renamed channel %q to %q, because its metrics folder "+
						"keeps the top-level name %q. Update each reference to the "+
						"old name.",
					folderName,
					name,
					folderName,
				),
				Time: telem.Now(),
			}
			return cfg.Status.NewWriter(tx).Set(ctx, &s)
		}
	})
}

// OpenService opens a new metric.Service using the provided configuration. See the
// ServiceConfig struct for details on the required configuration values. If OpenService
// returns an error, the service is not safe to use. If OpenService succeeds, it must be
// shut down by calling Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.group, err = openFolder(ctx, cfg); err != nil {
		return nil, err
	}
	namePrefix := fmt.Sprintf("sy_node_%s_metrics_", cfg.HostKey)
	if s.metrics, s.channels, err = s.createChannels(ctx, namePrefix); err != nil {
		return nil, err
	}
	// Do this in a separate transaction otherwise the Arc analyzer won't parse the
	// calculated channel expressions.
	if err = s.createCalculatedChannels(ctx, namePrefix); err != nil {
		return nil, err
	}
	return s, nil
}

// createChannels finds or creates the index and metric channels. It returns the metrics
// bound to their stored channels.
func (s *Service) createChannels(
	ctx context.Context,
	namePrefix string,
) ([]metric, []channel.Channel, error) {
	var (
		metrics         []metric
		metricsChannels []channel.Channel
	)
	if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		idx := []channel.Channel{{
			Name:     namePrefix + "time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}
		if err := s.findOrCreate(ctx, tx, idx); err != nil {
			return err
		}
		metrics = s.createMetrics(namePrefix, idx[0].LocalKey)
		metricsChannels = lo.Map(metrics, func(m metric, _ int) channel.Channel {
			return m.ch
		})
		if err := s.findOrCreate(ctx, tx, metricsChannels); err != nil {
			return err
		}
		for i, ch := range metricsChannels {
			metrics[i].ch = ch
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return metrics, metricsChannels, nil
}

// createCalculatedChannels finds or creates the calculated metrics. A calculated metric
// that does not compile, such as one that reads a name that several channels hold, is
// a status and never stops the service.
func (s *Service) createCalculatedChannels(
	ctx context.Context,
	namePrefix string,
) error {
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.findOrCreate(ctx, tx, createCalculatedMetrics(namePrefix))
	})
	diags, ok := errors.AsType[*diagnostics.Diagnostics](err)
	if !ok {
		return err
	}
	st := status.Status[any]{
		Name:        "Calculated metrics",
		Variant:     status.VariantError,
		Message:     "Failed to create calculated metrics",
		Description: diags.Error(),
		Time:        telem.Now(),
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.cfg.Status.NewWriter(tx).Set(ctx, &st)
	})
}

// findOrCreate sets each of chs to the stored channel of its name
// (channel.RetrieveIfNameExists), and creates each other one in the folder of the
// metrics.
func (s *Service) findOrCreate(
	ctx context.Context,
	tx gorp.Tx,
	chs []channel.Channel,
) error {
	created := slices.Clone(chs)
	for i := range created {
		created[i].Group = &s.group.Key
	}
	if err := s.cfg.Channel.NewWriter(tx).CreateMany(
		ctx,
		&created,
		channel.RetrieveIfNameExists(),
	); err != nil {
		return err
	}
	copy(chs, created)
	return nil
}

// Run writes the host metrics on each CollectionInterval until ctx ends. It then
// flushes the metrics it collected and returns nil. A second Run, or a Run after Close,
// returns an error.
func (s *Service) Run(ctx context.Context) (err error) {
	if err = s.guard.Begin(); err != nil {
		return err
	}
	defer s.guard.End()
	pprof.Do(ctx, pprof.Labels("routine", "metrics"), func(ctx context.Context) {
		err = s.run(ctx)
	})
	return err
}

func (s *Service) run(ctx context.Context) error {
	// The writer outlives ctx, so that it writes the last metrics before it closes.
	sCtx, cancel := signal.WithCancel(
		context.WithoutCancel(ctx),
		signal.WithInstrumentation(s.cfg.Instrumentation),
	)
	defer cancel()
	w, err := s.cfg.Framer.NewStreamWriter(
		sCtx,
		framer.WriterConfig{
			ControlSubject: control.Subject{
				Name: fmt.Sprintf("Node %v Metrics Writer", s.cfg.HostKey),
			},
			Keys:                     channel.KeysFromChannels(s.channels),
			AutoIndex:                new(true),
			AutoIndexPersistInterval: telem.Second * 30,
		},
	)
	if err != nil {
		return err
	}
	c := &collector{
		ins:      s.cfg.Child("collector"),
		interval: s.cfg.CollectionInterval,
		metrics:  s.metrics,
		stop:     ctx.Done(),
	}
	p := plumber.New()
	p.SetSegment[framer.WriterRequest, framer.WriterResponse](writerAddr, w)
	p.SetSource[framer.WriterRequest](collectorAddr, c)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(_ context.Context, response framer.WriterResponse) {
		if response.Err != nil {
			s.cfg.L.Error("failed to write metrics to node", zap.Error(response.Err))
		}
	})
	p.SetSink[framer.WriterResponse](loggerAddr, o)
	p.MustConnect[framer.WriterRequest](collectorAddr, writerAddr, channelBufferSize)
	p.MustConnect[framer.WriterResponse](writerAddr, loggerAddr, channelBufferSize)
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return sCtx.Wait()
}

// Close releases the resources of the Service. Call it after Run returns: while Run
// runs, Close returns an error.
func (s *Service) Close() error {
	_, err := s.guard.Close()
	return err
}
