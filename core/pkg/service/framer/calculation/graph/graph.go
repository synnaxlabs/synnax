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
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	// Channel resolves and retrieves calculated channels and builds the Arc symbol
	// resolver used to compile their expressions.
	//
	// [REQUIRED]
	Channel *channel.Service
	// DB holds the channels that the graph compiles.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Compile compiles a calculated channel. It is injectable for testing.
	//
	// [OPTIONAL] - defaults to compiler.Compile.
	Compile func(context.Context, *gorp.DB, ...compiler.Config) (compiler.Module, error)
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[Config] = Config{}

// ErrCycle is returned by Add when a calculated channel reads itself through other
// calculated channels.
var ErrCycle = errors.New("circular dependency detected")

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.DB = override.Nil(c.DB, other.DB)
	c.Compile = override.Nil(c.Compile, other.Compile)
	return c
}

func (c Config) Validate() error {
	v := validate.New("calculation.graph")
	v.NotNil("channel", c.Channel)
	v.NotNil("db", c.DB)
	return v.Error()
}

type channelInfo struct {
	module   compiler.Module
	calcDeps []channel.Key
	// baseDeps are the concrete channels the channel reads, directly or through other
	// calculated channels.
	baseDeps      set.Set[channel.Key]
	groupID       int
	explicitCount int
	depCount      int
	processing    bool
}

type groupInfo struct {
	baseDeps set.Set[channel.Key]
	members  set.Set[channel.Key]
}

type Graph struct {
	cfg      Config
	channels map[channel.Key]*channelInfo
	groups   map[int]*groupInfo
	alamos.Instrumentation
	nextGroupID int
}

// New creates a new Graph with the provided configuration.
func New(cfgs ...Config) (*Graph, error) {
	cfg, err := config.New(Config{Compile: compiler.Compile}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Graph{
		Instrumentation: cfg.Instrumentation,
		cfg:             cfg,
		channels:        make(map[channel.Key]*channelInfo),
		groups:          make(map[int]*groupInfo),
		nextGroupID:     0,
	}, nil
}

// Add compiles a channel and its dependencies, then assigns it to a group.
// Increments the explicit reference count for the channel. A failed Add leaves the
// graph as it was.
func (g *Graph) Add(ctx context.Context, ch channel.Channel) (err error) {
	if info := g.channels[ch.Key()]; info != nil {
		oldCount := info.explicitCount
		info.explicitCount++
		g.L.Debug("channel request added",
			zap.String("channel", ch.Key().String()),
			zap.Uint32("explicit_count", uint32(oldCount)),
			zap.Uint32("new_explicit_count", uint32(info.explicitCount)),
			zap.Uint32("dependency_count", uint32(info.depCount)),
		)
		return nil
	}
	defer g.restoreOnError(&err)()
	return g.addInternal(ctx, ch, true)
}

// restoreOnError saves the state of the graph. The returned function restores that
// state when *err is not nil.
func (g *Graph) restoreOnError(err *error) func() {
	channels := make(map[channel.Key]*channelInfo, len(g.channels))
	for k, info := range g.channels {
		saved := *info
		channels[k] = &saved
	}
	groups := make(map[int]*groupInfo, len(g.groups))
	for k, group := range g.groups {
		groups[k] = &groupInfo{baseDeps: group.baseDeps, members: group.members.Copy()}
	}
	nextGroupID := g.nextGroupID
	return func() {
		if *err != nil {
			g.channels, g.groups, g.nextGroupID = channels, groups, nextGroupID
		}
	}
}

// Update recompiles a channel with a new expression and updates its dependencies and
// group assignment. Preserves reference counts - use this when a channel's calculation
// changes but the same users are still requesting it. If the channel's DataType
// changes, all downstream channels that depend on it are recompiled in topological
// order. A failed Update leaves the graph as it was.
func (g *Graph) Update(ctx context.Context, ch channel.Channel) (err error) {
	info, err := g.getChannelInfo(ch.Key())
	if err != nil {
		return err
	}
	defer g.restoreOnError(&err)()
	mod, err := g.compile(ctx, ch)
	if err != nil {
		g.logCompileError(ctx, err, ch)
		return errors.Wrapf(err, "failed to compile calculated channel %s", ch)
	}
	var dependents []compiler.Module
	if oldDataType := info.module.Channel.DataType; oldDataType != mod.Channel.DataType {
		g.L.Debug("channel DataType changed, recompiling dependents",
			zap.String("channel", ch.Key().String()),
			zap.String("old_type", string(oldDataType)),
			zap.String("new_type", string(mod.Channel.DataType)),
		)
		if dependents, err = g.compileDependents(ctx, ch.Key()); err != nil {
			return err
		}
	}
	if err := g.updateSingle(ctx, ch, info, mod); err != nil {
		return err
	}
	for _, dep := range dependents {
		g.channels[dep.Channel.Key()].module = dep
	}
	return nil
}

// compileDependents compiles all transitive dependents of changedKey in topological
// order (parents before children). If a compile changes a channel's DataType, its own
// dependents are already in the ordered list.
func (g *Graph) compileDependents(
	ctx context.Context,
	changedKey channel.Key,
) ([]compiler.Module, error) {
	ordered := g.collectDependentsTopological(changedKey)
	if len(ordered) == 0 {
		return nil, nil
	}
	fetched, err := g.fetchChannels(ctx, ordered)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"failed to fetch dependent channels for recompilation",
		)
	}
	byKey := make(map[channel.Key]channel.Channel, len(fetched))
	for _, ch := range fetched {
		byKey[ch.Key()] = ch
	}
	mods := make([]compiler.Module, 0, len(ordered))
	for _, key := range ordered {
		ch, ok := byKey[key]
		if !ok {
			continue
		}
		mod, err := g.compile(ctx, ch)
		if err != nil {
			g.logCompileError(ctx, err, ch)
			return nil, errors.Wrapf(
				err,
				"failed to recompile dependent channel %s",
				ch,
			)
		}
		mods = append(mods, mod)
	}
	return mods, nil
}

