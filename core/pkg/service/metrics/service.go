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
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
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
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
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
	// Calculation stores the calculation of each calculated metric channel.
	//
	// [REQUIRED]
	Calculation *calculation.Service
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
	// Status holds the warning of the metrics that a name keeps from opening.
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
	c.Calculation = override.Nil(c.Calculation, other.Calculation)
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
	v.NotNil("calculation", c.Calculation)
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

// createChannel returns the create of openChannels for a metric that the service
// writes.
func createChannel(
	cfg ServiceConfig,
) func(context.Context, gorp.Tx, *channel.Channel) error {
	return func(ctx context.Context, tx gorp.Tx, ch *channel.Channel) error {
		return cfg.Channel.NewWriter(tx).Create(ctx, ch)
	}
}

// createOutput returns the create of openChannels for the outputs of calculated: it
// creates the calculation of each output, and the output with it.
func createOutput(
	cfg ServiceConfig,
	calculated []calculatedMetric,
) func(context.Context, gorp.Tx, *channel.Channel) error {
	return func(ctx context.Context, tx gorp.Tx, ch *channel.Channel) error {
		i := slices.IndexFunc(calculated, func(m calculatedMetric) bool {
			return m.name == ch.Name
		})
		c := calculation.Calculation{Expression: calculated[i].expression}
		made, err := cfg.Calculation.NewWriter(tx).
			CreateOutput(ctx, &c, ch.Name, ch.Group)
		if err != nil {
			return err
		}
		*ch = made[0]
		return nil
	}
}

// channelKeyPrefix prefixes the KV key that holds the key of a metrics channel, by the
// name that the service gives the channel.
const channelKeyPrefix = "sy_metrics_channel/"

// openChannels sets each of chs to its stored channel: the one whose key is stored for
// its name, so that a rename or a move keeps it. With no stored key, it takes the one
// channel of its name in folder, else the one channel of its name anywhere, as a store
// from before stored keys holds it. It creates the rest, each in a transaction of its
// own with create, and stores the key of each. A name never fails a start: a channel
// that a name keeps from being found or created is left out, and off holds its position
// and a channel.AmbiguousNameError for the name.
func openChannels(
	ctx context.Context,
	cfg ServiceConfig,
	folder group.Key,
	chs []channel.Channel,
	create func(context.Context, gorp.Tx, *channel.Channel) error,
) (off map[int]error, err error) {
	names := channel.Names(chs)
	missing, holders, err := findChannels(ctx, cfg, chs)
	if err != nil {
		return nil, err
	}
	adoptChannels(chs, folder, missing, holders)
	off = make(map[int]error)
	for i, name := range names {
		if _, ok := missing[name]; !ok {
			continue
		}
		err = cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return create(ctx, tx, &chs[i])
		})
		if blocked, isBlocked := nameBlock(err, name, holders[name]); isBlocked {
			off[i] = blocked
		} else if err != nil {
			return nil, err
		}
	}
	return off, cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		for i, ch := range chs {
			if _, isOff := off[i]; isOff {
				continue
			}
			if err := tx.Set(
				ctx,
				[]byte(channelKeyPrefix+names[i]),
				[]byte(ch.Key().String()),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// nameBlock returns the warning of a create of the channel named name that failed
// with err, and whether a name blocked it: an ambiguous name, or a record that holds
// the name. holders are the channels that hold the name.
func nameBlock(err error, name string, holders []channel.Channel) (error, bool) {
	if ambiguous, ok := errors.AsType[channel.AmbiguousNameError](err); ok {
		return ambiguous, true
	}
	clash, ok := errors.AsType[gorp.UniqueError](err)
	if !ok {
		return nil, false
	}
	if len(holders) > 1 {
		return channel.AmbiguousNameError{
			Name:    name,
			Holders: channel.KeysFromChannels(holders),
		}, true
	}
	return errors.Newf(
		"%s %s holds the name '%s'",
		clash.Holder.Table,
		clash.Holder.Key,
		name,
	), true
}

// findChannels sets each of chs whose key is stored to its stored channel. It returns
// the position of each other channel by name, and the channels that hold each of those
// names.
func findChannels(
	ctx context.Context,
	cfg ServiceConfig,
	chs []channel.Channel,
) (map[string]int, map[string][]channel.Channel, error) {
	var (
		missing = make(map[string]int, len(chs))
		holders = make(map[string][]channel.Channel, len(chs))
	)
	return missing, holders, cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		unfound := make([]string, 0, len(chs))
		for i, ch := range chs {
			stored, keyStored, exists, err := storedChannel(
				ctx,
				tx,
				cfg.Channel,
				ch.Name,
			)
			if err != nil {
				return err
			}
			if exists {
				chs[i] = stored
				continue
			}
			missing[ch.Name] = i
			// Only a store from before stored keys looks a channel up by name.
			if !keyStored {
				unfound = append(unfound, ch.Name)
			}
		}
		var held []channel.Channel
		if err := cfg.Channel.NewRetrieve().
			Where(channel.MatchNames(unfound...)).
			Entries(&held).
			Exec(ctx, tx); err != nil {
			return err
		}
		for _, ch := range held {
			holders[ch.Name] = append(holders[ch.Name], ch)
		}
		return nil
	})
}

