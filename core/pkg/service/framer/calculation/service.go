// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	channelgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// legacyStatusChannels are channels used in previous versions to store calculation
// status updates before the status service was introduced.
var legacyStatusChannels = []string{"sy_calculation_status", "sy_calculation_state"}

// Status is an alias for the calculation status type.
type Status = calculation.Status

// ServiceConfig is the configuration for opening the calculation service.
type ServiceConfig struct {
	// DB holds the channels that the service reads, and stores statuses.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Framer is the underlying frame service used to stream cached channel values.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Writer is the service-layer writer service used to open the writers that write
	// calculated samples back to the cluster.
	//
	// [REQUIRED]
	Writer *writer.Service
	// Channel is used to retrieve information about the channels being calculated and
	// to resolve channel symbols for Arc expression compilation.
	//
	// [REQUIRED]
	Channel *channel.Service
	// ChannelGraph reconciles calculated channel definitions. The service subscribes to
	// it for channel changes and reports calculation statuses through it.
	//
	// [REQUIRED]
	ChannelGraph *channelgraph.Graph
	// RetryDelay is the wait before the service compiles a changed calculation again
	// when the channels it reads changed during each compile.
	//
	// [OPTIONAL] - Defaults to 100 milliseconds.
	RetryDelay telem.TimeSpan
	// Compile compiles a calculated channel. It is injectable for testing.
	//
	// [OPTIONAL] - Defaults to compiler.Compile.
	Compile func(context.Context, *gorp.DB, ...compiler.Config) (compiler.Module, error)
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculate")
	v.NotNil("db", c.DB)
	v.NotNil("framer", c.Framer)
	v.NotNil("writer", c.Writer)
	v.NotNil("channel", c.Channel)
	v.NotNil("channel_graph", c.ChannelGraph)
	v.Positive("retry_delay", c.RetryDelay)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Writer = override.Nil(c.Writer, other.Writer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelGraph = override.Nil(c.ChannelGraph, other.ChannelGraph)
	c.RetryDelay = override.Numeric(c.RetryDelay, other.RetryDelay)
	c.Compile = override.Nil(c.Compile, other.Compile)
	return c
}

type Service struct {
	disconnectFromChannelChanges observe.Disconnect
	cfg                          ServiceConfig
	// retries compiles a changed calculation again after the channels it reads changed
	// during each compile.
	retries *debounce.Keyed[channel.Key]
	mu      struct {
		graph  *graph.Graph
		groups map[int]*group
		// pending counts the requests for each channel whose add waits on a retry.
		pending map[channel.Key]int
		sync.Mutex
	}
}

// OpenService opens the service with the provided configuration. The service must be
// closed when it is no longer needed.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(
		ServiceConfig{RetryDelay: 100 * telem.Millisecond},
		cfgs...,
	)
	if err != nil {
		return nil, err
	}
	g, err := graph.New(graph.Config{
		Instrumentation: cfg.Child("calculation.graph"),
		Channel:         cfg.Channel,
		DB:              cfg.DB,
		Compile:         cfg.Compile,
	})
	if err != nil {
		return nil, err
	}

	s := &Service{cfg: cfg}
	if s.retries, err = debounce.NewKeyed(debounce.KeyedConfig[channel.Key]{
		Delay:    cfg.RetryDelay.Duration(),
		MaxDelay: cfg.RetryDelay.Duration() * 4,
		Callback: s.retry,
	}); err != nil {
		return nil, err
	}
	s.mu.graph = g
	s.mu.groups = make(map[int]*group)
	s.mu.pending = make(map[channel.Key]int)
	s.disconnectFromChannelChanges = cfg.ChannelGraph.Observe().OnChange(s.handleChange)

	if err := cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return cfg.Channel.NewWriter(tx).DeleteManyByNames(
			ctx,
			legacyStatusChannels,
			true,
		)
	}); err != nil {
		cfg.L.Debug("failed to delete legacy status channels", zap.Error(err))
	}

	s.cfg.L.Info("calculation service initialized")

	return s, nil
}

// report saves a status a calculator reports for the channel with the given key.
func (s *Service) report(ctx context.Context, key channel.Key, st Status) bool {
	st.Time = telem.Now()
	if err := s.cfg.ChannelGraph.SetRuntimeStatus(ctx, key, &st); err != nil {
		s.cfg.L.Error(
			"failed to set status",
			zap.Error(err),
			zap.Stringer("channel", key),
		)
		return false
	}
	return true
}

// rejectDefinition reports that the Service cannot compile or request ch.
func (s *Service) rejectDefinition(
	ctx context.Context,
	ch channel.Channel,
	message string,
	err error,
) {
	st := &Status{
		Name:        ch.Name,
		Variant:     status.VariantError,
		Message:     message,
		Description: err.Error(),
		Time:        telem.Now(),
	}
	s.cfg.L.Warn(st.String())
	if err := s.cfg.ChannelGraph.RejectDefinition(ctx, ch.Key(), st); err != nil {
		s.cfg.L.Error(
			"failed to set status",
			zap.Error(err),
			zap.Stringer("channel", ch.Key()),
		)
	}
}

