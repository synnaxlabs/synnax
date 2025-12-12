// Copyright 2025 Synnax Labs, Inc.
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
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
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
	alamos.Instrumentation
	Channel        *channel.Service
	SymbolResolver arc.SymbolResolver
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.SymbolResolver = override.Nil(c.SymbolResolver, other.SymbolResolver)
	return c
}

func (c Config) Validate() error {
	v := validate.New("calculation.graph")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "symbol_resolver", c.SymbolResolver)
	return v.Error()
}

type channelInfo struct {
	module        compiler.Module
	groupID       int
	explicitCount int
	depCount      int
	calcDeps      []channel.Key
	processing    bool
}

type groupInfo struct {
	baseDeps set.Set[channel.Key]
	members  set.Set[channel.Key]
}

type Graph struct {
	alamos.Instrumentation
	cfg         Config
	channels    map[channel.Key]*channelInfo
	groups      map[int]*groupInfo
	nextGroupID int
}

// New creates a new Graph with the provided configuration.
func New(cfgs ...Config) (*Graph, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
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
// Increments the explicit reference count for the channel.
func (g *Graph) Add(ctx context.Context, ch channel.Channel) error {
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
	return g.addInternal(ctx, ch, true)
}

// Update recompiles a channel with a new expression and updates its dependencies and group assignment.
// Preserves reference counts - use this when a channel's calculation changes but the same users are still requesting it.
func (g *Graph) Update(ctx context.Context, ch channel.Channel) error {
	info, err := g.getChannelInfo(ch.Key())
	if err != nil {
		return err
	}

	// Store current state
	oldExplicitCount := info.explicitCount
	oldDepCount := info.depCount
	oldCalcDeps := info.calcDeps
	oldGroupID := info.groupID

	// Mark as processing to detect circular dependencies during recompilation
	info.processing = true
	defer func() { info.processing = false }()

	// Recompile with new expression
	mod, err := compiler.Compile(ctx, compiler.Config{
		ChannelService: g.cfg.Channel,
		Channel:        ch,
		SymbolResolver: g.cfg.SymbolResolver,
	})
	if err != nil {
		return errors.Wrap(err, "failed to compile channel")
	}
	dependencies := mod.StateConfig.Reads.Keys()

	// Process new dependencies
	var newCalcDeps []channel.Key
	var depChannels []channel.Channel

	if len(dependencies) > 0 {
		depChannels, err = g.fetchChannels(ctx, dependencies)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve dependency channels")
		}

		for _, depCh := range depChannels {
			if depCh.IsCalculated() {
				// Check for circular dependencies before adding
				if err := g.checkCircularDependency(ch.Key(), depCh.Key()); err != nil {
					return err
				}
				// Add dependency if not already in graph
				if err := g.addInternal(ctx, depCh, false); err != nil {
					return errors.Wrapf(err, "failed to add calculated dependency %v", depCh.Key())
				}
				newCalcDeps = append(newCalcDeps, depCh.Key())
			}
		}
	}

	// Update depCounts: increment for new dependencies, decrement for removed ones
	oldDepsSet := make(set.Set[channel.Key])
	for _, key := range oldCalcDeps {
		oldDepsSet.Add(key)
	}
	newDepsSet := make(set.Set[channel.Key])
	for _, key := range newCalcDeps {
		newDepsSet.Add(key)
	}

	// Increment depCount for new dependencies
	for key := range newDepsSet {
		if !oldDepsSet.Contains(key) {
			if depInfo := g.channels[key]; depInfo != nil {
				oldDepCount := depInfo.depCount
				depInfo.depCount++
				g.L.Debug("dependency count incremented",
					zap.String("channel", key.String()),
					zap.Uint32("dependency_count", uint32(oldDepCount)),
					zap.Uint32("new_dependency_count", uint32(depInfo.depCount)),
					zap.String("depender", ch.Key().String()),
				)
			}
		}
	}

	// Decrement depCount for removed dependencies
	for key := range oldDepsSet {
		if !newDepsSet.Contains(key) {
			if depInfo := g.channels[key]; depInfo != nil {
				oldDepCount := depInfo.depCount
				depInfo.depCount--
				g.L.Debug("dependency count decremented",
					zap.String("channel", key.String()),
					zap.Uint32("dependency_count", uint32(oldDepCount)),
					zap.Uint32("new_dependency_count", uint32(depInfo.depCount)),
					zap.String("former_depender", ch.Key().String()),
				)
				if depInfo.explicitCount == 0 && depInfo.depCount == 0 {
					g.L.Debug("channel eligible for removal",
						zap.String("channel", key.String()),
						zap.String("reason", "explicit and dependency counts reached zero"),
					)
					if err := g.removeChannel(key); err != nil {
						return err
					}
				}
			}
		}
	}

	// Resolve new base dependencies
	baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		return err
	}

	// Assign to new group (may be same or different)
	newGroupID := g.assignToGroup(baseDeps)

	// Update channel info with new data while preserving reference counts
	// Do this BEFORE recalculating group base deps so the new module is used
	info.module = mod
	info.groupID = newGroupID
	info.calcDeps = newCalcDeps
	info.explicitCount = oldExplicitCount
	info.depCount = oldDepCount

	// Remove from old group if it changed
	if oldGroupID != newGroupID {
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
			} else {
				// Recalculate base dependencies for the old group
				if err := g.recalculateGroupBaseDeps(ctx, oldGroupID); err != nil {
					return err
				}
			}
		}
		g.groups[newGroupID].members.Add(ch.Key())
	} else {
		g.L.Debug("channel updated in same group",
			zap.String("channel", ch.Key().String()),
			zap.Int("group_id", newGroupID),
			zap.String("dependency_tree", g.formatDependencyTree(ctx, ch.Key())),
		)
		// Even if group didn't change, recalculate base deps since dependencies changed
		if err := g.recalculateGroupBaseDeps(ctx, newGroupID); err != nil {
			return err
		}
	}

	return nil
}