// adoptChannels sets the channel at each position in missing to the one holder of its
// name in folder, else to the one holder of its name anywhere, and removes the name
// from missing.
func adoptChannels(
	chs []channel.Channel,
	folder group.Key,
	missing map[string]int,
	holders map[string][]channel.Channel,
) {
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
}

// storedChannel returns the channel whose key is stored for name, whether a key is
// stored, and whether its channel exists.
func storedChannel(
	ctx context.Context,
	tx gorp.Tx,
	channels *channel.Service,
	name string,
) (ch channel.Channel, stored, exists bool, err error) {
	b, closer, err := tx.Get([]byte(channelKeyPrefix + name))
	if errors.Is(err, query.ErrNotFound) {
		return channel.Channel{}, false, false, nil
	}
	if err != nil {
		return channel.Channel{}, false, false, err
	}
	key, err := channel.ParseKey(string(b))
	if err = errors.Combine(err, closer.Close()); err != nil {
		return channel.Channel{}, true, false, err
	}
	err = channels.NewRetrieve().Where(channel.MatchKeys(key)).Entry(&ch).Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return channel.Channel{}, true, false, nil
	}
	return ch, true, err == nil, err
}

// offErrors returns the errors of off at positions below n, in order.
func offErrors(off map[int]error, n int) []error {
	errs := make([]error, 0, len(off))
	for i := range n {
		if err, isOff := off[i]; isOff {
			errs = append(errs, err)
		}
	}
	return errs
}

// statusKey returns the key of the status of the metrics of the host.
func statusKey(host node.Key) status.Key {
	return fmt.Sprintf("sy_node_%s_metrics", host)
}

// report sets a warning status that names each of warnings when there is one, and
// deletes the status otherwise.
func (s *Service) report(ctx context.Context, warnings []error) error {
	key := statusKey(s.cfg.HostKey)
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.cfg.Status.NewWriter(tx)
		if len(warnings) == 0 {
			return errors.Skip(w.Delete(ctx, key), query.ErrNotFound)
		}
		texts := lo.Map(warnings, func(err error, _ int) string { return err.Error() })
		return w.Set(ctx, &status.Status[struct{}]{
			Key:     key,
			Name:    fmt.Sprintf("Node %s metrics", s.cfg.HostKey),
			Variant: status.VariantWarning,
			Message: "Metrics left off",
			Description: fmt.Sprintf(
				"%s. Rename or delete all but one channel of each name, then "+
					"restart the Core.",
				strings.Join(texts, ". "),
			),
			Time: telem.Now(),
		})
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
	idx := []channel.Channel{{
		Name:     namePrefix + "time",
		DataType: telem.TimestampT,
		IsIndex:  true,
		Group:    &s.group.Key,
	}}
	off, err := openChannels(ctx, cfg, s.group.Key, idx, createChannel(cfg))
	if err != nil {
		return nil, err
	}
	warnings := offErrors(off, len(idx))
	var metricsChannels []channel.Channel
	if len(off) == 0 {
		metrics := s.createMetrics(namePrefix, idx[0].LocalKey)
		chs := lo.Map(metrics, func(m metric, _ int) channel.Channel {
			m.ch.Group = &s.group.Key
			return m.ch
		})
		if off, err = openChannels(
			ctx,
			cfg,
			s.group.Key,
			chs,
			createChannel(cfg),
		); err != nil {
			return nil, err
		}
		warnings = append(warnings, offErrors(off, len(chs))...)
		opened := make(map[string]channel.Channel, len(metrics))
		for i, m := range metrics {
			if _, isOff := off[i]; !isOff {
				opened[m.ch.Name] = chs[i]
				m.ch = chs[i]
				c.metrics = append(c.metrics, m)
				metricsChannels = append(metricsChannels, chs[i])
			}
		}
		calculated := createCalculatedMetrics(namePrefix, opened)
		outputs := lo.Map(calculated, func(m calculatedMetric, _ int) channel.Channel {
			return channel.Channel{Name: m.name, Group: &s.group.Key}
		})
		if off, err = openChannels(
			ctx,
			cfg,
			s.group.Key,
			outputs,
			createOutput(cfg, calculated),
		); err != nil {
			return nil, err
		}
		warnings = append(warnings, offErrors(off, len(outputs))...)
	}
	if err = s.report(ctx, warnings); err != nil {
		return nil, err
	}
	if len(metricsChannels) == 0 {
		s.shutdown = xio.NoFailCloserFunc(func() {})
		return s, nil
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
