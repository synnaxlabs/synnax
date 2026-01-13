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
	"fmt"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening the calculation service.
type ServiceConfig struct {
	alamos.Instrumentation
	DB *gorp.DB
	// Framer is the underlying frame service to stream cache channel values and write
	// calculated samples.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve information about the channels being calculated.
	//
	// [REQUIRED]
	Channel *channel.Service
	// ChannelObservable is used to listen to real-time changes in calculated channels
	// so the calculation routines can be updated accordingly.
	// [REQUIRED]
	ChannelObservable observe.Observable[gorp.TxReader[channel.Key, channel.Channel]]
	// Arc is used for compiling arc programs used for executing calculations.
	// [REQUIRED]
	Arc *arc.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening the calculation
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "channel_observable", c.ChannelObservable)
	validate.NotNil(v, "arc", c.Arc)
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelObservable = override.Nil(c.ChannelObservable, other.ChannelObservable)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		graph       *graph.Graph
		calculators map[channel.Key]*calculator.Calculator
		groups      map[int]*group
	}
	disconnectFromChannelChanges observe.Disconnect
	stateKey                     channel.Key
	writer                       *framer.Writer
}

const statusChannelName = "sy_calculation_status"

// OpenService opens the service with the provided configuration. The service must be closed
// when it is no longer needed.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := graph.New(graph.Config{
		Instrumentation: cfg.Child("calculation.graph"),
		Channel:         cfg.Channel,
		SymbolResolver:  cfg.Arc.SymbolResolver(),
	})
	if err != nil {
		return nil, err
	}

	calculationStateCh := channel.Channel{
		Name:        statusChannelName,
		DataType:    telem.JSONT,
		Virtual:     true,
		Leaseholder: cluster.NodeKeyFree,
		Internal:    true,
	}

	if err = cfg.Channel.MapRename(ctx, map[string]string{
		"sy_calculation_state": statusChannelName,
	}, true); err != nil {
		return nil, err
	}

	if err = cfg.Channel.Create(
		ctx,
		&calculationStateCh,
		channel.RetrieveIfNameExists(),
	); err != nil {
		return nil, err
	}

	w, err := cfg.Framer.OpenWriter(ctx, framer.WriterConfig{
		Keys:        []channel.Key{calculationStateCh.Key()},
		Start:       telem.Now(),
		Authorities: []control.Authority{255},
	})
	if err != nil {
		return nil, err
	}

	s := &Service{cfg: cfg, writer: w, stateKey: calculationStateCh.Key()}
	s.disconnectFromChannelChanges = cfg.ChannelObservable.OnChange(s.handleChange)
	s.mu.graph = g
	s.mu.calculators = make(map[channel.Key]*calculator.Calculator)
	s.mu.groups = make(map[int]*group)

	s.cfg.L.Info("calculation service initialized",
		zap.String("status_channel", statusChannelName),
		zap.Uint32("status_channel_key", uint32(calculationStateCh.Key())),
	)

	return s, nil
}

func (s *Service) setStatus(
	_ context.Context,
	statuses ...calculator.Status,
) {
	if _, err := s.writer.Write(frame.NewUnary(
		s.stateKey,
		telem.NewSeriesStaticJSONV(statuses...),
	)); err != nil {
		s.cfg.L.Error("failed to encode state", zap.Error(err))
	}
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[channel.Key, channel.Channel],
) {
	for cg := range reader {
		ch := cg.Value
		// Don't stop calculating if the channel is deleted. The calculation will be
		// automatically shut down when it is no longer needed.
		if cg.Variant != change.Set || !ch.IsCalculated() {
			continue
		}
		s.mu.Lock()
		if _, found := s.mu.calculators[cg.Key]; !found {
			s.mu.Unlock()
			continue
		}
		if err := s.updateCalculation(ctx, ch); err != nil {
			s.setStatus(ctx, calculator.Status{
				Key:         ch.Key().String(),
				Variant:     status.VariantError,
				Message:     fmt.Sprintf("failed to update calculation for %s", ch),
				Description: err.Error(),
			})
		}
		s.mu.Unlock()
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

func (s *Service) openOrGetCalculator(
	ctx context.Context,
	mod compiler.Module,
) (*calculator.Calculator, error) {
	calc, err := calculator.Open(ctx, calculator.Config{Module: mod})
	if err != nil {
		return nil, err
	}
	s.mu.calculators[calc.Channel().Key()] = calc
	return calc, err
}

func groupEquals(
	mods []compiler.Module,
	g *group,
) bool {
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

func (s *Service) updateGroup(ctx context.Context, key int, mods []compiler.Module) error {
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
		calc, err := s.openOrGetCalculator(ctx, m)
		if err != nil {
			return err
		}
		calculators[i] = calc
	}
	g, err := openGroup(
		ctx,
		groupConfig{
			Instrumentation: s.cfg.Child("group"),
			Calculators:     calculators,
			OnStatusChange:  s.setStatus,
			Framer:          s.cfg.Framer,
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
	// Disconnect from channel changes FIRST to prevent new change events
	// This must be done outside the lock to avoid deadlock with handleChange
	s.disconnectFromChannelChanges()

	s.mu.Lock()
	defer s.mu.Unlock()
	c := errors.NewCatcher(errors.WithAggregation())
	for _, g := range s.mu.groups {
		c.Exec(g.Close)
	}
	return c.Error()
}

// Request requests that the Service starts calculation the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be increased. The caller must close the returned io.Closer when the
// calculation is no longer needed, which will decrement the number of active requests.
func (s *Service) updateRequests(ctx context.Context, added, removed []channel.Key) error {
	var (
		channels []channel.Channel
		statuses []calculator.Status
	)
	if err := s.cfg.Channel.NewRetrieve().
		WhereKeys(added...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, k := range removed {
		if err := s.mu.graph.Remove(k); err != nil {
			return err
		}
	}
	for _, ch := range channels {
		if !ch.IsCalculated() {
			continue
		}
		if err := s.mu.graph.Add(ctx, ch); err != nil {
			statuses = append(statuses, calculator.Status{
				Key:         ch.String(),
				Message:     fmt.Sprintf("Failed to request calculation for %s", ch),
				Description: err.Error(),
			})
		}
	}
	if len(statuses) > 0 {
		s.setStatus(ctx, statuses...)
	}
	if err := s.rebuildGroups(ctx); err != nil {
		return err
	}
	if len(added) > 0 {
		s.cfg.L.Debug("calculation requests added", zap.Stringers("channels", added))
	}
	if len(removed) > 0 {
		s.cfg.L.Debug("calculation requests removed", zap.Stringers("channels", removed))
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
	added, removed := lo.Difference(keys, r.currKeys)
	r.currKeys = keys
	return r.svc.updateRequests(ctx, added, removed)
}

func (r *RequestManager) Close(ctx context.Context) error {
	return r.svc.updateRequests(ctx, nil, r.currKeys)
}