func (s *Service) clearStatus(ctx context.Context, key channel.Key) {
	if err := s.cfg.ChannelGraph.ClearRuntimeStatus(ctx, key); err != nil {
		s.cfg.L.Error(
			"failed to clear status",
			zap.Error(err),
			zap.Stringer("channel", key),
		)
	}
}

func (s *Service) acceptDefinition(ctx context.Context, key channel.Key) {
	if err := s.cfg.ChannelGraph.AcceptDefinition(ctx, key); err != nil {
		s.cfg.L.Error(
			"failed to clear status",
			zap.Error(err),
			zap.Stringer("channel", key),
		)
	}
}

func (s *Service) handleChange(ctx context.Context, reader channelgraph.Changes) {
	for cg := range reader {
		// Don't stop calculating if the channel is deleted. The calculation will be
		// automatically shut down when it is no longer needed.
		if cg.Kind == gorp.ChangeDelete || !cg.Value.IsCalculated() {
			continue
		}
		s.update(ctx, cg.Value)
	}
}

// update compiles ch again when the service calculates it. When the channels that ch
// reads changed during each compile, update keeps the running calculation and
// schedules a retry.
func (s *Service) update(ctx context.Context, ch channel.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateLocked(ctx, ch)
}

func (s *Service) updateLocked(ctx context.Context, ch channel.Channel) {
	if !s.mu.graph.Contains(ch.Key()) {
		return
	}
	err := s.updateCalculation(ctx, ch)
	switch {
	case err == nil:
		s.acceptDefinition(ctx, ch.Key())
	case errors.Is(err, channel.ErrLookupsChanged):
		s.retries.Trigger(ch.Key())
	// A cancelled compile says nothing about the definition of ch.
	case ctx.Err() == nil:
		s.rejectDefinition(ctx, ch, "Failed to compile calculation", err)
	}
}

// request adds a request for ch to the graph. When the channels that ch reads changed
// during each compile, request holds the request as pending and schedules a retry.
func (s *Service) request(ctx context.Context, ch channel.Channel) {
	err := s.mu.graph.Add(ctx, ch)
	if errors.Is(err, channel.ErrLookupsChanged) {
		s.mu.pending[ch.Key()]++
		s.retries.Trigger(ch.Key())
		return
	}
	if err != nil && ctx.Err() == nil {
		s.rejectDefinition(ctx, ch, "Failed to request calculation", err)
	}
}

// start clears the status of each calculation not in running and starts the groups of
// the graph.
func (s *Service) start(ctx context.Context, running set.Set[channel.Key]) error {
	// A calculation that already runs owns its status, so only a new one clears it.
	// Dependencies start with the channel that reads them.
	for k := range s.mu.graph.CalculatedKeys() {
		if !running.Contains(k) {
			s.clearStatus(ctx, k)
		}
	}
	return s.rebuildGroups(ctx)
}

// retry runs update on the stored definition of the channel with key, then adds its
// pending requests. It runs on the retry debouncer, so the service holds no lock while
// a retry waits.
func (s *Service) retry(ctx context.Context, key channel.Key) {
	// A change handler installs a newer definition under the lock, so a read under the
	// lock cannot undo it.
	s.mu.Lock()
	defer s.mu.Unlock()
	var ch channel.Channel
	err := s.cfg.Channel.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, s.cfg.DB)
	if err != nil {
		if !errors.Is(err, query.ErrNotFound) && ctx.Err() == nil {
			s.cfg.L.Error("failed to retry calculation",
				zap.Error(err), zap.Stringer("channel", key))
			s.retries.Trigger(key)
		}
		return
	}
	if !ch.IsCalculated() {
		return
	}
	// Another request can add the channel while its own request is pending, so the
	// retry can be for an update and a request at once.
	s.updateLocked(ctx, ch)
	pending := s.mu.pending[key]
	if pending == 0 {
		return
	}
	delete(s.mu.pending, key)
	running := s.mu.graph.CalculatedKeys()
	for range pending {
		s.request(ctx, ch)
	}
	if err := s.start(ctx, running); err != nil {
		s.cfg.L.Error("failed to start calculation",
			zap.Error(err), zap.Stringer("channel", key))
	}
}

func (s *Service) updateCalculation(ctx context.Context, ch channel.Channel) error {
	s.cfg.L.Debug("updating calculation",
		zap.String("channel", ch.Key().String()),
		zap.String("reason", "channel definition changed"),
	)
	if err := s.mu.graph.Update(ctx, ch); err != nil {
		return err
	}
	return s.rebuildGroups(ctx)
}