// collectDependentsTopological returns all channels in the graph that transitively
// depend on root, ordered so that a channel always appears after all of its own
// dependencies. This is a simple BFS followed by a reverse-post-order DFS over the
// subset, which is cheap for the small graphs we operate on (tens to low hundreds of
// channels).
func (g *Graph) collectDependentsTopological(root channel.Key) []channel.Key {
	// Build the set of all transitive dependents via BFS.
	dependents := make(set.Set[channel.Key])
	queue := []channel.Key{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for key, info := range g.channels {
			if dependents.Contains(key) {
				continue
			}
			if slices.Contains(info.calcDeps, current) {
				dependents.Add(key)
				queue = append(queue, key)
			}
		}
	}
	if len(dependents) == 0 {
		return nil
	}

	// Topological sort via DFS post-order over the dependent subset.
	visited := make(set.Set[channel.Key])
	var ordered []channel.Key
	var visit func(channel.Key)
	visit = func(key channel.Key) {
		if visited.Contains(key) {
			return
		}
		visited.Add(key)
		info := g.channels[key]
		if info == nil {
			return
		}
		for _, dep := range info.calcDeps {
			if dependents.Contains(dep) {
				visit(dep)
			}
		}
		ordered = append(ordered, key)
	}
	for key := range dependents {
		visit(key)
	}
	return ordered
}

// updateSingle sets mod as the module of ch and updates its dependencies and group
// assignment without cascading to dependents.
func (g *Graph) updateSingle(
	ctx context.Context,
	ch channel.Channel,
	info *channelInfo,
	mod compiler.Module,
) error {
	// Store current state
	oldExplicitCount := info.explicitCount
	oldDepCount := info.depCount
	oldCalcDeps := info.calcDeps
	oldGroupID := info.groupID

	// Mark as processing to detect circular dependencies during recompilation
	info.processing = true
	defer func() { info.processing = false }()

	depChannels, newCalcDeps, err := g.addCalcDeps(
		ctx,
		ch,
		mod.Dependencies.Reads.Slice(),
	)
	if err != nil {
		return err
	}
	if err = g.updateDepCounts(ch, oldCalcDeps, newCalcDeps); err != nil {
		return err
	}

	// Resolve new base dependencies
	baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		return err
	}

	// Assign to new group (may be same or different)
	newGroupID := g.assignToGroup(baseDeps, newCalcDeps)

	// Update channel info with new data while preserving reference counts
	// Do this BEFORE recalculating group base deps so the new module is used
	info.module = mod
	info.groupID = newGroupID
	info.calcDeps = newCalcDeps
	info.baseDeps = baseDeps
	info.explicitCount = oldExplicitCount
	info.depCount = oldDepCount

	if newGroupID, err = g.moveGroup(ctx, ch, oldGroupID, newGroupID); err != nil {
		return err
	}
	return g.recalculateGroupBaseDeps(ctx, newGroupID)
}

// addCalcDeps fetches the channels in dependencies and adds each calculated one to
// the graph as a dependency of ch. It returns the fetched channels and the keys of
// the calculated ones.
func (g *Graph) addCalcDeps(
	ctx context.Context,
	ch channel.Channel,
	dependencies []channel.Key,
) ([]channel.Channel, []channel.Key, error) {
	if len(dependencies) == 0 {
		return nil, nil, nil
	}
	depChannels, err := g.fetchChannels(ctx, dependencies)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to retrieve dependency channels")
	}
	var newCalcDeps []channel.Key
	for _, depCh := range depChannels {
		if !depCh.IsCalculated() {
			continue
		}
		// Check for circular dependencies before adding
		if err = g.checkCircularDependency(ch.Key(), depCh.Key()); err != nil {
			return nil, nil, err
		}
		// Add dependency if not already in graph
		if err = g.addInternal(ctx, depCh, false); err != nil {
			return nil, nil, errors.Wrapf(
				err,
				"failed to add calculated dependency %v",
				depCh.Key(),
			)
		}
		newCalcDeps = append(newCalcDeps, depCh.Key())
	}
	return depChannels, newCalcDeps, nil
}

