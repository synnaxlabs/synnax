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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
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
	v.NotNil("db", c.DB)
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

// folderKey is the KV key that holds the key of the folder of the metrics channels.
var folderKey = []byte("sy_metrics_folder")

// openFolder returns the folder of the metrics channels, found by its stored key so
// that a rename keeps it. When no folder is stored, or the stored one is gone, it finds
// or creates a top-level channel folder named Metrics, or the first free name after it
// when a channel holds that name, and stores its key.
func openFolder(ctx context.Context, cfg ServiceConfig) (group.Group, error) {
	var (
		g     group.Group
		found bool
	)
	if err := cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		b, closer, err := tx.Get(folderKey)
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		key, err := uuid.Parse(string(b))
		if err = errors.Combine(err, closer.Close()); err != nil {
			return err
		}
		err = cfg.Group.NewRetrieve().
			Where(group.MatchKeys(key)).
			Entry(&g).
			Exec(ctx, tx)
		found = err == nil
		return errors.Skip(err, query.ErrNotFound)
	}); err != nil || found {
		return g, err
	}
	g, err := createFolder(ctx, cfg.Group)
	if err != nil {
		return group.Group{}, err
	}
	return g, cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return tx.Set(ctx, folderKey, []byte(g.Key.String()))
	})
}

// folderName is the name of the folder of the metrics channels.
const folderName = "Metrics"

// createFolder finds or creates the top-level channel folder named folderName. When
// another record holds that name, it takes the first free name of the form Metrics_2,
// so a name never stops the Core from starting.
func createFolder(ctx context.Context, groups *group.Service) (group.Group, error) {
	name := folderName
	for n := 2; ; n++ {
		g, err := groups.CreateOrRetrieve(ctx, name, resource.TypeChannel)
		if !errors.Is(err, query.ErrUniqueViolation) {
			return g, err
		}
		name = fmt.Sprintf("%s_%d", folderName, n)
	}
}

// channelKeyPrefix prefixes the KV key that holds the key of a metrics channel, by the
// name that the service gives the channel.
const channelKeyPrefix = "sy_metrics_channel/"

// openChannels sets each of chs to its stored channel: the one whose key is stored for
// its name, so that a rename or a move keeps it. With no stored key, it takes the one
// channel of its name in folder, else the one channel of its name anywhere, as a store
// from before stored keys holds it. It creates the rest and stores the key of each.
func openChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels *channel.Service,
	folder group.Key,
	chs []channel.Channel,
) error {
	var (
		names   = channel.Names(chs)
		missing = make(map[string]int, len(chs))
		unfound = make([]string, 0, len(chs))
	)
	for i, ch := range chs {
		stored, found, err := storedChannel(ctx, tx, channels, ch.Name)
		if err != nil {
			return err
		}
		if found {
			chs[i] = stored
			continue
		}
		missing[ch.Name] = i
		unfound = append(unfound, ch.Name)
	}
	var held []channel.Channel
	if err := channels.NewRetrieve().
		Where(channel.MatchNames(unfound...)).
		Entries(&held).
		Exec(ctx, tx); err != nil {
		return err
	}
	holders := make(map[string][]channel.Channel, len(unfound))
	for _, ch := range held {
		holders[ch.Name] = append(holders[ch.Name], ch)
	}
	for name, hs := range holders {
		inFolder := lo.Filter(hs, func(ch channel.Channel, _ int) bool {
			return ch.Group != nil && *ch.Group == folder
		})
		if len(inFolder) == 0 {
			inFolder = hs
		}
		if len(inFolder) == 1 {
			chs[missing[name]] = inFolder[0]
			delete(missing, name)
		}
	}
	created := make([]channel.Channel, 0, len(missing))
	for _, ch := range chs {
		if _, ok := missing[ch.Name]; ok {
			created = append(created, ch)
		}
	}
	if err := channels.NewWriter(tx).CreateMany(ctx, &created); err != nil {
		return err
	}
	for _, ch := range created {
		chs[missing[ch.Name]] = ch
	}
	for i, ch := range chs {
		if err := tx.Set(
			ctx,
			[]byte(channelKeyPrefix+names[i]),
			[]byte(ch.Key().String()),
		); err != nil {
			return err
		}
	}
	return nil
}

