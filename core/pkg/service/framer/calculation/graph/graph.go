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

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

type Config struct {
	Channel        channel.Readable
	SymbolResolver arc.SymbolResolver
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
	cfg         Config
	channels    map[channel.Key]*channelInfo
	groups      map[int]*groupInfo
	nextGroupID int
}

// New creates a new Graph with the provided configuration.
func New(cfg Config) *Graph {
	return &Graph{
		cfg:         cfg,
		channels:    make(map[channel.Key]*channelInfo),
		groups:      make(map[int]*groupInfo),
		nextGroupID: 0,
	}
}

// Add compiles a channel and its dependencies, then assigns it to a group.
// Increments the explicit reference count for the channel.
func (a *Graph) Add(ctx context.Context, ch channel.Channel) error {
	if info := a.channels[ch.Key()]; info != nil {
		info.explicitCount++
		return nil
	}
	return a.addInternal(ctx, ch, true)
}

// Update recompiles a channel with a new expression and updates its dependencies and group assignment.
// Preserves reference counts - use this when a channel's calculation changes but the same users are still requesting it.
func (a *Graph) Update(ctx context.Context, ch channel.Channel) error {
	info, err := a.getChannelInfo(ch.Key())
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
		Channels:       a.cfg.Channel,
		Channel:        ch,
		SymbolResolver: a.cfg.SymbolResolver,
	})
	if err != nil {
		return errors.Wrap(err, "failed to compile channel")
	}
	dependencies := mod.StateConfig.Reads.Keys()

	// Process new dependencies
	var newCalcDeps []channel.Key
	var depChannels []channel.Channel

	if len(dependencies) > 0 {
		depChannels, err = a.fetchChannels(ctx, dependencies)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve dependency channels")
		}

		for _, depCh := range depChannels {
			if depCh.IsCalculated() && !depCh.IsLegacyCalculated() {
				// Check for circular dependencies before adding
				if err := a.checkCircularDependency(ch.Key(), depCh.Key()); err != nil {
					return err
				}
				// Add dependency if not already in graph
				if err := a.addInternal(ctx, depCh, false); err != nil {
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
			if depInfo := a.channels[key]; depInfo != nil {
				depInfo.depCount++
			}
		}
	}

	// Decrement depCount for removed dependencies
	for key := range oldDepsSet {
		if !newDepsSet.Contains(key) {
			if depInfo := a.channels[key]; depInfo != nil {
				depInfo.depCount--
				if depInfo.explicitCount == 0 && depInfo.depCount == 0 {
					if err := a.removeChannel(key); err != nil {
						return err
					}
				}
			}
		}
	}

	// Resolve new base dependencies
	baseDeps, err := a.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		return err
	}

	// Assign to new group (may be same or different)
	newGroupID := a.assignToGroup(baseDeps)

	// Update channel info with new data while preserving reference counts
	// Do this BEFORE recalculating group base deps so the new module is used
	info.module = mod
	info.groupID = newGroupID
	info.calcDeps = newCalcDeps
	info.explicitCount = oldExplicitCount
	info.depCount = oldDepCount

	// Remove from old group if it changed
	if oldGroupID != newGroupID {
		if oldGroup := a.groups[oldGroupID]; oldGroup != nil {
			oldGroup.members.Remove(ch.Key())
			if len(oldGroup.members) == 0 {
				delete(a.groups, oldGroupID)
			} else {
				// Recalculate base dependencies for the old group
				a.recalculateGroupBaseDeps(ctx, oldGroupID)
			}
		}
		a.groups[newGroupID].members.Add(ch.Key())
	} else {
		// Even if group didn't change, recalculate base deps since dependencies changed
		a.recalculateGroupBaseDeps(ctx, newGroupID)
	}

	return nil
}

// checkCircularDependency checks if adding a dependency would create a circular dependency.
func (a *Graph) checkCircularDependency(source, target channel.Key) error {
	if source == target {
		return errors.Newf("circular dependency detected: channel %v depends on itself", source)
	}

	// Check if target (or any of its dependencies) depends on source
	visited := make(set.Set[channel.Key])
	var checkDeps func(channel.Key) error
	checkDeps = func(key channel.Key) error {
		if key == source {
			return errors.Newf("circular dependency detected involving channel %v", source)
		}
		if visited.Contains(key) {
			return nil
		}
		visited.Add(key)

		info := a.channels[key]
		if info == nil {
			return nil
		}

		for _, depKey := range info.calcDeps {
			if err := checkDeps(depKey); err != nil {
				return err
			}
		}
		return nil
	}

	return checkDeps(target)
}