// updateDepCounts increments the dependency count of each calculated dependency that
// ch gained and decrements it for each one that ch lost, removing a channel whose
// counts reach zero.
func (g *Graph) updateDepCounts(
	ch channel.Channel,
	oldCalcDeps, newCalcDeps []channel.Key,
) error {
	oldDepsSet := set.New(oldCalcDeps...)
	newDepsSet := set.New(newCalcDeps...)
	for key := range newDepsSet {
		if oldDepsSet.Contains(key) {
			continue
		}
		if depInfo := g.channels[key]; depInfo != nil {
			g.incrementDepCount(key, depInfo, ch.Key())
		}
	}
	for key := range oldDepsSet {
		if newDepsSet.Contains(key) {
			continue
		}
		if err := g.decrementDepCount(key, ch.Key()); err != nil {
			return err
		}
	}
	return nil
}

// incrementDepCount increments the dependency count of info, the channel with key,
// which depender now depends on.
func (g *Graph) incrementDepCount(
	key channel.Key,
	info *channelInfo,
	depender channel.Key,
) {
	oldDepCount := info.depCount
	info.depCount++
	g.L.Debug("dependency count incremented",
		zap.String("channel", key.String()),
		zap.Uint32("dependency_count", uint32(oldDepCount)),
		zap.Uint32("new_dependency_count", uint32(info.depCount)),
		zap.String("depender", depender.String()),
	)
}

// decrementDepCount decrements the dependency count of the channel with key, which
// formerDepender no longer depends on, and removes the channel when its counts reach
// zero.
func (g *Graph) decrementDepCount(key, formerDepender channel.Key) error {
	depInfo := g.channels[key]
	if depInfo == nil {
		return nil
	}
	oldDepCount := depInfo.depCount
	depInfo.depCount--
	g.L.Debug("dependency count decremented",
		zap.String("channel", key.String()),
		zap.Uint32("dependency_count", uint32(oldDepCount)),
		zap.Uint32("new_dependency_count", uint32(depInfo.depCount)),
		zap.String("former_depender", formerDepender.String()),
	)
	if depInfo.explicitCount != 0 || depInfo.depCount != 0 {
		return nil
	}
	g.L.Debug(
		"channel eligible for removal",
		zap.String("channel", key.String()),
		zap.String("reason", "explicit and dependency counts reached zero"),
	)
	return g.removeChannel(key)
}

// moveGroup moves ch from the group oldGroupID to newGroupID when they differ, then
// merges the groups of ch and its dependents. It returns the ID of the group of ch.
func (g *Graph) moveGroup(
	ctx context.Context,
	ch channel.Channel,
	oldGroupID, newGroupID int,
) (int, error) {
	if oldGroupID == newGroupID {
		g.L.Debug("channel updated in same group",
			zap.String("channel", ch.Key().String()),
			zap.Int("group_id", newGroupID),
			zap.String("dependency_tree", g.formatDependencyTree(ctx, ch.Key())),
		)
		return newGroupID, nil
	}
	g.L.Debug("channel moved to different group",
		zap.String("channel", ch.Key().String()),
		zap.Int("old_group_id", oldGroupID),
		zap.Int("new_group_id", newGroupID),
		zap.String("dependency_tree", g.formatDependencyTree(ctx, ch.Key())),
	)
	if oldGroup := g.groups[oldGroupID]; oldGroup != nil {
		oldGroup.members.Remove(ch.Key())
		if len(oldGroup.members) == 0 {
			delete(g.groups, oldGroupID)
		} else if err := g.recalculateGroupBaseDeps(ctx, oldGroupID); err != nil {
			return 0, err
		}
	}
	g.groups[newGroupID].members.Add(ch.Key())
	return g.mergeGroups(
		append(g.collectDependentsTopological(ch.Key()), ch.Key()),
	), nil
}