func (s *Service) openCalculator(
	ctx context.Context,
	mod compiler.Module,
) (*calculator.Calculator, error) {
	failed, err := s.cfg.ChannelGraph.Failed(ctx, mod.Channel.Key())
	if err != nil {
		return nil, err
	}
	calc, err := calculator.Open(ctx, calculator.Config{Module: mod, Failed: failed})
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to open calculator for channel %s", mod.Channel,
		)
	}
	return calc, nil
}

func groupEquals(mods []compiler.Module, g *group) bool {
	if g == nil {
		return false
	}
	if len(mods) != len(g.Calculators) {
		return false
	}
	for i, m := range mods {
		if !m.Channel.Equals(g.Calculators[i].Channel(), "Name") {
			return false
		}
	}
	return true
}

func (s *Service) updateGroup(
	ctx context.Context,
	key int,
	mods []compiler.Module,
) error {
	g := s.mu.groups[key]
	if groupEquals(mods, g) {
		return nil
	}
	if g != nil {
		s.cfg.L.Info("group stopping",
			zap.Int("group_id", key),
			zap.String("reason", "group composition changed"),
		)
		if err := g.Close(); err != nil {
			return err
		}
	}
	calculators := make([]*calculator.Calculator, len(mods))
	for i, m := range mods {
		calc, err := s.openCalculator(ctx, m)
		if err != nil {
			return err
		}
		calculators[i] = calc
	}
	g, err := openGroup(
		ctx,
		groupConfig{
			Instrumentation: s.cfg.Child("group"),
			calculators:     calculators,
			report:          s.report,
			framer:          s.cfg.Framer,
			writer:          s.cfg.Writer,
		},
	)
	if err != nil {
		return err
	}
	s.mu.groups[key] = g
	s.cfg.L.Info("group started",
		zap.Int("group_id", key),
		zap.Int("calculator_count", len(calculators)),
		zap.Stringers("calculators", calculators),
	)
	return nil
}

func (s *Service) rebuildGroups(ctx context.Context) error {
	groups := s.mu.graph.CalculateGrouped()
	s.cfg.L.Debug("rebuilding groups",
		zap.Int("new_group_count", len(groups)),
		zap.Int("current_group_count", len(s.mu.groups)),
	)
	for k, g := range s.mu.groups {
		if _, ok := groups[k]; !ok {
			s.cfg.L.Info("group stopping",
				zap.Int("group_id", k),
				zap.String("reason", "no longer in group allocation"),
			)
			delete(s.mu.groups, k)
			if err := g.Close(); err != nil {
				return err
			}
		}
	}
	for k, mods := range groups {
		if err := s.updateGroup(ctx, k, mods); err != nil {
			return err
		}
	}
	return nil
}

// Close stops all calculations and closes the service. No other methods should be
// called after Close.
func (s *Service) Close() error {
	// Disconnect from channel changes FIRST to prevent new change events This must be
	// done outside the lock to avoid deadlock with handleChange
	s.disconnectFromChannelChanges()
	retryErr := s.retries.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	err := retryErr
	for _, g := range s.mu.groups {
		err = errors.Join(err, g.Close())
	}
	return err
}

// Request requests that the Service starts calculation the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be increased. The caller must close the returned io.Closer when the
// calculation is no longer needed, which will decrement the number of active requests.
func (s *Service) updateRequests(
	ctx context.Context, added, removed []channel.Key,
) error {
	var channels []channel.Channel
	if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.cfg.Channel.NewRetrieve().
			Where(channel.MatchKeys(added...)).
			Entries(&channels).
			Exec(ctx, snap)
	}); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	running := s.mu.graph.CalculatedKeys()
	for _, k := range removed {
		if s.mu.pending[k] > 0 {
			if s.mu.pending[k]--; s.mu.pending[k] == 0 {
				delete(s.mu.pending, k)
			}
			continue
		}
		if _, err := s.mu.graph.Remove(k); err != nil {
			return err
		}
	}
	for _, ch := range channels {
		if ch.IsCalculated() {
			s.request(ctx, ch)
		}
	}
	if err := s.start(ctx, running); err != nil {
		return err
	}
	if len(added) > 0 {
		s.cfg.L.Debug("calculation requests added", zap.Stringers("channels", added))
	}
	if len(removed) > 0 {
		s.cfg.L.Debug(
			"calculation requests removed", zap.Stringers("channels", removed),
		)
	}
	return nil
}

func (s *Service) OpenRequestManager() *RequestManager {
	return &RequestManager{svc: s}
}

type RequestManager struct {
	svc      *Service
	currKeys channel.Keys
}

func (r *RequestManager) Set(ctx context.Context, keys channel.Keys) error {
	keys = keys.Unique()
	added, removed := lo.Difference(keys, r.currKeys)
	if err := r.svc.updateRequests(ctx, added, removed); err != nil {
		return err
	}
	r.currKeys = keys
	return nil
}

func (r *RequestManager) Close(ctx context.Context) error {
	return r.svc.updateRequests(ctx, nil, r.currKeys)
}
