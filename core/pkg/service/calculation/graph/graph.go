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
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/coalesce"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Changes is a batch of channel changes the Graph reconciles as a unit.
type Changes = gorp.TxReader[channel.Key, channel.Channel]

// channelChange is one change to a channel.
type channelChange = gorp.EntryChange[channel.Key, channel.Channel]

type node struct {
	channel.Channel
	calculation calculation.Calculation
	deps        channel.Keys
	unresolved  []string
	invalid     bool
}

// Graph tracks all calculated channels, their dependency edges, and their inferred
// DataTypes. It subscribes to the channel observable and reactively re-inspects
// affected nodes when channels are created, updated, or deleted.
//
// The Graph is the sole writer of calculated channel statuses. The calculation runtime
// reports through SetRuntimeStatus instead of writing the record itself.
type Graph struct {
	alamos.Instrumentation
	db           *gorp.DB
	svc          *channel.Service
	calculations *calculation.Service
	status       *status.Service
	obs          observe.Observer[Changes]
	disconnect   observe.Disconnect
	// disconnectCalculations stops the observation of the calculation table.
	disconnectCalculations observe.Disconnect
	// outputs maps the key of each calculation to the key of its output, so a delete of
	// a calculation, which carries no value, reaches its output.
	outputs struct {
		keys map[calculation.Key]channel.Key
		sync.Mutex
	}
	// changed reconciles the latest change to each channel that commits change, since a
	// reader of a commit must not commit.
	changed *coalesce.Worker[channel.Key, channelChange]
	mu      struct {
		nodes            map[channel.Key]node
		dependents       map[channel.Key]set.Set[channel.Key]
		unresolvedByName map[string]set.Set[channel.Key]
		// rejected holds the channels reported through RejectDefinition.
		rejected set.Set[channel.Key]
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
	// Calculation holds the calculation of each calculated channel. The graph observes
	// it, so an edit of a calculation re-inspects its output.
	//
	// [REQUIRED]
	Calculation *calculation.Service
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
	v.NotNil("calculation", c.Calculation)
	v.NotNil("status", c.Status)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Calculation = override.Nil(c.Calculation, other.Calculation)
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
		calculations:    cfg.Calculation,
		status:          cfg.Status,
		obs:             observe.New[Changes](),
	}
	g.mu.nodes = make(map[channel.Key]node)
	g.mu.dependents = make(map[channel.Key]set.Set[channel.Key])
	g.mu.unresolvedByName = make(map[string]set.Set[channel.Key])
	g.mu.rejected = make(set.Set[channel.Key])
	g.outputs.keys = make(map[calculation.Key]channel.Key)
	if err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.hydrate(ctx, tx)
	}); err != nil {
		return nil, err
	}
	// ctx can end before the Graph closes.
	if g.changed, err = coalesce.Open(
		context.WithoutCancel(ctx),
		coalesce.Config[channel.Key, channelChange]{
			Handle: g.handleChanges,
			Merge:  keepDeletedName,
		},
	); err != nil {
		return nil, err
	}
	g.disconnect = cfg.Channel.Observe().OnChange(func(_ context.Context, r Changes) {
		for ch := range r {
			g.changed.Add(ch.Key, ch)
		}
	})
	g.disconnectCalculations = cfg.Calculation.Observe().OnChange(g.observeCalculations)
	return g, nil
}

// Observe returns an observable that fires with a channel change batch after the Graph
// has reconciled it and committed the resulting statuses and node state. The batch also
// holds an update for each calculated channel the Graph re-inspected and found valid
// because a channel it reads changed. Subscribers that act on calculated channels must
// use this instead of the channel observable, so that their work runs against a
// reconciled graph and their status reports land after the Graph's own.
func (g *Graph) Observe() observe.Observable[Changes] { return g.obs }

// RejectDefinition persists a status the calculation runtime reports when it cannot
// compile or request the definition of the channel with the given key. The Graph then
// drops runtime reports for the channel until ClearRuntimeStatus.
func (g *Graph) RejectDefinition(
	ctx context.Context,
	key channel.Key,
	stat *calculation.Status,
) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.mu.rejected.Add(key)
	return g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.writeStatus(ctx, tx, key, stat)
	})
}

