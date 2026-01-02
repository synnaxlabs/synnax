// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package store exposes a simple copy-on-read Store for managing cluster state.
// SinkTarget create a new Store, call store.New().
package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/store"
)

type Change struct {
	State   State
	Changes []node.Change
}

// Store is an interface representing a copy-on-read Store for managing cluster state.
type Store interface {
	// Observable allows the caller to react to state changes. This state is not diffed i.e.
	// any call that modifies the state, even if no actual change occurs, will get sent to the
	// Observable.
	store.Observable[State, Change]
	// ClusterKey returns the cluster key.
	ClusterKey() uuid.UUID
	// SetClusterKey sets the cluster key.
	SetClusterKey(ctx context.Context, key uuid.UUID)
	// SetNode sets a node in state.
	SetNode(context.Context, node.Node)
	// GetNode returns a node from state. Returns false if the node is not found.
	GetNode(key node.Key) (node.Node, bool)
	// Merge merges a node.Group into State.Nodes by selecting nodes from group with heartbeats
	// that are either not in State or are older than in State.
	Merge(ctx context.Context, group node.Group)
	// GetHost returns the host node of the Store.
	GetHost() node.Node
	// SetHost sets the host for the Store.
	SetHost(ctx context.Context, node node.Node)
}

func _copy(s State) State {
	return State{Nodes: s.Nodes.Copy(), HostKey: s.HostKey, ClusterKey: s.ClusterKey}
}

// transform decides whether we should notify observers
// of the cluster state change. We only notify if:
//
//  1. The cluster key has been set.
//  2. The host node has been set.
//  3. A node has been added or removed from the cluster.
//  4. The state of a node has changed.
//
// We DO NOT notify on heartbeat increments.
func transform(
	prevState,
	nextState State,
) (Change, bool) {
	// This means aspen hasn't been initialized yet.
	if prevState.ClusterKey != nextState.ClusterKey || nextState.HostKey == 0 {
		return Change{}, false
	}
	changes := change.Map(
		prevState.Nodes,
		nextState.Nodes,
		node.BasicallyEqual,
	)
	return Change{State: nextState, Changes: changes}, len(changes) > 0
}

// New opens a new empty, invalid Store.
func New(ctx context.Context) Store {
	c := &core{Observable: lo.Must(store.WrapObservable(store.ObservableConfig[State, Change]{
		Store:     store.New(_copy),
		Transform: transform,
	}))}
	c.SetState(ctx, State{Nodes: make(node.Group)})
	return c
}

// State is the current state of the cluster as viewed from the host.
type State struct {
	ClusterKey uuid.UUID
	HostKey    node.Key
	Nodes      node.Group
}

func (s *State) IsZero() bool {
	return s.ClusterKey == uuid.Nil && s.HostKey == 0 && len(s.Nodes) == 0
}

type core struct {
	store.Observable[State, Change]
}

// ClusterKey implements Store.
func (c *core) ClusterKey() uuid.UUID {
	s, release := c.PeekState()
	ck := s.ClusterKey
	release()
	return ck
}

// SetClusterKey implements Store.
func (c *core) SetClusterKey(ctx context.Context, key uuid.UUID) {
	s := c.CopyState()
	s.ClusterKey = key
	c.SetState(ctx, s)
}

// GetNode implements Store.
func (c *core) GetNode(key node.Key) (node.Node, bool) {
	state, release := c.PeekState()
	defer release()
	n, ok := state.Nodes[key]
	return n, ok
}

// GetHost implements Store.
func (c *core) GetHost() node.Node {
	state, release := c.PeekState()
	h := state.HostKey
	release()
	n, _ := c.GetNode(h)
	return n
}

// SetHost implements Store.
func (c *core) SetHost(ctx context.Context, n node.Node) {
	snap := c.CopyState()
	snap.Nodes[n.Key] = n
	snap.HostKey = n.Key
	c.SetState(ctx, snap)
}

// SetNode implements Store.
func (c *core) SetNode(ctx context.Context, n node.Node) {
	snap := c.CopyState()
	snap.Nodes[n.Key] = n
	c.SetState(ctx, snap)
}

// Merge implements Store.
func (c *core) Merge(ctx context.Context, other node.Group) {
	snap := c.CopyState()
	for _, n := range other {
		in, ok := snap.Nodes[n.Key]
		if !ok || n.Heartbeat.OlderThan(in.Heartbeat) {
			snap.Nodes[n.Key] = n
		}
	}
	c.SetState(ctx, snap)
}