// checkCircularDependency checks if adding a dependency would create a circular
// dependency.
func (g *Graph) checkCircularDependency(source, target channel.Key) error {
	if source == target {
		err := errors.Newf(
			"circular dependency detected: channel %v depends on itself",
			source,
		)
		g.L.Info("circular dependency detected",
			zap.String("channel", source.String()),
			zap.String("dependency_chain", fmt.Sprintf("%s → %s", source, source)),
			zap.Error(err),
		)
		return err
	}

	// Check if target (or any of its dependencies) depends on source
	visited := make(set.Set[channel.Key])
	var chain []channel.Key
	var checkDeps func(channel.Key) error
	checkDeps = func(key channel.Key) error {
		if key == source {
			// Build the full chain for logging
			chainStrs := make([]string, 0, len(chain)+1)
			for _, k := range chain {
				chainStrs = append(chainStrs, k.String())
			}
			chainStrs = append(chainStrs, source.String())
			err := errors.Newf(
				"circular dependency detected involving channel %v",
				source,
			)
			g.L.Info("circular dependency detected",
				zap.String("channel", source.String()),
				zap.String("dependency_chain", strings.Join(chainStrs, " → ")),
				zap.Error(err),
			)
			return err
		}
		if visited.Contains(key) {
			return nil
		}
		visited.Add(key)
		chain = append(chain, key)

		info := g.channels[key]
		if info == nil {
			chain = chain[:len(chain)-1]
			return nil
		}

		for _, depKey := range info.calcDeps {
			if err := checkDeps(depKey); err != nil {
				return err
			}
		}
		chain = chain[:len(chain)-1]
		return nil
	}

	return checkDeps(target)
}

// recalculateGroupBaseDeps recalculates the base dependencies for a group based on its
// current members.
func (g *Graph) recalculateGroupBaseDeps(ctx context.Context, groupID int) error {
	group := g.groups[groupID]
	if group == nil {
		return nil
	}

	newBaseDeps := make(set.Set[channel.Key])
	for memberKey := range group.members {
		info := g.channels[memberKey]
		if info == nil {
			continue
		}

		// Get all dependencies for this member
		deps := info.module.Dependencies.Reads.Slice()
		if len(deps) == 0 {
			continue
		}

		depChannels, err := g.fetchChannels(ctx, deps)
		if err != nil {
			return errors.Wrap(
				err,
				"failed to retrieve dependency channels for base deps recalculation",
			)
		}

		baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
		if err != nil {
			return err
		}

		for dep := range baseDeps {
			newBaseDeps.Add(dep)
		}
	}

	group.baseDeps = newBaseDeps
	return nil
}

// addInternal compiles a channel and its dependencies, then assigns it to a group.
func (g *Graph) addInternal(
	ctx context.Context,
	ch channel.Channel,
	explicit bool,
) error {
	if info := g.channels[ch.Key()]; info != nil {
		if info.processing {
			return errors.Wrapf(ErrCycle, "channel %v", ch.Key())
		}
		return nil
	}

	info := &channelInfo{processing: true}
	g.channels[ch.Key()] = info
	defer func() { info.processing = false }()

	mod, err := g.compile(ctx, ch)
	if err != nil {
		g.logCompileError(ctx, err, ch)
		return errors.Wrapf(err, "failed to compile calculated channel %s", ch)
	}
	depChannels, calcDeps, err := g.addDeps(ctx, ch, mod.Dependencies.Reads.Slice())
	if err != nil {
		return err
	}
	baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		return err
	}

	groupID := g.assignToGroup(baseDeps, calcDeps)

	info.module = mod
	info.groupID = groupID
	info.calcDeps = calcDeps
	info.baseDeps = baseDeps
	if explicit {
		info.explicitCount = 1
	}

	g.groups[groupID].members.Add(ch.Key())

	// Log the addition with full dependency tree
	g.L.Debug("channel added to graph",
		zap.String("channel", ch.Key().String()),
		zap.String("dependency_tree", g.formatDependencyTree(ctx, ch.Key())),
		zap.Uint32("explicit_count", uint32(info.explicitCount)),
		zap.Uint32("dependency_count", uint32(info.depCount)),
		zap.Int("group_id", groupID),
	)

	return nil
}

// addDeps fetches the channels in dependencies, adds each calculated one to the graph,
// and counts ch as its dependent. It returns the fetched channels and the keys of the
// calculated ones.
func (g *Graph) addDeps(
	ctx context.Context,
	ch channel.Channel,
	dependencies []channel.Key,
) ([]channel.Channel, []channel.Key, error) {
	if len(dependencies) == 0 {
		return nil, nil, nil
	}
	depChannels, err := g.fetchChannels(ctx, dependencies)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to retrieve dependency channels")
	}
	var calcDeps []channel.Key
	for _, depCh := range depChannels {
		if !depCh.IsCalculated() {
			continue
		}
		if err = g.addInternal(ctx, depCh, false); err != nil {
			return nil, nil, errors.Wrapf(
				err,
				"failed to add calculated dependency %v",
				depCh.Key(),
			)
		}
		g.incrementDepCount(depCh.Key(), g.channels[depCh.Key()], ch.Key())
		calcDeps = append(calcDeps, depCh.Key())
	}
	return depChannels, calcDeps, nil
}

// fetchChannels retrieves channels by their keys.
func (g *Graph) fetchChannels(
	ctx context.Context,
	keys []channel.Key,
) ([]channel.Channel, error) {
	var channels []channel.Channel
	if err := g.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return g.cfg.Channel.NewRetrieve().
			Entries(&channels).
			Where(channel.MatchKeys(keys...)).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	return channels, nil
}