// SetRuntimeStatus persists a status reported by the calculation runtime for the
// channel with the given key when the report changes the stored status. A success
// changes only a stored error. Any other report changes a stored status with a
// different variant, message, or description. Routing runtime reports through the Graph
// keeps a single writer on the status record, so a report and a validity clear apply in
// submission order. The Graph drops a report for a deleted channel, and a report for a
// channel it holds as invalid or rejected: that record describes the channel
// definition. A report that changes nothing takes only the read lock of the DB.
func (g *Graph) SetRuntimeStatus(
	ctx context.Context,
	key channel.Key,
	stat *calculation.Status,
) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if nd, ok := g.mu.nodes[key]; (ok && nd.invalid) || g.mu.rejected.Contains(key) {
		return nil
	}
	var changed bool
	if err := g.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		stored, found, err := g.retrieveStatus(ctx, snap, key)
		changed = changes(stored, found, stat)
		return err
	}); err != nil || !changed {
		return err
	}
	if stat.Variant == status.VariantError {
		g.L.Warn(stat.String())
	} else {
		g.L.Info(stat.String())
	}
	return g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.writeStatus(ctx, tx, key, stat)
	})
}

func changes(stored calculation.Status, found bool, next *calculation.Status) bool {
	if next.Variant == status.VariantSuccess {
		return found && stored.Variant == status.VariantError
	}
	return !found || stored.Variant != next.Variant ||
		stored.Message != next.Message || stored.Description != next.Description
}

// Failed returns true when the stored status of the channel with the given key is an
// error.
func (g *Graph) Failed(ctx context.Context, key channel.Key) (bool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var failed bool
	err := g.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		stored, found, err := g.retrieveStatus(ctx, snap, key)
		failed = found && stored.Variant == status.VariantError
		return err
	})
	return failed, err
}

func (g *Graph) retrieveStatus(
	ctx context.Context,
	snap gorp.Snapshot,
	key channel.Key,
) (calculation.Status, bool, error) {
	var stored calculation.Status
	c, ok, err := g.calculationOf(ctx, snap, key)
	if err != nil || !ok {
		return stored, false, err
	}
	err = g.status.NewRetrieve[struct{}]().
		Where(status.MatchOwners[struct{}](calculation.ResourceID(c.Key))).
		Entry(&stored).
		Exec(ctx, snap)
	if errors.Is(err, query.ErrNotFound) {
		return stored, false, nil
	}
	return stored, err == nil, err
}

func (g *Graph) writeStatus(
	ctx context.Context,
	tx gorp.Tx,
	key channel.Key,
	stat *calculation.Status,
) error {
	c, ok, err := g.calculationOf(ctx, tx, key)
	if err != nil {
		return err
	}
	if !ok {
		g.L.Debug("dropping runtime status for a deleted calculation",
			zap.Stringer("channel", key),
		)
		return nil
	}
	stat.Owner = new(calculation.ResourceID(c.Key))
	return g.status.NewWriter(tx).Set(ctx, stat)
}

// AcceptDefinition acts as ClearRuntimeStatus for the channel with the given key when
// RejectDefinition rejected the channel. It keeps the status of a channel that is not
// rejected: a running calculation owns that record.
func (g *Graph) AcceptDefinition(ctx context.Context, key channel.Key) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.mu.rejected.Contains(key) {
		return nil
	}
	return g.clearRuntimeStatus(ctx, key)
}

// ClearRuntimeStatus removes the status record for the channel with the given key after
// the calculation runtime recovers. A node the Graph holds as invalid keeps its status:
// that record describes the channel definition, not the runtime.
func (g *Graph) ClearRuntimeStatus(ctx context.Context, key channel.Key) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.clearRuntimeStatus(ctx, key)
}

func (g *Graph) clearRuntimeStatus(ctx context.Context, key channel.Key) error {
	g.mu.rejected.Remove(key)
	if nd, ok := g.mu.nodes[key]; ok && nd.invalid {
		return nil
	}
	return g.db.WithTx(ctx, func(tx gorp.Tx) error {
		return g.deleteStatus(ctx, tx, key)
	})
}

func (g *Graph) deleteStatus(ctx context.Context, tx gorp.Tx, key channel.Key) error {
	c, ok, err := g.calculationOf(ctx, tx, key)
	if err != nil || !ok {
		return err
	}
	return g.status.NewWriter(tx).DeleteOwned(ctx, calculation.ResourceID(c.Key))
}

// calculationOf returns the calculation that writes the channel with key, as snap sees
// it, and false when none does.
func (g *Graph) calculationOf(
	ctx context.Context,
	snap gorp.Snapshot,
	key channel.Key,
) (calculation.Calculation, bool, error) {
	calcs, err := g.calculations.Outputs(ctx, snap, key)
	c, ok := calcs[key]
	return c, ok, err
}

