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
	"fmt"
	"go/types"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

type calcNode struct {
	Channel
	deps       Keys
	unresolved []string
	invalid    bool
}

// calculationGraph tracks all calculated channels, their dependency edges, and their
// inferred DataTypes. It subscribes to the channel observable and reactively
// re-inspects affected nodes when channels are created, updated, or deleted.
type calculationGraph struct {
	alamos.Instrumentation
	db         *gorp.DB
	svc        *Service
	status     status.Writer[types.Nil]
	disconnect observe.Disconnect
	mu         struct {
		nodes            map[Key]calcNode
		dependents       map[Key]set.Set[Key]
		unresolvedByName map[string]set.Set[Key]
		sync.RWMutex
	}
}

// openCalculationGraph creates a calculationGraph, hydrates it from all existing
// calculated channels, and subscribes to the channel observable for reactive updates.
func (s *Service) openCalculationGraph(ctx context.Context) (*calculationGraph, error) {
	g := &calculationGraph{
		Instrumentation: s.cfg.Child("calculation.graph"),
		db:              s.cfg.DB,
		svc:             s,
		status:          status.NewWriter[types.Nil](s.cfg.Status, nil),
	}
	g.mu.nodes = make(map[Key]calcNode)
	g.mu.dependents = make(map[Key]set.Set[Key])
	g.mu.unresolvedByName = make(map[string]set.Set[Key])
	if err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.hydrate(ctx, tx)
	}); err != nil {
		return nil, err
	}
	g.disconnect = s.Observe().OnChange(g.handleChanges)
	return g, nil
}

// Close disconnects the graph from the channel observable.
func (g *calculationGraph) Close() error {
	if g.disconnect != nil {
		g.disconnect()
	}
	return nil
}