// storedChannel returns the channel whose key is stored for name, and whether it
// exists.
func storedChannel(
	ctx context.Context,
	tx gorp.Tx,
	channels *channel.Service,
	name string,
) (channel.Channel, bool, error) {
	b, closer, err := tx.Get([]byte(channelKeyPrefix + name))
	if errors.Is(err, query.ErrNotFound) {
		return channel.Channel{}, false, nil
	}
	if err != nil {
		return channel.Channel{}, false, err
	}
	key, err := channel.ParseKey(string(b))
	if err = errors.Combine(err, closer.Close()); err != nil {
		return channel.Channel{}, false, err
	}
	var ch channel.Channel
	err = channels.NewRetrieve().Where(channel.MatchKeys(key)).Entry(&ch).Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return channel.Channel{}, false, nil
	}
	return ch, err == nil, err
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
	s := &Service{cfg: cfg, stopCollector: make(chan struct{})}
	if s.group, err = openFolder(ctx, cfg); err != nil {
		return nil, err
	}
	namePrefix := fmt.Sprintf("sy_node_%s_metrics_", cfg.HostKey)
	c := &collector{
		ins:      cfg.Child("collector"),
		interval: cfg.CollectionInterval,
		stop:     s.stopCollector,
	}
	idx := channel.Channel{
		Name:     namePrefix + "time",
		DataType: telem.TimestampT,
		IsIndex:  true,
		Group:    &s.group.Key,
	}
	var metricsChannels []channel.Channel
	if err = cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		indexes := []channel.Channel{idx}
		if err = openChannels(ctx, tx, cfg.Channel, s.group.Key, indexes); err != nil {
			return err
		}
		idx = indexes[0]
		metrics := s.createMetrics(namePrefix, idx.LocalKey)
		metricsChannels = lo.Map(metrics, func(m metric, _ int) channel.Channel {
			m.ch.Group = &s.group.Key
			return m.ch
		})
		if err = openChannels(
			ctx,
			tx,
			cfg.Channel,
			s.group.Key,
			metricsChannels,
		); err != nil {
			return err
		}
		for i, ch := range metricsChannels {
			metrics[i].ch = ch
		}
		c.metrics = metrics
		return nil
	}); err != nil {
		return nil, err
	}
	// Do this in a separate transaction otherwise the Arc analyzer won't parse the
	// calculated channel expressions.
	if err = cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		calculatedChannels := createCalculatedMetrics(namePrefix)
		for i := range calculatedChannels {
			calculatedChannels[i].Group = &s.group.Key
		}
		return openChannels(ctx, tx, cfg.Channel, s.group.Key, calculatedChannels)
	}); err != nil {
		return nil, err
	}
	w, err := cfg.Framer.NewStreamWriter(
		ctx,
		framer.WriterConfig{
			ControlSubject: control.Subject{
				Name: fmt.Sprintf("Node %v Metrics Writer", cfg.HostKey),
			},
			Keys:                     channel.KeysFromChannels(metricsChannels),
			AutoIndex:                new(true),
			AutoIndexPersistInterval: telem.Second * 30,
		},
	)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	p := plumber.New()
	p.SetSegment[framer.WriterRequest, framer.WriterResponse](writerAddr, w)
	p.SetSource[framer.WriterRequest](collectorAddr, c)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(_ context.Context, response framer.WriterResponse) {
		if response.Err != nil {
			cfg.L.Error("failed to write metrics to node", zap.Error(response.Err))
		}
	})
	p.SetSink[framer.WriterResponse](loggerAddr, o)
	p.MustConnect[framer.WriterRequest](collectorAddr, writerAddr, channelBufferSize)
	p.MustConnect[framer.WriterResponse](writerAddr, loggerAddr, channelBufferSize)
	s.shutdown = signal.NewGracefulShutdown(sCtx, cancel)
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return s, nil
}

// Close gracefully stops the service, waiting for all internal goroutines to exit.
func (s *Service) Close() error { close(s.stopCollector); return s.shutdown.Close() }