func (g *Graph) compile(
	ctx context.Context,
	ch channel.Channel,
) (compiler.Module, error) {
	return g.cfg.Compile(ctx, g.cfg.DB, compiler.Config{
		ChannelService: g.cfg.Channel,
		Channel:        ch,
	})
}

// resolveBaseDependencies recursively resolves all dependencies to find the concrete
// (non-calculated) base channels that this channel ultimately depends on. depChannels
// should be the already-fetched channels for mod.Dependencies.Reads.
func (g *Graph) resolveBaseDependencies(
	ctx context.Context,
	depChannels []channel.Channel,
) (set.Set[channel.Key], error) {
	baseDeps := make(set.Set[channel.Key])

	if len(depChannels) == 0 {
		return baseDeps, nil
	}

	for _, depCh := range depChannels {
		if !depCh.IsCalculated() {
			// An index of a calculated channel is not a base dependency.
			if !g.isCalculatedIndex(depCh.Key()) {
				baseDeps.Add(depCh.Key())
			}
			continue
		}
		recursiveDeps, err := g.resolveCalculatedBaseDeps(ctx, depCh.Key())
		if err != nil {
			return nil, err
		}
		baseDeps.Add(recursiveDeps.Slice()...)
	}

	return baseDeps, nil
}

// resolveCalculatedBaseDeps returns the base dependencies of the calculated channel
// with key.
func (g *Graph) resolveCalculatedBaseDeps(
	ctx context.Context,
	key channel.Key,
) (set.Set[channel.Key], error) {
	info, err := g.getChannelInfo(key)
	if err != nil {
		return nil, err
	}
	depDeps := info.module.Dependencies.Reads.Slice()
	var depDepChannels []channel.Channel
	if len(depDeps) > 0 {
		if depDepChannels, err = g.fetchChannels(ctx, depDeps); err != nil {
			return nil, errors.Wrap(
				err,
				"failed to retrieve recursive dependency channels",
			)
		}
	}
	return g.resolveBaseDependencies(ctx, depDepChannels)
}

// isCalculatedIndex reports whether the channel with key is the index of a calculated
// channel in the graph.
func (g *Graph) isCalculatedIndex(key channel.Key) bool {
	for _, info := range g.channels {
		if info.module.Channel.Index() == key {
			return true
		}
	}
	return false
}

// assignToGroup returns the group for a channel with the given dependencies. A channel
// that reads calculated channels joins their group, merging their groups when they
// differ. Otherwise, it joins the best-fit group by base dependencies or a new one.
func (g *Graph) assignToGroup(
	baseDeps set.Set[channel.Key],
	calcDeps []channel.Key,
) int {
	if len(calcDeps) > 0 {
		groupID := g.mergeGroups(calcDeps)
		group := g.groups[groupID]
		group.baseDeps = group.baseDeps.Copy().Add(baseDeps.Slice()...)
		return groupID
	}
	var (
		bestGroup        int
		bestGroupFound   bool
		smallestDiff     = -1
		shouldExtendBest bool
	)

	baseDepStr := formatKeys(baseDeps)

	for groupID, group := range g.groups {
		if group.baseDeps.Equal(baseDeps) {
			g.L.Debug(
				"channel assigned to existing group with exact base dependency match",
				zap.Int("group_id", groupID),
				zap.String("base_dependencies", baseDepStr),
			)
			return groupID
		}

		if baseDeps.IsSubsetOf(group.baseDeps) {
			diff := len(group.baseDeps) - len(baseDeps)
			if !bestGroupFound || diff < smallestDiff {
				bestGroup = groupID
				bestGroupFound = true
				smallestDiff = diff
				shouldExtendBest = false
			}
		} else if group.baseDeps.IsSubsetOf(baseDeps) {
			diff := len(baseDeps) - len(group.baseDeps)
			if !bestGroupFound || diff < smallestDiff {
				bestGroup = groupID
				bestGroupFound = true
				smallestDiff = diff
				shouldExtendBest = true
			}
		}
	}

	if bestGroupFound {
		g.joinGroup(bestGroup, shouldExtendBest, baseDeps, baseDepStr)
		return bestGroup
	}

	return g.newGroup(baseDeps, baseDepStr)
}

// joinGroup assigns a channel with baseDeps to the group groupID, whose base
// dependencies become baseDeps when extend is true. baseDepStr formats baseDeps for
// logging.
func (g *Graph) joinGroup(
	groupID int,
	extend bool,
	baseDeps set.Set[channel.Key],
	baseDepStr string,
) {
	if !extend {
		g.L.Debug("channel assigned to existing group as subset",
			zap.Int("group_id", groupID),
			zap.String("base_dependencies", baseDepStr),
		)
		return
	}
	g.L.Debug("extending existing group with new base dependencies",
		zap.Int("group_id", groupID),
		zap.String("new_base_dependencies", baseDepStr),
	)
	g.groups[groupID].baseDeps = baseDeps.Copy()
}

