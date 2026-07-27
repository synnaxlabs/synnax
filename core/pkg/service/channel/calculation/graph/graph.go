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
	"go/types"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Channels is the surface of the channel service the graph drives. It is implemented
// by the service-layer channel service, which owns the graph's lifecycle; defining it
// here keeps the graph free of a dependency on that package.
type Channels interface {
	// RetrieveCalculated returns all calculated channels. tx scopes the read; nil
	// reads the backing DB directly.
	RetrieveCalculated(ctx context.Context, tx gorp.Tx) ([]versions.Channel, error)
	// Retrieve returns the channel with the given key. tx scopes the read; nil reads
	// the backing DB directly.
	Retrieve(ctx context.Context, tx gorp.Tx, key versions.Key) (versions.Channel, error)
	// ChangeDataType persists dataType to the channel with the given key without
	// re-analyzing its expression. tx scopes the write; nil writes to the backing DB
	// directly.
	ChangeDataType(
		ctx context.Context, tx gorp.Tx, key versions.Key, dataType telem.DataType,
	) error
	// NewAnalyzer returns an analyzer for calculated channel expressions whose symbol
	// resolution is scoped to tx; nil resolves against the backing DB directly.
	NewAnalyzer(tx gorp.Tx) *calculation.Analyzer
	// Observe returns an observable that notifies the graph of changes to channels.
	Observe() observe.Observable[gorp.TxReader[versions.Key, versions.Channel]]
}

type node struct {
	versions.Channel
	deps       []versions.Key
	unresolved []string
	invalid    bool
}

// Graph tracks all calculated channels, their dependency edges, and their inferred
// DataTypes. It subscribes to the channel observable and reactively re-inspects
// affected nodes when channels are created, updated, or deleted.
type Graph struct {
	alamos.Instrumentation
	db         *gorp.DB
	channels   Channels
	status     status.Writer[types.Nil]
	disconnect observe.Disconnect
	mu         struct {
		nodes            map[versions.Key]node
		dependents       map[versions.Key]set.Set[versions.Key]
		unresolvedByName map[string]set.Set[versions.Key]
		sync.RWMutex
	}
}

// Config configures a Graph.
type Config struct {
	// DB is the metadata database backing the channel table. It must be the same DB
	// the Channels implementation reads and writes, so the graph can hydrate —
	// retrieve, analyze, and persist DataType repairs — within a single transaction
	// consistent with the channels it reads.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channels is the channel service surface the graph retrieves channels from,
	// persists DataType repairs to, and observes for changes.
	//
	// [REQUIRED]
	Channels Channels
	// Status is used to publish error/clear statuses for calculated channels.
	//
	// [REQUIRED]
	Status *status.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[Config] = Config{}

func (c Config) Validate() error {
	v := validate.New("service.channel.calculation.graph")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "channels", c.Channels)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channels = override.Nil(c.Channels, other.Channels)
	c.Status = override.Nil(c.Status, other.Status)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Open creates a Graph, hydrates it from all existing calculated channels, and
// subscribes to the channel observable for reactive updates.
func Open(
	ctx context.Context,
	cfgs ...Config,
) (*Graph, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	g := &Graph{
		Instrumentation: cfg.Instrumentation,
		db:              cfg.DB,
		channels:        cfg.Channels,
		status:          status.NewWriter[types.Nil](cfg.Status, nil),
	}
	g.mu.nodes = make(map[versions.Key]node)
	g.mu.dependents = make(map[versions.Key]set.Set[versions.Key])
	g.mu.unresolvedByName = make(map[string]set.Set[versions.Key])
	if err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.hydrate(ctx, tx)
	}); err != nil {
		return nil, err
	}
	g.disconnect = cfg.Channels.Observe().OnChange(g.handleChanges)
	return g, nil
}

// Close disconnects the graph from the channel observable.
func (g *Graph) Close() error {
	if g.disconnect != nil {
		g.disconnect()
	}
	return nil
}