// fetchNode returns the channel with key and its calculation, as tx sees them. It
// reports false when the channel does not exist, and returns a nil calculation when no
// calculation writes the channel.
func (g *Graph) fetchNode(
	ctx context.Context,
	tx gorp.Tx,
	key channel.Key,
) (channel.Channel, *calculation.Calculation, bool, error) {
	var ch channel.Channel
	err := g.svc.NewRetrieve().Where(channel.MatchKeys(key)).Entry(&ch).Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return ch, nil, false, nil
	}
	if err != nil {
		return ch, nil, false, err
	}
	c, ok, err := g.calculationOf(ctx, tx, key)
	if err != nil || !ok {
		return ch, nil, true, err
	}
	return ch, &c, true, nil
}

// renameStatus gives the status of c the new name of its output.
func (g *Graph) renameStatus(
	ctx context.Context,
	tx gorp.Tx,
	c calculation.Calculation,
	name string,
) {
	err := g.status.NewWriter(tx).RenameOwned(ctx, calculation.ResourceID(c.Key), name)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		g.L.Warn(
			"failed to rename status for calculation",
			zap.Stringer("channel", c.Output),
			zap.Error(err),
		)
	}
}

// Close disconnects the graph from the channel observable.
func (g *Graph) Close() error {
	g.disconnectCalculations()
	g.disconnect()
	return g.changed.Close()
}

// keepDeletedName keeps the name of a deleted channel when a change of its calculation
// follows the delete, so the Graph can still re-inspect the channels that name it.
func keepDeletedName(waiting, added channelChange) channelChange {
	if added.Value.Name == "" {
		added.Value.Name = waiting.Value.Name
	}
	return added
}

// observeCalculations hands the output of each changed calculation to the Graph, which
// inspects the output again with its stored calculation.
func (g *Graph) observeCalculations(
	_ context.Context,
	r gorp.TxReader[calculation.Key, calculation.Calculation],
) {
	g.outputs.Lock()
	defer g.outputs.Unlock()
	for c := range r {
		output, ok := g.outputs.keys[c.Key]
		if c.Kind == gorp.ChangeDelete {
			delete(g.outputs.keys, c.Key)
		} else {
			output, ok = c.Value.Output, true
			g.outputs.keys[c.Key] = output
		}
		if ok {
			g.changed.Add(output, channelChange{Key: output, Kind: gorp.ChangeUpdate})
		}
	}
}