// newGroup adds a new group with baseDeps and returns its ID. baseDepStr formats
// baseDeps for logging.
func (g *Graph) newGroup(baseDeps set.Set[channel.Key], baseDepStr string) int {
	newGroupID := g.nextGroupID
	g.nextGroupID++
	g.groups[newGroupID] = &groupInfo{
		baseDeps: baseDeps.Copy(),
		members:  make(set.Set[channel.Key]),
	}
	g.L.Debug("new group created",
		zap.Int("group_id", newGroupID),
		zap.String("base_dependencies", baseDepStr),
		zap.String("reason", "no suitable existing group found"),
	)
	return newGroupID
}

// formatKeys joins keys for logging.
func formatKeys(keys set.Set[channel.Key]) string {
	strs := make([]string, 0, len(keys))
	for key := range keys {
		strs = append(strs, key.String())
	}
	return strings.Join(strs, ", ")
}

// mergeGroups moves every member of the groups of the given channels into the
// lowest-numbered of those groups and returns its ID. Each key must be in the graph.
func (g *Graph) mergeGroups(keys []channel.Key) int {
	ids := make(set.Set[int], len(keys))
	for _, key := range keys {
		ids.Add(g.channels[key].groupID)
	}
	target := slices.Min(ids.Slice())
	targetGroup := g.groups[target]
	targetGroup.baseDeps = targetGroup.baseDeps.Copy()
	for id := range ids {
		if id == target {
			continue
		}
		source := g.groups[id]
		for member := range source.members {
			g.channels[member].groupID = target
			targetGroup.members.Add(member)
		}
		targetGroup.baseDeps.Add(source.baseDeps.Slice()...)
		delete(g.groups, id)
		g.L.Debug("group merged", zap.Int("source", id), zap.Int("target", target))
	}
	return target
}

// CalculateGrouped returns all modules grouped by their assigned group keys.
// Modules within each group are sorted in topological order (dependencies first).
func (g *Graph) CalculateGrouped() map[int][]compiler.Module {
	result := make(map[int][]compiler.Module, len(g.groups))

	for groupID, group := range g.groups {
		modules := make([]compiler.Module, 0, len(group.members))
		for _, key := range slices.Sorted(maps.Keys(group.members)) {
			modules = append(modules, g.channels[key].module)
		}
		sorted, err := g.topologicalSortGroup(groupID, modules)
		if err != nil {
			continue
		}
		result[groupID] = sorted
	}

	return result
}

// CalculateFlat returns all modules in a flat list sorted in topological order
// (dependencies first). Unlike CalculateGrouped, this does not group modules and
// returns a single sorted list.
func (g *Graph) CalculateFlat() []compiler.Module {
	if len(g.channels) == 0 {
		return nil
	}

	allChannels := make(set.Set[channel.Key])
	for key := range g.channels {
		allChannels.Add(key)
	}

	graph := make(map[channel.Key][]channel.Key)
	for key, info := range g.channels {
		var deps []channel.Key
		for depKey := range info.module.Dependencies.Reads {
			if allChannels.Contains(depKey) {
				deps = append(deps, depKey)
			}
		}
		graph[key] = deps
	}

	var (
		visited = make(set.Set[channel.Key])
		inStack = make(set.Set[channel.Key])
		result  []channel.Key
		visit   func(channel.Key) error
	)
	visit = func(key channel.Key) error {
		if visited.Contains(key) {
			return nil
		}
		if inStack.Contains(key) {
			return errors.Newf("circular dependency detected involving channel %v", key)
		}
		inStack.Add(key)
		for _, dep := range graph[key] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		inStack.Remove(key)
		visited.Add(key)
		result = append(result, key)
		return nil
	}

	for key := range g.channels {
		if err := visit(key); err != nil {
			continue
		}
	}

	sortedModules := make([]compiler.Module, 0, len(result))
	for _, key := range result {
		sortedModules = append(sortedModules, g.channels[key].module)
	}

	return sortedModules
}

// CalculatedKeys returns the set of all calculated channel keys managed by this
// allocator.
func (g *Graph) CalculatedKeys() set.Set[channel.Key] {
	keys := make(set.Set[channel.Key])
	for key := range g.channels {
		keys.Add(key)
	}
	return keys
}

// Contains returns whether the graph holds the calculated channel with the given key.
func (g *Graph) Contains(key channel.Key) bool {
	_, ok := g.channels[key]
	return ok
}

// ConcreteBaseKeys returns the set of all concrete (non-calculated) base channel keys
// that the calculated channels depend on.
func (g *Graph) ConcreteBaseKeys() set.Set[channel.Key] {
	allBaseDeps := make(set.Set[channel.Key])
	for _, group := range g.groups {
		for dep := range group.baseDeps {
			allBaseDeps.Add(dep)
		}
	}
	return allBaseDeps
}

