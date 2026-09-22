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
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Changes is a batch of channel changes the Graph reconciles as a unit.
type Changes = gorp.TxReader[channel.Key, channel.Channel]

type node struct {
	channel.Channel
	deps       channel.Keys
	unresolved []string
	invalid    bool
}

// Graph tracks all calculated channels, their dependency edges, and their inferred
// DataTypes. It subscribes to the channel observable and reactively re-inspects
// affected nodes when channels are created, updated, or deleted.
//
// The Graph is the sole writer of calculated channel statuses. The calculation runtime
// reports through SetRuntimeStatus instead of writing the record itself.
type Graph struct {
	alamos.Instrumentation
	db         *gorp.DB
	svc        *channel.Service
	status     *status.Service
	obs        observe.Observer[Changes]
	disconnect observe.Disconnect
	mu         struct {
		nodes            map[channel.Key]node
		dependents       map[channel.Key]set.Set[channel.Key]
		unresolvedByName map[string]set.Set[channel.Key]
		sync.RWMutex
	}
}

// Config configures a Graph.
type Config struct {
	// DB is the metadata database backing the channel table. It must be the same DB the
	// Channel service is configured with, so the graph can hydrate — retrieve, analyze,
	// and persist DataType repairs — within a single transaction consistent with the
	// channels it reads.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel is the service-layer channel service. The graph retrieves and writes
	// channels through its embedded distribution service and builds the Arc symbol
	// resolver (via NewArcSymbolResolver) used to analyze calculated channel
	// expressions.
	//
	// [REQUIRED]
	Channel *channel.Service
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
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("status", c.Status)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Open creates a Graph, hydrates it from all existing calculated channels, and
// subscribes to the channel observable for reactive updates.
func Open(ctx context.Context, cfgs ...Config) (*Graph, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	g := &Graph{
		Instrumentation: cfg.Instrumentation,
		db:              cfg.DB,
		svc:             cfg.Channel,
		status:          cfg.Status,
		obs:             observe.New[Changes](),
	}
	g.mu.nodes = make(map[channel.Key]node)
	g.mu.dependents = make(map[channel.Key]set.Set[channel.Key])
	g.mu.unresolvedByName = make(map[string]set.Set[channel.Key])
	if err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.hydrate(ctx, tx)
	}); err != nil {
		return nil, err
	}
	g.disconnect = cfg.Channel.Observe().OnChange(g.handleChanges)
	return g, nil
}

// Observe returns an observable that fires with a channel change batch after the Graph
// has reconciled it and committed the resulting statuses and node state. Subscribers
// that act on calculated channels must use this instead of the channel observable, so
// that their work runs against a reconciled graph and their status reports land after
// the Graph's own.
func (g *Graph) Observe() observe.Observable[Changes] { return g.obs }

// SetRuntimeStatus persists a status reported by the calculation runtime for the
// channel with the given key. Routing runtime reports through the Graph keeps a single
// writer on the status record, so a report and a validity clear apply in submission
// order. A report for a channel deleted in the meantime is dropped.
func (g *Graph) SetRuntimeStatus(
	ctx context.Context,
	key channel.Key,
	stat *calculation.Status,
) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.db.WithTx(ctx, func(tx gorp.Tx) error {
		exists, err := g.svc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Exists(ctx, tx)
		if err != nil {
			return err
		}
		if !exists {
			g.L.Debug("dropping runtime status for a deleted channel",
				zap.Stringer("channel", key),
			)
			return nil
		}
		stat.Key = calculation.StatusKey(key)
		return g.status.NewWriter(tx).Set(ctx, stat)
	})
}

// ClearRuntimeStatus removes the status record for the channel with the given key
// after the calculation runtime recovers. A node the Graph holds as invalid keeps its
// status: that record describes the channel definition, not the runtime.
func (g *Graph) ClearRuntimeStatus(ctx context.Context, key channel.Key) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if nd, ok := g.mu.nodes[key]; ok && nd.invalid {
		return nil
	}
	return g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.status.NewWriter(tx).Delete(ctx, calculation.StatusKey(key))
	})
}

// Close disconnects the graph from the channel observable.
func (g *Graph) Close() error {
	if g.disconnect != nil {
		g.disconnect()
	}
	return nil
}

