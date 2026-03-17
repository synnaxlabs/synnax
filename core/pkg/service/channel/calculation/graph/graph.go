// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/synnaxlabs/alamos"
	channel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	channelanalyzer "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	calcompiler "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type StatusDetails struct {
	Channel channel.Key `json:"channel" msgpack:"channel"`
}

type node struct {
	channel.Channel
	deps       channel.Keys
	unresolved []string
	invalid    bool
}

type Graph struct {
	alamos.Instrumentation
	distribution *channel.Service
	status       status.Writer[StatusDetails]
	disconnect   observe.Disconnect
	mu           struct {
		nodes            map[channel.Key]node
		dependents       map[channel.Key]map[channel.Key]struct{}
		unresolvedByName map[string]map[channel.Key]struct{}
		sync.RWMutex
	}
}

type Config struct {
	Channel *channel.Service
	Status  *status.Service
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

func (c Config) Validate() error {
	v := validate.New("service.channel.calculation.graph")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

func Open(
	ctx context.Context,
	cfgs ...Config,
) (*Graph, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Graph{
		Instrumentation: cfg.Instrumentation,
		distribution:    cfg.Channel,
		status:          status.NewWriter[StatusDetails](cfg.Status, nil),
	}
	s.mu.nodes = make(map[channel.Key]node)
	s.mu.dependents = make(map[channel.Key]map[channel.Key]struct{})
	s.mu.unresolvedByName = make(map[string]map[channel.Key]struct{})
	if err = s.hydrate(ctx); err != nil {
		return nil, err
	}
	s.disconnect = cfg.Channel.NewObservable().OnChange(s.handleChanges)
	return s, nil
}

func (s *Graph) Close() error {
	if s.disconnect != nil {
		s.disconnect()
	}
	return nil
}

func (s *Graph) hydrate(ctx context.Context) error {
	var channels []channel.Channel
	if err := s.distribution.NewRetrieve().WhereCalculated().Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	s.L.Info("hydrating calculated channel graph", zap.Int("count", len(channels)))
	repairs := make([]channel.Channel, 0)
	pass := 0
	invalidCount := 0
	for {
		changed := false
		analyzer := s.newAnalyzer(nil)
		nextNodes := make(map[channel.Key]node)
		nextDependents := make(map[channel.Key]map[channel.Key]struct{})
		nextUnresolved := make(map[string]map[channel.Key]struct{})
		invalidCount = 0
		for _, ch := range channels {
			nd, err := s.inspectNode(ctx, nil, ch, analyzer)
			if err != nil {
				s.setNodeStatus(ctx, ch.Key(), ch.Name, err)
				invalidCount++
				s.L.Debug("channel expression invalid",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
			} else {
				s.clearNodeStatus(ctx, ch.Key())
			}
			upsertNode(nextNodes, nextDependents, nextUnresolved, nd)
			if !nd.invalid && ch.DataType != nd.DataType {
				s.L.Info("repairing channel DataType",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.String("old", string(ch.DataType)),
					zap.String("new", string(nd.DataType)),
				)
				ch.DataType = nd.DataType
				repairs = append(repairs, ch)
				changed = true
			}
		}
		pass++
		if !changed {
			s.mu.Lock()
			s.mu.nodes = nextNodes
			s.mu.dependents = nextDependents
			s.mu.unresolvedByName = nextUnresolved
			s.mu.Unlock()
			break
		}
		s.L.Debug("hydration fixpoint pass required another iteration",
			zap.Int("pass", pass),
			zap.Int("repairs", len(repairs)),
		)
	}
	if len(repairs) > 0 {
		s.L.Info("persisting DataType repairs from hydration", zap.Int("count", len(repairs)))
		if err := s.distribution.NewWriter(nil).CreateMany(ctx, &repairs); err != nil {
			return err
		}
	}
	s.L.Info("hydration complete",
		zap.Int("channels", len(channels)),
		zap.Int("invalid", invalidCount),
		zap.Int("repairs", len(repairs)),
		zap.Int("passes", pass),
	)
	return nil
}

func (s *Graph) handleChanges(ctx context.Context, reader gorp.TxReader[channel.Key, channel.Channel]) {
	s.mu.Lock()
	analyzer := s.newAnalyzer(nil)
	queued := make(map[channel.Key]struct{})
	var unresolvedNames []string
	var updates []channel.Channel
	for chg := range reader {
		ch := chg.Value
		if chg.Variant == change.VariantDelete {
			s.L.Debug("channel deleted, removing node and re-inspecting dependents",
				zap.Stringer("channel", chg.Key),
			)
			s.removeNode(chg.Key)
			if ch.Name != "" {
				unresolvedNames = append(unresolvedNames, ch.Name)
			}
			s.enqueueDependents(chg.Key, queued)
			continue
		}
		if ch.IsCalculated() {
			node, err := s.inspectNode(ctx, nil, ch, analyzer)
			if err != nil {
				s.L.Info("calculated channel has invalid expression",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
				s.setNodeStatus(ctx, ch.Key(), ch.Name, err)
			} else {
				s.L.Debug("calculated channel inspected",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Stringers("deps", node.deps),
				)
				s.clearNodeStatus(ctx, ch.Key())
			}
			if !node.invalid && node.DataType != ch.DataType {
				s.L.Debug("calculated channel DataType changed",
					zap.Stringer("channel", ch.Key()),
					zap.String("old", string(ch.DataType)),
					zap.String("new", string(node.DataType)),
				)
				updates = append(updates, node.Channel)
			}
			s.upsertNode(node)
			s.enqueueDependents(ch.Key(), queued)
			continue
		}
		s.enqueueDependents(ch.Key(), queued)
		unresolvedNames = append(unresolvedNames, ch.Name)
	}
	updates = append(updates, s.reconcileQueued(ctx, nil, queued, unresolvedNames, nil, analyzer)...)
	s.mu.Unlock()
	if len(updates) > 0 {
		s.L.Info("persisting DataType updates", zap.Int("count", len(updates)))
		if err := s.distribution.NewWriter(nil).CreateMany(ctx, &updates); err != nil {
			s.L.Error("failed to persist DataType updates", zap.Error(err))
		}
	}
}

func (s *Graph) setNodeStatus(ctx context.Context, key channel.Key, name string, err error) {
	if sErr := s.status.Set(ctx, &status.Status[StatusDetails]{
		Key:         channel.OntologyID(key).String(),
		Name:        name,
		Variant:     xstatus.VariantError,
		Message:     fmt.Sprintf("invalid expression for %s", name),
		Description: err.Error(),
		Time:        telem.Now(),
		Details:     StatusDetails{Channel: key},
	}); sErr != nil {
		s.L.Warn("failed to set error status for channel",
			zap.Stringer("channel", key),
			zap.Error(sErr),
		)
	}
}

func (s *Graph) clearNodeStatus(ctx context.Context, key channel.Key) {
	if err := s.status.Delete(ctx, channel.OntologyID(key).String()); err != nil {
		s.L.Warn("failed to clear status for channel",
			zap.Stringer("channel", key),
			zap.Error(err),
		)
	}
}

func (s *Graph) newAnalyzer(tx gorp.Tx) *channelanalyzer.Analyzer {
	return channelanalyzer.New(symbol.NewResolver(s.distribution, tx))
}

func (s *Graph) inspectNode(
	ctx context.Context,
	tx gorp.Tx,
	ch channel.Channel,
	analyzer *channelanalyzer.Analyzer,
) (node, error) {
	if analyzer == nil {
		analyzer = s.newAnalyzer(tx)
	}
	dt, err := analyzer.Analyze(ctx, ch)
	nd := node{Channel: ch}
	if ch.Key() == 0 {
		nd.LocalKey = 1
	}
	if err == nil {
		nd.DataType = dt
		prog, preErr := calcompiler.PreProcess(ctx, calcompiler.Config{
			ChannelService: s.distribution,
			Channel:        nd.Channel,
			SymbolResolver: symbol.NewResolver(s.distribution, tx),
		})
		if preErr != nil {
			err = preErr
		} else if len(prog.Functions) > 0 {
			nd.deps = make(channel.Keys, 0, len(prog.Functions[0].Channels.Read))
			for key := range prog.Functions[0].Channels.Read {
				nd.deps = append(nd.deps, channel.Key(key))
			}
		}
	}
	nd.invalid = err != nil
	return nd, err
}

func (s *Graph) reconcileQueued(
	ctx context.Context,
	tx gorp.Tx,
	queued map[channel.Key]struct{},
	unresolvedNames []string,
	overlayMap map[channel.Key]channel.Channel,
	analyzer *channelanalyzer.Analyzer,
) []channel.Channel {
	if overlayMap == nil {
		overlayMap = make(map[channel.Key]channel.Channel)
	}
	s.enqueueUnresolved(unresolvedNames, queued)
	if len(queued) > 0 {
		s.L.Debug("reconciling dependent channels", zap.Int("count", len(queued)))
	}
	updates := make([]channel.Channel, 0)
	for len(queued) > 0 {
		next := make(map[channel.Key]struct{})
		for key := range queued {
			nd, ok := s.mu.nodes[key]
			if !ok {
				continue
			}
			refetched := nd.Channel
			if err := s.distribution.NewRetrieve().WhereKeys(key).Entry(&refetched).Exec(ctx, tx); err != nil {
				s.L.Warn("failed to refetch channel during reconciliation",
					zap.Stringer("channel", key),
					zap.Error(err),
				)
				continue
			}
			newNode, err := s.inspectNode(ctx, tx, refetched, analyzer)
			oldInvalid := nd.invalid
			oldType := nd.DataType
			s.upsertNode(newNode)
			if err != nil {
				s.L.Info("dependent channel became invalid after reconciliation",
					zap.Stringer("channel", key),
					zap.String("name", refetched.Name),
					zap.Error(err),
				)
				s.setNodeStatus(ctx, key, refetched.Name, err)
				continue
			}
			s.clearNodeStatus(ctx, key)
			if oldInvalid || oldType != newNode.DataType {
				if oldType != newNode.DataType {
					s.L.Debug("dependent channel DataType changed during reconciliation",
						zap.Stringer("channel", key),
						zap.String("name", refetched.Name),
						zap.String("old", string(oldType)),
						zap.String("new", string(newNode.DataType)),
					)
					updates = append(updates, newNode.Channel)
					overlayMap[key] = newNode.Channel
				}
				s.enqueueDependents(key, next)
			}
		}
		queued = next
	}
	return updates
}

func (s *Graph) removeNode(key channel.Key) {
	nd, ok := s.mu.nodes[key]
	if !ok {
		return
	}
	for _, dep := range nd.deps {
		delete(s.mu.dependents[dep], key)
		if len(s.mu.dependents[dep]) == 0 {
			delete(s.mu.dependents, dep)
		}
	}
	for _, name := range nd.unresolved {
		delete(s.mu.unresolvedByName[name], key)
		if len(s.mu.unresolvedByName[name]) == 0 {
			delete(s.mu.unresolvedByName, name)
		}
	}
	delete(s.mu.nodes, key)
}

func (s *Graph) upsertNode(node node) {
	s.removeNode(node.Key())
	upsertNode(s.mu.nodes, s.mu.dependents, s.mu.unresolvedByName, node)
}

func upsertNode(
	nodes map[channel.Key]node,
	dependents map[channel.Key]map[channel.Key]struct{},
	unresolvedByName map[string]map[channel.Key]struct{},
	nd node,
) {
	nodes[nd.Key()] = nd
	for _, dep := range nd.deps {
		if dependents[dep] == nil {
			dependents[dep] = make(map[channel.Key]struct{})
		}
		dependents[dep][nd.Key()] = struct{}{}
	}
	for _, name := range nd.unresolved {
		if unresolvedByName[name] == nil {
			unresolvedByName[name] = make(map[channel.Key]struct{})
		}
		unresolvedByName[name][nd.Key()] = struct{}{}
	}
}

func (s *Graph) enqueueDependents(key channel.Key, queued map[channel.Key]struct{}) {
	for dep := range s.mu.dependents[key] {
		queued[dep] = struct{}{}
	}
}

func (s *Graph) enqueueUnresolved(names []string, queued map[channel.Key]struct{}) {
	for _, name := range names {
		for key := range s.mu.unresolvedByName[name] {
			queued[key] = struct{}{}
		}
	}
}