// BaseKeys returns the concrete channels that the calculated channel with the given key
// reads, directly or through other calculated channels. It returns an error if the
// graph does not hold the channel.
func (g *Graph) BaseKeys(key channel.Key) (set.Set[channel.Key], error) {
	info, err := g.getChannelInfo(key)
	if err != nil {
		return nil, err
	}
	return info.baseDeps, nil
}

func (g *Graph) topologicalSortGroup(
	groupKey int,
	modules []compiler.Module,
) ([]compiler.Module, error) {
	channelsInGroup := make(set.Set[channel.Key])
	for _, mod := range modules {
		channelsInGroup.Add(mod.Channel.Key())
	}

	graph := make(map[channel.Key][]channel.Key)
	for _, mod := range modules {
		var deps []channel.Key
		for _, depKey := range slices.Sorted(maps.Keys(mod.Dependencies.Reads)) {
			if channelsInGroup.Contains(depKey) {
				deps = append(deps, depKey)
			}
		}
		graph[mod.Channel.Key()] = deps
	}

	var (
		visited = make(set.Set[channel.Key])
		inStack = make(set.Set[channel.Key])
		result  []channel.Key
		visit   func(channel.Key) error
	)
	visit = func(key channel.Key) error {
		if visited.Contains(key) {
			return nil
		}
		if inStack.Contains(key) {
			return errors.Newf("circular dependency detected in group %v", groupKey)
		}
		inStack.Add(key)
		for _, dep := range graph[key] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		inStack.Remove(key)
		visited.Add(key)
		result = append(result, key)
		return nil
	}

	for _, mod := range modules {
		if err := visit(mod.Channel.Key()); err != nil {
			return nil, err
		}
	}

	sortedModules := make([]compiler.Module, 0, len(result))
	for _, key := range result {
		sortedModules = append(sortedModules, g.channels[key].module)
	}
	return sortedModules, nil
}

func (g *Graph) getChannelInfo(key channel.Key) (*channelInfo, error) {
	info := g.channels[key]
	if info == nil {
		return nil, errors.Wrapf(
			query.ErrNotFound,
			"channel %v not found in allocator",
			key,
		)
	}
	return info, nil
}

// logCompileError logs detailed diagnostic information when a calculated channel
// fails to compile, including its expression, data type, and dependency graph. It logs
// nothing for channel.ErrLookupsChanged, which the caller retries.
func (g *Graph) logCompileError(
	ctx context.Context,
	err error,
	ch channel.Channel,
) {
	if errors.Is(err, channel.ErrLookupsChanged) {
		return
	}
	fields := []zap.Field{
		zap.String("channel", ch.String()),
		zap.String("expression", ch.Expression),
		zap.String("data_type", string(ch.DataType)),
		zap.Error(err),
	}
	deps := g.fetchDependencyDiagnostics(ctx, ch)
	if len(deps) > 0 {
		depStrs := make([]string, len(deps))
		for i, dep := range deps {
			s := fmt.Sprintf("%s %s", dep, dep.DataType)
			if dep.IsCalculated() {
				s += " expression: " + dep.Expression
			}
			depStrs[i] = s
		}
		fields = append(fields, zap.Strings("dependencies", depStrs))
	}
	g.L.Error("failed to compile calculated channel", fields...)
}

// fetchDependencyDiagnostics attempts to discover the channels referenced by an
// expression. This is best-effort for diagnostics only.
func (g *Graph) fetchDependencyDiagnostics(
	ctx context.Context,
	ch channel.Channel,
) []channel.Channel {
	// Try preProcess to discover channel references from the expression
	prog, preErr := compiler.PreProcess(ctx, g.cfg.DB, compiler.Config{
		ChannelService: g.cfg.Channel,
		Channel:        ch,
	})
	if preErr != nil || len(prog.Functions) == 0 {
		return nil
	}
	keys := make([]channel.Key, 0)
	for k := range prog.Functions[0].Channels.Read {
		keys = append(keys, channel.Key(k))
	}
	if len(keys) == 0 {
		return nil
	}
	deps, fetchErr := g.fetchChannels(ctx, keys)
	if fetchErr != nil {
		return nil
	}
	return deps
}