func (g *Graph) hydrate(ctx context.Context, tx gorp.Tx) error {
	var channels []channel.Channel
	if err := g.svc.NewRetrieve().Where(
		channel.MatchCalculated(),
	).Entries(&channels).Exec(ctx, tx); err != nil {
		return err
	}
	g.L.Info("hydrating calculated channel graph", zap.Int("count", len(channels)))
	repairs := make([]channel.Channel, 0)
	var (
		pass           int
		invalidCount   int
		nextNodes      map[channel.Key]node
		nextDependents map[channel.Key]set.Set[channel.Key]
		nextUnresolved map[string]set.Set[channel.Key]
	)
	analyzer := g.newAnalyzer(tx)
	statuses := make(map[channel.Key]*calculation.Status)
	for {
		changed := false
		nextNodes = make(map[channel.Key]node)
		nextDependents = make(map[channel.Key]set.Set[channel.Key])
		nextUnresolved = make(map[string]set.Set[channel.Key])
		invalidCount = 0
		for i, ch := range channels {
			nd, err := g.inspectNode(ctx, tx, ch, analyzer)
			if err != nil {
				statuses[ch.Key()] = calculation.StatusFromError(
					ch.Key(),
					ch.Name,
					fmt.Sprintf("invalid expression for %s", ch.Name),
					err,
				)
				invalidCount++
				g.L.Debug(
					"channel expression invalid",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
			} else {
				statuses[ch.Key()] = nil
			}
			upsertNode(nextNodes, nextDependents, nextUnresolved, nd)
			if !nd.invalid && ch.DataType != nd.DataType {
				g.L.Info(
					"repairing channel DataType",
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
			g.L.Warn(
				"hydration fixpoint did not converge, breaking",
				zap.Int("pass", pass),
				zap.Int("channels", len(channels)),
			)
			break
		}
		g.L.Debug(
			"hydration fixpoint pass required another iteration",
			zap.Int("pass", pass),
			zap.Int("repairs", len(repairs)),
		)
	}
	for key, st := range statuses {
		if st != nil {
			g.setNodeStatus(ctx, tx, st)
		} else {
			g.clearNodeStatus(ctx, tx, key)
		}
	}
	g.mu.Lock()
	g.mu.nodes = nextNodes
	g.mu.dependents = nextDependents
	g.mu.unresolvedByName = nextUnresolved
	g.mu.Unlock()
	if len(repairs) > 0 {
		g.L.Info(
			"persisting DataType repairs from hydration",
			zap.Int("count", len(repairs)),
		)
		w := g.svc.NewWriter(tx)
		for _, ch := range repairs {
			if err := w.ChangeDataType(ctx, ch.Key(), ch.DataType); err != nil {
				return err
			}
		}
	}
	g.L.Info(
		"hydration complete",
		zap.Int("channels", len(channels)),
		zap.Int("invalid", invalidCount),
		zap.Int("repairs", len(repairs)),
		zap.Int("passes", pass),
	)
	return nil
}

func (g *Graph) handleChanges(ctx context.Context, reader Changes) {
	var (
		updates []channel.Channel
		batch   []change.Change[channel.Key, channel.Channel]
	)
	g.mu.Lock()
	err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		analyzer := g.newAnalyzer(tx)
		queued := make(set.Set[channel.Key])
		var unresolvedNames []string
		for chg := range reader {
			batch = append(batch, chg)
			ch := chg.Value
			if chg.Variant == change.VariantDelete {
				g.L.Debug("channel deleted, removing node and re-inspecting dependents",
					zap.Stringer("channel", chg.Key),
				)
				if _, tracked := g.mu.nodes[chg.Key]; tracked {
					g.clearNodeStatus(ctx, tx, chg.Key)
				}
				g.removeNode(chg.Key)
				if ch.Name != "" {
					unresolvedNames = append(unresolvedNames, ch.Name)
				}
				g.enqueueDependents(chg.Key, queued)
				continue
			}
			if ch.IsCalculated() {
				nd, err := g.inspectNode(ctx, tx, ch, analyzer)
				if err != nil {
					g.L.Info("calculated channel has invalid expression",
						zap.Stringer("channel", ch.Key()),
						zap.String("name", ch.Name),
						zap.Error(err),
					)
					g.setNodeStatus(
						ctx,
						tx,
						calculation.StatusFromError(
							ch.Key(),
							ch.Name,
							fmt.Sprintf("invalid expression for %s", ch.Name),
							err,
						),
					)
				} else {
					g.L.Debug("calculated channel inspected",
						zap.Stringer("channel", ch.Key()),
						zap.String("name", ch.Name),
						zap.Stringers("deps", nd.deps),
					)
					prev, ok := g.mu.nodes[ch.Key()]
					if ok && (prev.invalid || prev.Expression != ch.Expression) {
						g.clearNodeStatus(ctx, tx, ch.Key())
					}
				}
				if !nd.invalid && nd.DataType != ch.DataType {
					g.L.Debug(
						"calculated channel DataType changed",
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
		updates = append(
			updates,
			g.reconcileQueued(ctx, tx, queued, unresolvedNames, analyzer)...,
		)
		return nil
	})
	g.mu.Unlock()
	if err != nil {
		g.L.Error("failed to apply calculated channel changes", zap.Error(err))
		return
	}
	g.obs.Notify(ctx, slices.Values(batch))
	if len(updates) > 0 {
		g.L.Info("updating channel data types", zap.Int("count", len(updates)))
		w := g.svc.NewWriter(nil)
		for _, ch := range updates {
			if err := w.ChangeDataType(ctx, ch.Key(), ch.DataType); err != nil {
				g.L.Error(
					"failed to update channel data type",
					zap.Stringer("channel", ch.Key()),
					zap.Error(err),
				)
			}
		}
	}
}

func (g *Graph) setNodeStatus(
	ctx context.Context,
	tx gorp.Tx,
	stat *calculation.Status,
) {
	if err := g.status.NewWriter(tx).Set(ctx, stat); err != nil {
		g.L.Warn(
			"failed to set error status for channel",
			zap.String("key", stat.Key),
			zap.Error(err),
		)
	}
}

func (g *Graph) clearNodeStatus(ctx context.Context, tx gorp.Tx, key channel.Key) {
	if err := g.status.NewWriter(tx).
		Delete(ctx, calculation.StatusKey(key)); err != nil {
		g.L.Warn(
			"failed to clear status for channel",
			zap.Stringer("channel", key),
			zap.Error(err),
		)
	}
}

func (g *Graph) newAnalyzer(tx gorp.Tx) *channel.CalculationAnalyzer {
	return channel.NewCalculationAnalyzer(g.svc.NewArcSymbolResolver(tx), parser.Config{
		AllowDashedNames: !g.svc.ShouldValidateNames(),
	})
}

func (g *Graph) inspectNode(
	ctx context.Context,
	tx gorp.Tx,
	ch channel.Channel,
	analyzer *channel.CalculationAnalyzer,
) (node, error) {
	if analyzer == nil {
		analyzer = g.newAnalyzer(tx)
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
	queued set.Set[channel.Key],
	unresolvedNames []string,
	analyzer *channel.CalculationAnalyzer,
) []channel.Channel {
	g.enqueueUnresolved(unresolvedNames, queued)
	if len(queued) > 0 {
		g.L.Debug("reconciling dependent channels", zap.Int("count", len(queued)))
	}
	updates := make([]channel.Channel, 0)
	for len(queued) > 0 {
		next := make(set.Set[channel.Key])
		for key := range queued {
			nd, ok := g.mu.nodes[key]
			if !ok {
				continue
			}
			refetched := nd.Channel
			if err := g.svc.NewRetrieve().
				Where(channel.MatchKeys(key)).
				Entry(&refetched).
				Exec(ctx, tx); err != nil {
				g.L.Warn(
					"failed to refetch channel during reconciliation",
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
				g.L.Info(
					"dependent channel became invalid after reconciliation",
					zap.Stringer("channel", key),
					zap.String("name", refetched.Name),
					zap.Error(err),
				)
				g.setNodeStatus(
					ctx,
					tx,
					calculation.StatusFromError(
						key,
						refetched.Name,
						fmt.Sprintf("invalid expression for %s", refetched.Name),
						err,
					),
				)
				continue
			}
			if oldInvalid {
				g.clearNodeStatus(ctx, tx, key)
			}
			if oldInvalid || oldType != newNode.DataType {
				if oldType != newNode.DataType {
					g.L.Debug(
						"dependent channel DataType changed during reconciliation",
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

func (g *Graph) removeNode(key channel.Key) {
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

func (g *Graph) upsertNode(node node) {
	g.removeNode(node.Key())
	upsertNode(g.mu.nodes, g.mu.dependents, g.mu.unresolvedByName, node)
}

func upsertNode(
	nodes map[channel.Key]node,
	dependents map[channel.Key]set.Set[channel.Key],
	unresolvedByName map[string]set.Set[channel.Key],
	nd node,
) {
	nodes[nd.Key()] = nd
	for _, dep := range nd.deps {
		if dependents[dep] == nil {
			dependents[dep] = make(set.Set[channel.Key])
		}
		dependents[dep].Add(nd.Key())
	}
	for _, name := range nd.unresolved {
		if unresolvedByName[name] == nil {
			unresolvedByName[name] = make(set.Set[channel.Key])
		}
		unresolvedByName[name].Add(nd.Key())
	}
}

func (g *Graph) enqueueDependents(key channel.Key, queued set.Set[channel.Key]) {
	for dep := range g.mu.dependents[key] {
		queued.Add(dep)
	}
}

func (g *Graph) enqueueUnresolved(names []string, queued set.Set[channel.Key]) {
	for _, name := range names {
		for key := range g.mu.unresolvedByName[name] {
			queued.Add(key)
		}
	}
}