// recalculateGroupBaseDeps recalculates the base dependencies for a group based on its current members.
func (a *Graph) recalculateGroupBaseDeps(ctx context.Context, groupID int) error {
	group := a.groups[groupID]
	if group == nil {
		return nil
	}

	newBaseDeps := make(set.Set[channel.Key])
	for memberKey := range group.members {
		info := a.channels[memberKey]
		if info == nil {
			continue
		}

		// Get all dependencies for this member
		deps := info.module.StateConfig.Reads.Keys()
		if len(deps) == 0 {
			continue
		}

		depChannels, err := a.fetchChannels(ctx, deps)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve dependency channels for base deps recalculation")
		}

		baseDeps, err := a.resolveBaseDependencies(ctx, depChannels)
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
func (a *Graph) addInternal(ctx context.Context, ch channel.Channel, explicit bool) error {
	if info := a.channels[ch.Key()]; info != nil {
		if info.processing {
			return errors.Newf("circular dependency detected involving channel %v", ch.Key())
		}
		return nil
	}

	info := &channelInfo{processing: true}
	a.channels[ch.Key()] = info
	defer func() { info.processing = false }()

	mod, err := compiler.Compile(ctx, compiler.Config{
		Channels:       a.cfg.Channel,
		Channel:        ch,
		SymbolResolver: a.cfg.SymbolResolver,
	})
	if err != nil {
		delete(a.channels, ch.Key())
		return errors.Wrap(err, "failed to compile channel")
	}
	dependencies := mod.StateConfig.Reads.Keys()

	var calcDeps []channel.Key
	var depChannels []channel.Channel

	if len(dependencies) > 0 {
		var err error
		depChannels, err = a.fetchChannels(ctx, dependencies)
		if err != nil {
			delete(a.channels, ch.Key())
			return errors.Wrap(err, "failed to retrieve dependency channels")
		}

		for _, depCh := range depChannels {
			if depCh.IsCalculated() && !depCh.IsLegacyCalculated() {
				if err := a.addInternal(ctx, depCh, false); err != nil {
					delete(a.channels, ch.Key())
					return errors.Wrapf(err, "failed to add calculated dependency %v", depCh.Key())
				}
				a.channels[depCh.Key()].depCount++
				calcDeps = append(calcDeps, depCh.Key())
			}
		}
	}

	baseDeps, err := a.resolveBaseDependencies(ctx, depChannels)
	if err != nil {
		delete(a.channels, ch.Key())
		return err
	}

	groupID := a.assignToGroup(baseDeps)

	info.module = mod
	info.groupID = groupID
	info.calcDeps = calcDeps
	if explicit {
		info.explicitCount = 1
	}

	a.groups[groupID].members.Add(ch.Key())

	return nil
}

// fetchChannels retrieves channels by their keys.
func (a *Graph) fetchChannels(ctx context.Context, keys []channel.Key) ([]channel.Channel, error) {
	var channels []channel.Channel
	if err := a.cfg.Channel.NewRetrieve().
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
func (a *Graph) resolveBaseDependencies(
	ctx context.Context,
	depChannels []channel.Channel,
) (set.Set[channel.Key], error) {
	baseDeps := make(set.Set[channel.Key])

	if len(depChannels) == 0 {
		return baseDeps, nil
	}

	for _, depCh := range depChannels {
		if depCh.IsCalculated() && !depCh.IsLegacyCalculated() {
			info, err := a.getChannelInfo(depCh.Key())
			if err != nil {
				return nil, err
			}
			// Recursively resolve - fetch the dependency's dependencies
			depDeps := info.module.StateConfig.Reads.Keys()
			var depDepChannels []channel.Channel
			if len(depDeps) > 0 {
				depDepChannels, err = a.fetchChannels(ctx, depDeps)
				if err != nil {
					return nil, errors.Wrap(err, "failed to retrieve recursive dependency channels")
				}
			}
			recursiveDeps, err := a.resolveBaseDependencies(ctx, depDepChannels)
			if err != nil {
				return nil, err
			}
			for dep := range recursiveDeps {
				baseDeps.Add(dep)
			}
		} else {
			// Check if this channel is an index of a calculated channel
			isIndexOfCalculated := false
			for _, info := range a.channels {
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
func (a *Graph) assignToGroup(baseDeps set.Set[channel.Key]) int {
	var (
		bestGroup        int
		bestGroupFound   bool
		smallestDiff     = -1
		shouldExtendBest bool
	)

	for groupID, group := range a.groups {
		if group.baseDeps.Equals(baseDeps) {
			return groupID
		}

		if baseDeps.Subset(group.baseDeps) {
			diff := len(group.baseDeps) - len(baseDeps)
			if !bestGroupFound || diff < smallestDiff {
				bestGroup = groupID
				bestGroupFound = true
				smallestDiff = diff
				shouldExtendBest = false
			}
		} else if group.baseDeps.Subset(baseDeps) {
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
			a.groups[bestGroup].baseDeps = baseDeps.Copy()
		}
		return bestGroup
	}

	newGroupID := a.nextGroupID
	a.nextGroupID++
	a.groups[newGroupID] = &groupInfo{
		baseDeps: baseDeps.Copy(),
		members:  make(set.Set[channel.Key]),
	}
	return newGroupID
}

// CalculateGrouped returns all modules grouped by their assigned group keys.
// Modules within each group are sorted in topological order (dependencies first).
func (a *Graph) CalculateGrouped() map[int][]compiler.Module {
	result := make(map[int][]compiler.Module, len(a.groups))

	for groupID, group := range a.groups {
		modules := make([]compiler.Module, 0, len(group.members))
		for key := range group.members {
			modules = append(modules, a.channels[key].module)
		}
		sorted, err := a.topologicalSortGroup(groupID, modules)
		if err != nil {
			continue
		}
		result[groupID] = sorted
	}

	return result
}

// CalculateFlat returns all modules in a flat list sorted in topological order (dependencies first).
// Unlike CalculateGrouped, this does not group modules and returns a single sorted list.
func (a *Graph) CalculateFlat() []compiler.Module {
	if len(a.channels) == 0 {
		return nil
	}

	allChannels := make(set.Set[channel.Key])
	for key := range a.channels {
		allChannels.Add(key)
	}

	graph := make(map[channel.Key][]channel.Key)
	for key, info := range a.channels {
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

	for key := range a.channels {
		if err := visit(key); err != nil {
			continue
		}
	}

	sortedModules := make([]compiler.Module, 0, len(result))
	for _, key := range result {
		sortedModules = append(sortedModules, a.channels[key].module)
	}

	return sortedModules
}

// CalculatedKeys returns the set of all calculated channel keys managed by this allocator.
func (a *Graph) CalculatedKeys() set.Set[channel.Key] {
	keys := make(set.Set[channel.Key])
	for key := range a.channels {
		keys.Add(key)
	}
	return keys
}

// ConcreteBaseKeys returns the set of all concrete (non-calculated) base channel keys
// that the calculated channels depend on.
func (a *Graph) ConcreteBaseKeys() set.Set[channel.Key] {
	allBaseDeps := make(set.Set[channel.Key])
	for _, group := range a.groups {
		for dep := range group.baseDeps {
			allBaseDeps.Add(dep)
		}
	}
	return allBaseDeps
}

func (a *Graph) topologicalSortGroup(groupKey int, modules []compiler.Module) ([]compiler.Module, error) {
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
			return errors.Newf("circular dependency detected in group %s", groupKey)
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
		sortedModules = append(sortedModules, a.channels[key].module)
	}
	return sortedModules, nil
}

func (a *Graph) getChannelInfo(key channel.Key) (*channelInfo, error) {
	info := a.channels[key]
	if info == nil {
		return nil, errors.Wrapf(query.NotFound, "channel %v not found in allocator", key)
	}
	return info, nil
}

// Remove decrements the explicit reference count for a channel.
// When both explicit and dependency counts reach 0, removes the channel and cascades to dependencies.
func (a *Graph) Remove(key channel.Key) error {
	info, ok := a.channels[key]
	if !ok {
		return nil
	}
	info.explicitCount--
	if info.explicitCount > 0 || info.depCount > 0 {
		return nil
	}
	return a.removeChannel(key)
}

// removeChannel removes a channel and cascades to its dependencies.
func (a *Graph) removeChannel(key channel.Key) error {
	chInfo, err := a.getChannelInfo(key)
	if err != nil {
		return err
	}

	if group := a.groups[chInfo.groupID]; group != nil {
		group.members.Remove(key)
		if len(group.members) == 0 {
			delete(a.groups, chInfo.groupID)
		}
	}

	delete(a.channels, key)
	for _, depKey := range chInfo.calcDeps {
		if depInfo := a.channels[depKey]; depInfo != nil {
			depInfo.depCount--
			if depInfo.explicitCount == 0 && depInfo.depCount == 0 {
				if err := a.removeChannel(depKey); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