// formatDependencyTree builds a string representation of the full dependency tree for a
// channel. Returns a string like: "ch1 → [calculated: ch2, ch3] → [base: ch4, ch5,
// ch6]".
func (g *Graph) formatDependencyTree(ctx context.Context, key channel.Key) string {
	info := g.channels[key]
	if info == nil {
		return key.String()
	}

	// Start with the channel itself
	parts := []string{key.String()}

	// Add calculated dependencies if any
	if len(info.calcDeps) > 0 {
		calcDepStrs := make([]string, len(info.calcDeps))
		for i, dep := range info.calcDeps {
			calcDepStrs[i] = dep.String()
		}
		parts = append(
			parts,
			fmt.Sprintf("[calculated: %s]", strings.Join(calcDepStrs, ", ")),
		)
	}

	// Resolve and add base dependencies
	deps := info.module.Dependencies.Reads.Slice()
	if len(deps) > 0 {
		depChannels, err := g.fetchChannels(ctx, deps)
		if err == nil {
			baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
			if err == nil && len(baseDeps) > 0 {
				baseDepStrings := make([]string, 0, len(baseDeps))
				for dep := range baseDeps {
					baseDepStrings = append(baseDepStrings, dep.String())
				}
				parts = append(
					parts,
					fmt.Sprintf("[base: %s]", strings.Join(baseDepStrings, ", ")),
				)
			}
		}
	}

	return strings.Join(parts, " → ")
}

// MarshalLogObject implements zapcore.ObjectMarshaler for groupInfo.
func (g *groupInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("member_count", len(g.members))
	enc.AddInt("base_dep_count", len(g.baseDeps))

	// Add member keys
	memberKeys := make([]string, 0, len(g.members))
	for key := range g.members {
		memberKeys = append(memberKeys, key.String())
	}
	enc.AddString("members", strings.Join(memberKeys, ", "))

	// Add base dependency keys
	baseDepKeys := make([]string, 0, len(g.baseDeps))
	for key := range g.baseDeps {
		baseDepKeys = append(baseDepKeys, key.String())
	}
	enc.AddString("base_dependencies", strings.Join(baseDepKeys, ", "))

	return nil
}

// Remove decrements the explicit reference count for a channel. When both explicit and
// dependency counts reach 0, removes the channel and cascades to dependencies. Returns
// true if the channel was found in the graph (i.e., it was a calculated channel).
func (g *Graph) Remove(key channel.Key) (bool, error) {
	info, ok := g.channels[key]
	if !ok {
		g.L.Debug("channel removal requested but not found in graph",
			zap.String("channel", key.String()),
		)
		return false, nil
	}
	oldExplicitCount := info.explicitCount
	info.explicitCount--
	g.L.Debug("channel request removed",
		zap.String("channel", key.String()),
		zap.Uint32("explicit_count", uint32(oldExplicitCount)),
		zap.Uint32("new_explicit_count", uint32(info.explicitCount)),
		zap.Uint32("dependency_count", uint32(info.depCount)),
	)
	if info.explicitCount > 0 || info.depCount > 0 {
		g.L.Debug(
			"channel retained in graph",
			zap.String("channel", key.String()),
			zap.String(
				"reason",
				fmt.Sprintf(
					"explicit_count=%d or dependency_count=%d is non-zero",
					info.explicitCount,
					info.depCount,
				),
			),
		)
		return true, nil
	}
	g.L.Debug("channel eligible for removal",
		zap.String("channel", key.String()),
		zap.String("reason", "explicit and dependency counts reached zero"),
	)
	return true, g.removeChannel(key)
}

// removeChannel removes a channel and cascades to its dependencies.
func (g *Graph) removeChannel(key channel.Key) error {
	chInfo, err := g.getChannelInfo(key)
	if err != nil {
		return err
	}

	groupID := chInfo.groupID
	if group := g.groups[groupID]; group != nil {
		group.members.Remove(key)
		remainingMembers := len(group.members)
		g.L.Debug("calculator removed from group",
			zap.String("calculator", key.String()),
			zap.Int("group_id", groupID),
			zap.Int("remaining_members", remainingMembers),
		)
		if remainingMembers == 0 {
			g.L.Debug("group destroyed",
				zap.Int("group_id", groupID),
				zap.String("reason", "no remaining calculations"),
			)
			delete(g.groups, groupID)
		}
	}

	g.L.Debug("channel removed from graph",
		zap.String("channel", key.String()),
		zap.Int("num_dependencies", len(chInfo.calcDeps)),
	)

	delete(g.channels, key)

	// Cascade to dependencies
	for _, depKey := range chInfo.calcDeps {
		if depInfo := g.channels[depKey]; depInfo != nil {
			oldDepCount := depInfo.depCount
			depInfo.depCount--
			g.L.Debug("dependency count decremented",
				zap.String("channel", depKey.String()),
				zap.Uint32("dependency_count", uint32(oldDepCount)),
				zap.Uint32("new_dependency_count", uint32(depInfo.depCount)),
				zap.String("reason", fmt.Sprintf("depender %s removed", key)),
			)
			if depInfo.explicitCount == 0 && depInfo.depCount == 0 {
				g.L.Debug("cascading removal to dependency",
					zap.String("dependency", depKey.String()),
					zap.String("reason", "explicit and dependency counts reached zero"),
				)
				if err := g.removeChannel(depKey); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
