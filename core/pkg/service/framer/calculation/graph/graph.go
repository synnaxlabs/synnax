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
	if len(dependencies) > 0 {
		depChannels, err := a.fetchChannels(ctx, dependencies)
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

	baseDeps, err := a.resolveBaseDependencies(ctx, ch.Key(), mod)
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
func (a *Graph) resolveBaseDependencies(
	ctx context.Context,
	_ channel.Key,
	mod compiler.Module,
) (set.Set[channel.Key], error) {
	baseDeps := make(set.Set[channel.Key])
	dependencies := mod.StateConfig.Reads.Keys()

	if len(dependencies) == 0 {
		return baseDeps, nil
	}

	depChannels, err := a.fetchChannels(ctx, dependencies)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve dependency channels")
	}

	for _, depCh := range depChannels {
		if depCh.IsCalculated() && !depCh.IsLegacyCalculated() {
			info, err := a.getChannelInfo(depCh.Key())
			if err != nil {
				return nil, err
			}
			recursiveDeps, err := a.resolveBaseDependencies(ctx, depCh.Key(), info.module)
			if err != nil {
				return nil, err
			}
			for dep := range recursiveDeps {
				baseDeps.Add(dep)
			}
		} else {
			baseDeps.Add(depCh.Key())
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
	info, err := a.getChannelInfo(key)
	if err != nil {
		return err
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