func (g *Graph) hydrate(ctx context.Context, tx gorp.Tx) error {
	var calcs []calculation.Calculation
	if err := g.calculations.NewRetrieve().Entries(&calcs).Exec(ctx, tx); err != nil {
		return err
	}
	byOutput := make(map[channel.Key]calculation.Calculation, len(calcs))
	outputs := make([]channel.Key, len(calcs))
	for i, c := range calcs {
		byOutput[c.Output], outputs[i] = c, c.Output
		g.outputs.keys[c.Key] = c.Output
	}
	var channels []channel.Channel
	if len(outputs) > 0 {
		if err := g.svc.NewRetrieve().
			Where(channel.MatchKeys(outputs...)).
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
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
			c := byOutput[ch.Key()]
			nd, err := g.inspectNode(ctx, tx, ch, c, analyzer)
			if err != nil {
				statuses[ch.Key()] = calculation.StatusFromError(c.Key, ch.Name, err)
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

// handleChanges reconciles the Graph with the latest change to each channel.
func (g *Graph) handleChanges(
	ctx context.Context,
	latest map[channel.Key]channelChange,
) {
	var (
		updates []channel.Channel
		valid   []channel.Channel
		batch   []channelChange
	)
	g.mu.Lock()
	err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
		analyzer := g.newAnalyzer(tx)
		queued := make(set.Set[channel.Key])
		var unresolvedNames []string
		for _, chg := range latest {
			ch, c, found, err := g.fetchNode(ctx, tx, chg.Key)
			if err != nil {
				g.L.Warn(
					"failed to retrieve changed channel",
					zap.Stringer("channel", chg.Key),
					zap.Error(err),
				)
				continue
			}
			if !found {
				batch = append(batch, channelChange{
					Key:   chg.Key,
					Value: chg.Value,
					Kind:  gorp.ChangeDelete,
				})
				g.L.Debug(
					"channel deleted, removing node and re-inspecting dependents",
					zap.Stringer("channel", chg.Key),
				)
				g.removeNode(chg.Key)
				if chg.Value.Name != "" {
					unresolvedNames = append(unresolvedNames, chg.Value.Name)
				}
				g.enqueueDependents(chg.Key, queued)
				g.mu.rejected.Remove(chg.Key)
				continue
			}
			batch = append(batch, channelChange{
				Key:   chg.Key,
				Value: ch,
				Kind:  gorp.ChangeUpdate,
			})
			unresolvedNames = append(unresolvedNames, ch.Name)
			prev, tracked := g.mu.nodes[chg.Key]
			if c == nil {
				if tracked {
					g.removeNode(chg.Key)
					g.mu.rejected.Remove(chg.Key)
				}
				g.enqueueDependents(chg.Key, queued)
				continue
			}
			if tracked && prev.Name != ch.Name {
				g.renameStatus(ctx, tx, *c, ch.Name)
			}
			nd, err := g.inspectNode(ctx, tx, ch, *c, analyzer)
			if err != nil {
				g.L.Info("calculated channel has invalid expression",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Error(err),
				)
				g.setNodeStatus(
					ctx,
					tx,
					calculation.StatusFromError(c.Key, ch.Name, err),
				)
			} else {
				g.L.Debug("calculated channel inspected",
					zap.Stringer("channel", ch.Key()),
					zap.String("name", ch.Name),
					zap.Stringers("deps", nd.deps),
				)
				// An update to an untracked node holds a coalesced create, so the
				// expression a runtime status describes may be gone.
				if (!tracked && chg.Kind == gorp.ChangeUpdate) ||
					(tracked && (prev.invalid ||
						prev.calculation.Expression != c.Expression)) {
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
		}
		var reconciled []channel.Channel
		reconciled, valid = g.reconcileQueued(
			ctx, tx, queued, unresolvedNames, analyzer,
		)
		updates = append(updates, reconciled...)
		return nil
	})
	g.mu.Unlock()
	if err != nil {
		g.L.Error("failed to apply calculated channel changes", zap.Error(err))
		return
	}
	for _, ch := range valid {
		batch = append(batch, channelChange{
			Key:   ch.Key(),
			Value: ch,
			Kind:  gorp.ChangeUpdate,
		})
	}
	g.obs.Notify(ctx, slices.Values(batch))
	if len(updates) > 0 {
		g.L.Info("updating channel data types", zap.Int("count", len(updates)))
		for _, ch := range updates {
			if err := g.db.WithTx(ctx, func(tx gorp.Tx) error {
				return g.svc.NewWriter(tx).ChangeDataType(ctx, ch.Key(), ch.DataType)
			}); err != nil {
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
	if err := g.deleteStatus(ctx, tx, key); err != nil {
		g.L.Warn(
			"failed to clear status for channel",
			zap.Stringer("channel", key),
			zap.Error(err),
		)
	}
}

// hasDerivative reports whether ops holds a derivative.
func hasDerivative(ops []calculation.Operation) bool {
	return slices.ContainsFunc(ops, func(op calculation.Operation) bool {
		_, ok := op.Variant.(calculation.DerivativeOperation)
		return ok
	})
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
	c calculation.Calculation,
	analyzer *channel.CalculationAnalyzer,
) (node, error) {
	if analyzer == nil {
		analyzer = g.newAnalyzer(tx)
	}
	if ch.Key() == 0 {
		return node{}, errors.Newf("channel %q has no key, cannot inspect", ch.Name)
	}
	result, err := analyzer.AnalyzeExpression(
		ctx,
		ch,
		c.Expression,
		hasDerivative(c.Operations),
	)
	nd := node{Channel: ch, calculation: c}
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
) (updates, valid []channel.Channel) {
	g.enqueueUnresolved(unresolvedNames, queued)
	if len(queued) > 0 {
		g.L.Debug("reconciling dependent channels", zap.Int("count", len(queued)))
	}
	updates = make([]channel.Channel, 0)
	for len(queued) > 0 {
		next := make(set.Set[channel.Key])
		for key := range queued {
			nd, ok := g.mu.nodes[key]
			if !ok {
				continue
			}
			refetched, c, found, err := g.fetchNode(ctx, tx, key)
			if err != nil || !found || c == nil {
				if err != nil {
					g.L.Warn(
						"failed to refetch channel during reconciliation",
						zap.Stringer("channel", key),
						zap.Error(err),
					)
				}
				continue
			}
			newNode, err := g.inspectNode(ctx, tx, refetched, *c, analyzer)
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
					calculation.StatusFromError(c.Key, refetched.Name, err),
				)
				continue
			}
			valid = append(valid, refetched)
			if oldInvalid || c.Expression != nd.calculation.Expression {
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
	return updates, valid
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