func (g *calculationGraph) hydrate(ctx context.Context, tx gorp.Tx) error {
	var channels []Channel
	if err := g.svc.NewRetrieve().Where(
		MatchCalculated(),
	).Entries(&channels).Exec(ctx, tx); err != nil {
		return err
	}
	g.L.Info("hydrating calculated channel graph", zap.Int("count", len(channels)))
	repairs := make([]Channel, 0)
	var (
		pass           int
		invalidCount   int
		nextNodes      map[Key]calcNode
		nextDependents map[Key]set.Set[Key]
		nextUnresolved map[string]set.Set[Key]
	)
	analyzer := g.newAnalyzer(tx)
	statuses := make(map[Key]*CalculationStatus)
	for {
		changed := false
		nextNodes = make(map[Key]calcNode)
		nextDependents = make(map[Key]set.Set[Key])
		nextUnresolved = make(map[string]set.Set[Key])
		invalidCount = 0
		for i, ch := range channels {
			nd, err := g.inspectNode(ctx, tx, ch, analyzer)
			if err != nil {
				statuses[ch.Key()] = CalculationStatusFromError(ch.Key(), ch.Name, fmt.Sprintf("invalid expression for %s", ch.Name), err)
				invalidCount++
				g.L.Debug("channel expression invalid",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
			} else {
				statuses[ch.Key()] = nil
			}
			upsertCalcNode(nextNodes, nextDependents, nextUnresolved, nd)
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
		w := g.svc.NewWriter(tx)
		for _, ch := range repairs {
			if err := w.ChangeDataType(ctx, ch.Key(), ch.DataType); err != nil {
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

func (g *calculationGraph) handleChanges(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
	g.mu.Lock()
	analyzer := g.newAnalyzer(nil)
	queued := make(set.Set[Key])
	var unresolvedNames []string
	var updates []Channel
	for chg := range reader {
		ch := chg.Value
		if chg.Variant == xchange.VariantDelete {
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
				g.setNodeStatus(ctx, CalculationStatusFromError(ch.Key(), ch.Name, fmt.Sprintf("invalid expression for %s", ch.Name), err))
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

func (g *calculationGraph) setNodeStatus(ctx context.Context, st *CalculationStatus) {
	if sErr := g.status.Set(ctx, st); sErr != nil {
		g.L.Warn("failed to set error status for channel",
			zap.String("key", st.Key),
			zap.Error(sErr),
		)
	}
}

func (g *calculationGraph) clearNodeStatus(ctx context.Context, key Key) {
	if err := g.status.Delete(ctx, CalculationStatusKey(key)); err != nil {
		g.L.Warn("failed to clear status for channel",
			zap.Stringer("channel", key),
			zap.Error(err),
		)
	}
}

func (g *calculationGraph) newAnalyzer(tx gorp.Tx) *CalculationAnalyzer {
	return NewCalculationAnalyzer(g.svc.NewArcSymbolResolver(tx), parser.Config{
		AllowDashedNames: !g.svc.ShouldValidateNames(),
	})
}

func (g *calculationGraph) inspectNode(
	ctx context.Context,
	tx gorp.Tx,
	ch Channel,
	analyzer *CalculationAnalyzer,
) (calcNode, error) {
	if analyzer == nil {
		analyzer = g.newAnalyzer(tx)
	}
	if ch.Key() == 0 {
		return calcNode{}, errors.Newf("channel %q has no key, cannot inspect", ch.Name)
	}
	result, err := analyzer.Analyze(ctx, ch)
	nd := calcNode{Channel: ch}
	if err == nil {
		nd.DataType = result.ChanDataType
		nd.deps = result.Deps
	} else {
		nd.unresolved = result.Unresolved
	}
	nd.invalid = err != nil
	return nd, err
}

func (g *calculationGraph) reconcileQueued(
	ctx context.Context,
	tx gorp.Tx,
	queued set.Set[Key],
	unresolvedNames []string,
	analyzer *CalculationAnalyzer,
) []Channel {
	g.enqueueUnresolved(unresolvedNames, queued)
	if len(queued) > 0 {
		g.L.Debug("reconciling dependent channels", zap.Int("count", len(queued)))
	}
	updates := make([]Channel, 0)
	for len(queued) > 0 {
		next := make(set.Set[Key])
		for key := range queued {
			nd, ok := g.mu.nodes[key]
			if !ok {
				continue
			}
			refetched := nd.Channel
			if err := g.svc.NewRetrieve().Where(MatchKeys(key)).Entry(&refetched).Exec(ctx, tx); err != nil {
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
				g.setNodeStatus(ctx, CalculationStatusFromError(key, refetched.Name, fmt.Sprintf("invalid expression for %s", refetched.Name), err))
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

func (g *calculationGraph) removeNode(key Key) {
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

func (g *calculationGraph) upsertNode(node calcNode) {
	g.removeNode(node.Key())
	upsertCalcNode(g.mu.nodes, g.mu.dependents, g.mu.unresolvedByName, node)
}

func upsertCalcNode(
	nodes map[Key]calcNode,
	dependents map[Key]set.Set[Key],
	unresolvedByName map[string]set.Set[Key],
	nd calcNode,
) {
	nodes[nd.Key()] = nd
	for _, dep := range nd.deps {
		if dependents[dep] == nil {
			dependents[dep] = make(set.Set[Key])
		}
		dependents[dep].Add(nd.Key())
	}
	for _, name := range nd.unresolved {
		if unresolvedByName[name] == nil {
			unresolvedByName[name] = make(set.Set[Key])
		}
		unresolvedByName[name].Add(nd.Key())
	}
}

func (g *calculationGraph) enqueueDependents(key Key, queued set.Set[Key]) {
	for dep := range g.mu.dependents[key] {
		queued.Add(dep)
	}
}

func (g *calculationGraph) enqueueUnresolved(names []string, queued set.Set[Key]) {
	for _, name := range names {
		for key := range g.mu.unresolvedByName[name] {
			queued.Add(key)
		}
	}
}