func (g *Graph) hydrate(ctx context.Context, tx gorp.Tx) error {
	channels, err := g.channels.RetrieveCalculated(ctx, tx)
	if err != nil {
		return err
	}
	g.L.Info("hydrating calculated channel graph", zap.Int("count", len(channels)))
	repairs := make([]versions.Channel, 0)
	var (
		pass           int
		invalidCount   int
		nextNodes      map[versions.Key]node
		nextDependents map[versions.Key]set.Set[versions.Key]
		nextUnresolved map[string]set.Set[versions.Key]
	)
	analyzer := g.channels.NewAnalyzer(tx)
	statuses := make(map[versions.Key]*calculation.Status)
	for {
		changed := false
		nextNodes = make(map[versions.Key]node)
		nextDependents = make(map[versions.Key]set.Set[versions.Key])
		nextUnresolved = make(map[string]set.Set[versions.Key])
		invalidCount = 0
		for i, ch := range channels {
			nd, err := g.inspectNode(ctx, tx, ch, analyzer)
			if err != nil {
				statuses[ch.Key()] = calculation.StatusFromError(ch.Key(), ch.Name, fmt.Sprintf("invalid expression for %s", ch.Name), err)
				invalidCount++
				g.L.Debug("channel expression invalid",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
			} else {
				statuses[ch.Key()] = nil
			}
			upsertNode(nextNodes, nextDependents, nextUnresolved, nd)
			if !nd.invalid && ch.DataType != nd.DataType {
				g.L.Info("repairing channel DataType",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.String("old", string(ch.DataType)),
					zap.String("new", string(nd.DataType)),
				)
				channels[i].DataType = nd.DataType
				repairs = append(repairs, channels[i])
				changed = true
			}
		}
		pass++
		if !changed {
			break
		}
		if pass > len(channels)+1 {
			g.L.Warn("hydration fixpoint did not converge, breaking",
				zap.Int("pass", pass),
				zap.Int("channels", len(channels)),
			)
			break
		}
		g.L.Debug("hydration fixpoint pass required another iteration",
			zap.Int("pass", pass),
			zap.Int("repairs", len(repairs)),
		)
	}
	for key, st := range statuses {
		if st != nil {
			g.setNodeStatus(ctx, st)
		} else {
			g.clearNodeStatus(ctx, key)
		}
	}
	g.mu.Lock()
	g.mu.nodes = nextNodes
	g.mu.dependents = nextDependents
	g.mu.unresolvedByName = nextUnresolved
	g.mu.Unlock()
	if len(repairs) > 0 {
		g.L.Info("persisting DataType repairs from hydration", zap.Int("count", len(repairs)))
		for _, ch := range repairs {
			if err := g.channels.ChangeDataType(ctx, tx, ch.Key(), ch.DataType); err != nil {
				return err
			}
		}
	}
	g.L.Info("hydration complete",
		zap.Int("channels", len(channels)),
		zap.Int("invalid", invalidCount),
		zap.Int("repairs", len(repairs)),
		zap.Int("passes", pass),
	)
	return nil
}

func (g *Graph) handleChanges(ctx context.Context, reader gorp.TxReader[versions.Key, versions.Channel]) {
	g.mu.Lock()
	analyzer := g.channels.NewAnalyzer(nil)
	queued := make(set.Set[versions.Key])
	var unresolvedNames []string
	var updates []versions.Channel
	for chg := range reader {
		ch := chg.Value
		if chg.Variant == change.VariantDelete {
			g.L.Debug("channel deleted, removing node and re-inspecting dependents",
				zap.Stringer("channel", chg.Key),
			)
			g.removeNode(chg.Key)
			if ch.Name != "" {
				unresolvedNames = append(unresolvedNames, ch.Name)
			}
			g.enqueueDependents(chg.Key, queued)
			continue
		}
		if ch.IsCalculated() {
			nd, err := g.inspectNode(ctx, nil, ch, analyzer)
			if err != nil {
				g.L.Info("calculated channel has invalid expression",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
				g.setNodeStatus(ctx, calculation.StatusFromError(ch.Key(), ch.Name, fmt.Sprintf("invalid expression for %s", ch.Name), err))
			} else {
				g.L.Debug("calculated channel inspected",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Stringers("deps", nd.deps),
				)
				g.clearNodeStatus(ctx, ch.Key())
			}
			if !nd.invalid && nd.DataType != ch.DataType {
				g.L.Debug("calculated channel DataType changed",
					zap.Stringer("channel", ch.Key()),
					zap.String("old", string(ch.DataType)),
					zap.String("new", string(nd.DataType)),
				)
				updates = append(updates, nd.Channel)
			}
			g.upsertNode(nd)
			g.enqueueDependents(ch.Key(), queued)
			continue
		}
		g.enqueueDependents(ch.Key(), queued)
		unresolvedNames = append(unresolvedNames, ch.Name)
	}
	updates = append(updates, g.reconcileQueued(ctx, nil, queued, unresolvedNames, analyzer)...)
	g.mu.Unlock()
	if len(updates) > 0 {
		g.L.Info("updating channel data types", zap.Int("count", len(updates)))
		for _, ch := range updates {
			if err := g.channels.ChangeDataType(ctx, nil, ch.Key(), ch.DataType); err != nil {
				g.L.Error(
					"failed to update channel data type",
					zap.Stringer("channel", ch.Key()),
					zap.Error(err),
				)
			}
		}
	}
}

func (g *Graph) setNodeStatus(ctx context.Context, st *calculation.Status) {
	if sErr := g.status.Set(ctx, st); sErr != nil {
		g.L.Warn("failed to set error status for channel",
			zap.String("key", st.Key),
			zap.Error(sErr),
		)
	}
}

func (g *Graph) clearNodeStatus(ctx context.Context, key versions.Key) {
	if err := g.status.Delete(ctx, calculation.StatusKey(key)); err != nil {
		g.L.Warn("failed to clear status for channel",
			zap.Stringer("channel", key),
			zap.Error(err),
		)
	}
}

func (g *Graph) inspectNode(
	ctx context.Context,
	tx gorp.Tx,
	ch versions.Channel,
	analyzer *calculation.Analyzer,
) (node, error) {
	if analyzer == nil {
		analyzer = g.channels.NewAnalyzer(tx)
	}
	if ch.Key() == 0 {
		return node{}, errors.Newf("channel %q has no key, cannot inspect", ch.Name)
	}
	result, err := analyzer.Analyze(ctx, ch)
	nd := node{Channel: ch}
	if err == nil {
		nd.DataType = result.ChanDataType
		nd.deps = result.Deps
	} else {
		nd.unresolved = result.Unresolved
	}
	nd.invalid = err != nil
	return nd, err
}

func (g *Graph) reconcileQueued(
	ctx context.Context,
	tx gorp.Tx,
	queued set.Set[versions.Key],
	unresolvedNames []string,
	analyzer *calculation.Analyzer,
) []versions.Channel {
	g.enqueueUnresolved(unresolvedNames, queued)
	if len(queued) > 0 {
		g.L.Debug("reconciling dependent channels", zap.Int("count", len(queued)))
	}
	updates := make([]versions.Channel, 0)
	for len(queued) > 0 {
		next := make(set.Set[versions.Key])
		for key := range queued {
			nd, ok := g.mu.nodes[key]
			if !ok {
				continue
			}
			refetched, err := g.channels.Retrieve(ctx, tx, key)
			if err != nil {
				g.L.Warn("failed to refetch channel during reconciliation",
					zap.Stringer("channel", key),
					zap.Error(err),
				)
				continue
			}
			newNode, err := g.inspectNode(ctx, tx, refetched, analyzer)
			oldInvalid := nd.invalid
			oldType := nd.DataType
			g.upsertNode(newNode)
			if err != nil {
				g.L.Info("dependent channel became invalid after reconciliation",
					zap.Stringer("channel", key),
					zap.String("name", refetched.Name),
					zap.Error(err),
				)
				g.setNodeStatus(ctx, calculation.StatusFromError(key, refetched.Name, fmt.Sprintf("invalid expression for %s", refetched.Name), err))
				continue
			}
			g.clearNodeStatus(ctx, key)
			if oldInvalid || oldType != newNode.DataType {
				if oldType != newNode.DataType {
					g.L.Debug("dependent channel DataType changed during reconciliation",
						zap.Stringer("channel", key),
						zap.String("name", refetched.Name),
						zap.String("old", string(oldType)),
						zap.String("new", string(newNode.DataType)),
					)
					updates = append(updates, newNode.Channel)
				}
				g.enqueueDependents(key, next)
			}
		}
		queued = next
	}
	return updates
}

func (g *Graph) removeNode(key versions.Key) {
	nd, ok := g.mu.nodes[key]
	if !ok {
		return
	}
	for _, dep := range nd.deps {
		g.mu.dependents[dep].Remove(key)
		if len(g.mu.dependents[dep]) == 0 {
			delete(g.mu.dependents, dep)
		}
	}
	for _, name := range nd.unresolved {
		g.mu.unresolvedByName[name].Remove(key)
		if len(g.mu.unresolvedByName[name]) == 0 {
			delete(g.mu.unresolvedByName, name)
		}
	}
	delete(g.mu.nodes, key)
}

func (g *Graph) upsertNode(nd node) {
	g.removeNode(nd.Key())
	upsertNode(g.mu.nodes, g.mu.dependents, g.mu.unresolvedByName, nd)
}

func upsertNode(
	nodes map[versions.Key]node,
	dependents map[versions.Key]set.Set[versions.Key],
	unresolvedByName map[string]set.Set[versions.Key],
	nd node,
) {
	nodes[nd.Key()] = nd
	for _, dep := range nd.deps {
		if dependents[dep] == nil {
			dependents[dep] = make(set.Set[versions.Key])
		}
		dependents[dep].Add(nd.Key())
	}
	for _, name := range nd.unresolved {
		if unresolvedByName[name] == nil {
			unresolvedByName[name] = make(set.Set[versions.Key])
		}
		unresolvedByName[name].Add(nd.Key())
	}
}

func (g *Graph) enqueueDependents(key versions.Key, queued set.Set[versions.Key]) {
	for dep := range g.mu.dependents[key] {
		queued.Add(dep)
	}
}

func (g *Graph) enqueueUnresolved(names []string, queued set.Set[versions.Key]) {
	for _, name := range names {
		for key := range g.mu.unresolvedByName[name] {
			queued.Add(key)
		}
	}
}