// checkCircularDependency checks if adding a dependency would create a circular dependency.
func (g *Graph) checkCircularDependency(source, target channel.Key) error {
	if source == target {
		err := errors.Newf("circular dependency detected: channel %v depends on itself", source)
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
			chainStrs := make([]string, len(chain))
			for i, k := range chain {
				chainStrs[i] = k.String()
			}
			chainStrs = append(chainStrs, source.String())
			err := errors.Newf("circular dependency detected involving channel %v", source)
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

// recalculateGroupBaseDeps recalculates the base dependencies for a group based on its current members.
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
		deps := info.module.StateConfig.Reads.Keys()
		if len(deps) == 0 {
			continue
		}

		depChannels, err := g.fetchChannels(ctx, deps)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve dependency channels for base deps recalculation")
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
func (g *Graph) addInternal(ctx context.Context, ch channel.Channel, explicit bool) error {
	if info := g.channels[ch.Key()]; info != nil {
		if info.processing {
			return errors.Newf("circular dependency detected involving channel %v", ch.Key())
		}
		return nil
	}

	info := &channelInfo{processing: true}
	g.channels[ch.Key()] = info
	defer func() { info.processing = false }()

	mod, err := compiler.Compile(ctx, compiler.Config{
		ChannelService: g.cfg.Channel,
		Channel:        ch,
		SymbolResolver: g.cfg.SymbolResolver,
	})
	if err != nil {
		delete(g.channels, ch.Key())
		return errors.Wrap(err, "failed to compile channel")
	}
	dependencies := mod.StateConfig.Reads.Keys()

	var calcDeps []channel.Key
	var depChannels []channel.Channel

	if len(dependencies) > 0 {
		var err error
		depChannels, err = g.fetchChannels(ctx, dependencies)
		if err != nil {
			delete(g.channels, ch.Key())
			return errors.Wrap(err, "failed to retrieve dependency channels")
		}

		for _, depCh := range depChannels {
			if depCh.IsCalculated() {
				if err := g.addInternal(ctx, depCh, false); err != nil {
					delete(g.channels, ch.Key())
					return errors.Wrapf(err, "failed to add calculated dependency %v", depCh.Key())
				}
				depInfo := g.channels[depCh.Key()]
				oldDepCount := depInfo.depCount
				depInfo.depCount++
				g.L.Debug("dependency count incremented",
					zap.String("channel", depCh.Key().String()),
					zap.Uint32("dependency_count", uint32(oldDepCount)),
					zap.Uint32("new_dependency_count", uint32(depInfo.depCount)),
					zap.String("depender", ch.Key().String()),
				)
				calcDeps = append(calcDeps, depCh.Key())
			}
		}
	}

	baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		delete(g.channels, ch.Key())
		return err
	}

	groupID := g.assignToGroup(baseDeps)

	info.module = mod
	info.groupID = groupID
	info.calcDeps = calcDeps
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

// fetchChannels retrieves channels by their keys.
func (g *Graph) fetchChannels(ctx context.Context, keys []channel.Key) ([]channel.Channel, error) {
	var channels []channel.Channel
	if err := g.cfg.Channel.NewRetrieve().
		Entries(&channels).
		WhereKeys(keys...).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return channels, nil
}

// resolveBaseDependencies recursively resolves all dependencies to find the concrete
// (non-calculated) base channels that this channel ultimately depends on.
// depChannels should be the already-fetched channels for mod.StateConfig.Reads.
func (g *Graph) resolveBaseDependencies(
	ctx context.Context,
	depChannels []channel.Channel,
) (set.Set[channel.Key], error) {
	baseDeps := make(set.Set[channel.Key])

	if len(depChannels) == 0 {
		return baseDeps, nil
	}

	for _, depCh := range depChannels {
		if depCh.IsCalculated() {
			info, err := g.getChannelInfo(depCh.Key())
			if err != nil {
				return nil, err
			}
			// Recursively resolve - fetch the dependency's dependencies
			depDeps := info.module.StateConfig.Reads.Keys()
			var depDepChannels []channel.Channel
			if len(depDeps) > 0 {
				depDepChannels, err = g.fetchChannels(ctx, depDeps)
				if err != nil {
					return nil, errors.Wrap(err, "failed to retrieve recursive dependency channels")
				}
			}
			recursiveDeps, err := g.resolveBaseDependencies(ctx, depDepChannels)
			if err != nil {
				return nil, err
			}
			for dep := range recursiveDeps {
				baseDeps.Add(dep)
			}
		} else {
			// Check if this channel is an index of a calculated channel
			isIndexOfCalculated := false
			for _, info := range g.channels {
				if info.module.Channel.Index() == depCh.Key() {
					isIndexOfCalculated = true
					break
				}
			}
			// Only add if it's not an index of a calculated channel
			if !isIndexOfCalculated {
				baseDeps.Add(depCh.Key())
			}
		}
	}

	return baseDeps, nil
}

// assignToGroup finds the best-fit existing group or creates a new one.
// This implements the stable grouping strategy.
func (g *Graph) assignToGroup(baseDeps set.Set[channel.Key]) int {
	var (
		bestGroup        int
		bestGroupFound   bool
		smallestDiff     = -1
		shouldExtendBest bool
	)

	// Format base deps for logging
	baseDepStrs := make([]string, 0, len(baseDeps))
	for key := range baseDeps {
		baseDepStrs = append(baseDepStrs, key.String())
	}

	for groupID, group := range g.groups {
		if group.baseDeps.Equals(baseDeps) {
			g.L.Debug("channel assigned to existing group with exact base dependency match",
				zap.Int("group_id", groupID),
				zap.String("base_dependencies", strings.Join(baseDepStrs, ", ")),
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
		if shouldExtendBest {
			g.L.Debug("extending existing group with new base dependencies",
				zap.Int("group_id", bestGroup),
				zap.String("new_base_dependencies", strings.Join(baseDepStrs, ", ")),
			)
			g.groups[bestGroup].baseDeps = baseDeps.Copy()
		} else {
			g.L.Debug("channel assigned to existing group as subset",
				zap.Int("group_id", bestGroup),
				zap.String("base_dependencies", strings.Join(baseDepStrs, ", ")),
			)
		}
		return bestGroup
	}

	newGroupID := g.nextGroupID
	g.nextGroupID++
	g.groups[newGroupID] = &groupInfo{
		baseDeps: baseDeps.Copy(),
		members:  make(set.Set[channel.Key]),
	}
	g.L.Debug("new group created",
		zap.Int("group_id", newGroupID),
		zap.String("base_dependencies", strings.Join(baseDepStrs, ", ")),
		zap.String("reason", "no suitable existing group found"),
	)
	return newGroupID
}

// CalculateGrouped returns all modules grouped by their assigned group keys.
// Modules within each group are sorted in topological order (dependencies first).
func (g *Graph) CalculateGrouped() map[int][]compiler.Module {
	result := make(map[int][]compiler.Module, len(g.groups))

	for groupID, group := range g.groups {
		modules := make([]compiler.Module, 0, len(group.members))
		for key := range group.members {
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

// CalculateFlat returns all modules in a flat list sorted in topological order (dependencies first).
// Unlike CalculateGrouped, this does not group modules and returns a single sorted list.
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
		for depKey := range info.module.StateConfig.Reads {
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

// CalculatedKeys returns the set of all calculated channel keys managed by this allocator.
func (g *Graph) CalculatedKeys() set.Set[channel.Key] {
	keys := make(set.Set[channel.Key])
	for key := range g.channels {
		keys.Add(key)
	}
	return keys
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

func (g *Graph) topologicalSortGroup(groupKey int, modules []compiler.Module) ([]compiler.Module, error) {
	channelsInGroup := make(set.Set[channel.Key])
	for _, mod := range modules {
		channelsInGroup.Add(mod.Channel.Key())
	}

	graph := make(map[channel.Key][]channel.Key)
	for _, mod := range modules {
		var deps []channel.Key
		for depKey := range mod.StateConfig.Reads {
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
		return nil, errors.Wrapf(query.NotFound, "channel %v not found in allocator", key)
	}
	return info, nil
}

// formatDependencyTree builds a string representation of the full dependency tree for a channel.
// Returns a string like: "ch1 → [calculated: ch2, ch3] → [base: ch4, ch5, ch6]"
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
		parts = append(parts, fmt.Sprintf("[calculated: %s]", strings.Join(calcDepStrs, ", ")))
	}

	// Resolve and add base dependencies
	deps := info.module.StateConfig.Reads.Keys()
	if len(deps) > 0 {
		depChannels, err := g.fetchChannels(ctx, deps)
		if err == nil {
			baseDeps, err := g.resolveBaseDependencies(ctx, depChannels)
			if err == nil && len(baseDeps) > 0 {
				baseDepStrings := make([]string, 0, len(baseDeps))
				for dep := range baseDeps {
					baseDepStrings = append(baseDepStrings, dep.String())
				}
				parts = append(parts, fmt.Sprintf("[base: %s]", strings.Join(baseDepStrings, ", ")))
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

// Remove decrements the explicit reference count for a channel.
// When both explicit and dependency counts reach 0, removes the channel and cascades to dependencies.
func (g *Graph) Remove(key channel.Key) error {
	info, ok := g.channels[key]
	if !ok {
		g.L.Debug("channel removal requested but not found in graph",
			zap.String("channel", key.String()),
		)
		return nil
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
		g.L.Debug("channel retained in graph",
			zap.String("channel", key.String()),
			zap.String("reason", fmt.Sprintf("explicit_count=%d or dependency_count=%d is non-zero", info.explicitCount, info.depCount)),
		)
		return nil
	}
	g.L.Debug("channel eligible for removal",
		zap.String("channel", key.String()),
		zap.String("reason", "explicit and dependency counts reached zero"),
	)
	return g.removeChannel(key)
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
