// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	// ControlState is the state of control over one channel.
	ControlState = control.State[channel.Key]
	// ControlUpdate is a batch of control transfers.
	ControlUpdate = control.Update[channel.Key]
	transfer      = control.Transfer[channel.Key]
	gate          = control.Gate[channel.Key, *region]
)

// region is the resource of a control region. The framer gates writes on it, and Cesium
// holds the samples.
type region struct{ key channel.Key }

// Key implements control.Resource.
func (r *region) Key() channel.Key { return r.key }

// controls holds the controller of each channel that a session writes, and notifies
// handlers of every control transfer.
type controls struct {
	updates observe.Observer[ControlUpdate]
	byKey   map[channel.Key]*controlled
	mu      sync.Mutex
}

// controlled is the controller of one channel, with the number of gates on it.
type controlled struct {
	c *control.Controller[channel.Key, *region]
	// index is the index of the channel, or zero.
	index channel.Key
	gates int
}

func newControls() *controls {
	return &controls{
		updates: observe.New[ControlUpdate](),
		byKey:   make(map[channel.Key]*controlled),
	}
}

// acquire returns the controller of ch, creating it with the concurrency of ch.
func (cs *controls) acquire(ch cesium.Channel) (*controlled, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	key := channel.Key(ch.Key)
	if c, ok := cs.byKey[key]; ok {
		c.gates++
		return c, nil
	}
	concurrency := ch.Concurrency
	if ch.Virtual {
		concurrency = control.ConcurrencyShared
	}
	c, err := control.New[channel.Key, *region](
		control.Config{Concurrency: concurrency},
	)
	if err != nil {
		return nil, err
	}
	ctl := &controlled{c: c, index: channel.Key(ch.Index), gates: 1}
	cs.byKey[key] = ctl
	return ctl, nil
}

// release drops one gate from the controller of key, and the controller with its last
// gate.
func (cs *controls) release(key channel.Key) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if c, ok := cs.byKey[key]; ok {
		if c.gates--; c.gates == 0 {
			delete(cs.byKey, key)
		}
	}
}

// states returns the state of control over each channel that a gate controls.
func (cs *controls) states() []ControlState {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	states := make([]ControlState, 0, len(cs.byKey))
	for _, c := range cs.byKey {
		if s := c.c.LeadingState(); s != nil {
			states = append(states, *s)
		}
	}
	return states
}

// holderAt returns the subject that controls the time range tr of key.
func (cs *controls) holderAt(
	key channel.Key,
	tr telem.TimeRange,
) (control.Subject, bool) {
	cs.mu.Lock()
	c, ok := cs.byKey[key]
	cs.mu.Unlock()
	if !ok {
		return control.Subject{}, false
	}
	return c.c.HolderAt(tr)
}

// heldIndexed returns the subject and key of a channel indexed by index that a gate
// controls and that own does not hold.
func (cs *controls) heldIndexed(
	index channel.Key,
	own set.Set[channel.Key],
) (control.Subject, channel.Key, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for key, c := range cs.byKey {
		if c.index != index || key == index || own.Contains(key) {
			continue
		}
		if s := c.c.LeadingState(); s != nil {
			return s.Subject, key, true
		}
	}
	return control.Subject{}, 0, false
}

// notify notifies handlers of the transfers that occurred.
func (cs *controls) notify(transfers []transfer) {
	transfers = slices.DeleteFunc(
		transfers,
		func(t transfer) bool { return !t.Occurred() },
	)
	if len(transfers) > 0 {
		cs.updates.Notify(context.Background(), ControlUpdate{Transfers: transfers})
	}
}

// gates is the hold of one session on the control of its channels. A persist only
// session holds no gates: it commands nothing.
type gates struct {
	controls *controls
	byKey    map[channel.Key]*gate
	groups   map[channel.Key]*group
	own      set.Set[channel.Key]
	// written holds the channels of the Cesium writer of the session.
	written []cesium.ChannelKey
}

// group is the index of the session and the data channels it indexes.
type group struct {
	// dataAuth is the last authority of each data channel. The index takes the highest
	// unless a SetAuthority names it.
	dataAuth map[channel.Key]control.Authority
	// written is true when the session writes the index.
	written bool
}

// openGates opens a gate on each channel of cfg that the session writes, indexes and
// virtual channels first, and returns the transfers of the opens. On an error it
// releases the gates it opened.
func (cs *controls) openGates(
	cfg Config,
	channels []cesium.Channel,
) (g *gates, transfers []transfer, err error) {
	g = &gates{
		controls: cs,
		byKey:    make(map[channel.Key]*gate, len(channels)),
		groups:   make(map[channel.Key]*group),
		own:      set.New[channel.Key](),
	}
	defer func() {
		if err != nil {
			g.release()
			g, transfers = nil, nil
		}
	}()
	byKey := make(map[channel.Key]cesium.Channel, len(channels))
	for _, ch := range channels {
		byKey[channel.Key(ch.Key)] = ch
	}
	if cfg.Start.IsZero() {
		cfg.Start = telem.Now()
	}
	persistOnly := cfg.Mode == ModePersistOnly
	for i, key := range cfg.Keys {
		ch, ok := byKey[key]
		if !ok {
			return g, transfers, errors.Newf("no channel for key %v of the writer", key)
		}
		if (!ch.Virtual && !ch.IsIndex) || (ch.Virtual && persistOnly) {
			continue
		}
		if transfers, err = g.open(cfg, i, ch, transfers); err != nil {
			return g, transfers, err
		}
		if ch.IsIndex {
			g.group(key).written = true
		}
	}
	for i, key := range cfg.Keys {
		ch := byKey[key]
		if ch.Virtual || ch.IsIndex || ch.Index == 0 {
			continue
		}
		if transfers, err = g.open(cfg, i, ch, transfers); err != nil {
			return g, transfers, err
		}
		if grp := g.group(channel.Key(ch.Index)); grp.written {
			grp.dataAuth[key] = cfg.Authorities[i]
		}
	}
	return g, transfers, nil
}

// open opens the gate of the session on ch, the channel at position i of cfg, and
// appends its transfer to transfers. A persist only session opens no gate.
func (g *gates) open(
	cfg Config,
	i int,
	ch cesium.Channel,
	transfers []transfer,
) ([]transfer, error) {
	g.written = append(g.written, ch.Key)
	if cfg.Mode == ModePersistOnly {
		return transfers, nil
	}
	ctl, err := g.controls.acquire(ch)
	if err != nil {
		return transfers, err
	}
	key := channel.Key(ch.Key)
	gt, t, err := ctl.c.OpenGate(control.GateConfig[channel.Key, *region]{
		TimeRange:             cfg.Start.Range(telem.TimeStampMax),
		Subject:               cfg.ControlSubject,
		Authority:             cfg.Authorities[i],
		ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorized,
		OpenResource:          func() (*region, error) { return &region{key: key}, nil },
	})
	if err != nil {
		g.controls.release(key)
		return transfers, cesium.ChannelError{Key: ch.Key, Err: err}
	}
	g.byKey[key] = gt
	g.own.Add(key)
	return append(transfers, t), nil
}

func (g *gates) group(index channel.Key) *group {
	grp, ok := g.groups[index]
	if !ok {
		grp = &group{dataAuth: make(map[channel.Key]control.Authority)}
		g.groups[index] = grp
	}
	return grp
}

// refused returns the channels of the session that it does not control.
func (g *gates) refused() set.Set[cesium.ChannelKey] {
	var refused set.Set[cesium.ChannelKey]
	for key, gt := range g.byKey {
		if g.authorize(gt) != nil {
			if refused == nil {
				refused = set.New[cesium.ChannelKey]()
			}
			refused.Add(uint32(key))
		}
	}
	return refused
}

func (g *gates) authorize(gt *gate) error {
	return gt.Authorize(func(*region) error { return nil })
}

// setAuthority sets the authority of the session over its channels. Naming a data
// channel also moves its written index to the highest authority across the index's data
// channels, unless cfg names the index too. It returns an error when the authority
// count is neither 1 nor the channel count.
func (g *gates) setAuthority(cfg Config) ([]transfer, error) {
	if len(cfg.Authorities) == 0 {
		return nil, nil
	}
	if len(cfg.Authorities) != 1 && len(cfg.Authorities) != len(cfg.Keys) {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"authority count must be 1 or equal to channel count",
		)
	}
	auths := g.authorities(cfg)
	transfers := make([]transfer, 0, len(auths))
	for key, gt := range g.byKey {
		if auth, ok := auths[key]; ok {
			transfers = append(transfers, gt.SetAuthority(auth))
		}
	}
	return transfers, nil
}

// authorities returns the authority that cfg sets on each channel, with the authority
// it
// moves each written index to.
func (g *gates) authorities(cfg Config) map[channel.Key]control.Authority {
	auths := make(map[channel.Key]control.Authority, len(g.byKey))
	if len(cfg.Keys) == 0 {
		for key := range g.byKey {
			auths[key] = cfg.Authorities[0]
		}
		for _, grp := range g.groups {
			for k := range grp.dataAuth {
				grp.dataAuth[k] = cfg.Authorities[0]
			}
		}
		return auths
	}
	explicit := set.New[channel.Key]()
	for i, key := range cfg.Keys {
		auth := cfg.Authorities[0]
		if len(cfg.Authorities) > 1 {
			auth = cfg.Authorities[i]
		}
		auths[key] = auth
		if _, isIndex := g.groups[key]; isIndex {
			explicit.Add(key)
			continue
		}
		for _, grp := range g.groups {
			if _, ok := grp.dataAuth[key]; ok {
				grp.dataAuth[key] = auth
			}
		}
	}
	for index, grp := range g.groups {
		if !grp.written || explicit.Contains(index) || len(grp.dataAuth) == 0 {
			continue
		}
		var highest control.Authority
		for _, a := range grp.dataAuth {
			highest = max(highest, a)
		}
		auths[index] = highest
	}
	return auths
}

// endDomain returns the indexes of indexes whose domains the session may end, the
// channels it must not commit, and the refusal of the other indexes. The session must
// control an index and the channels it indexes in the session, and no other session
// may control a channel the index indexes.
func (g *gates) endDomain(
	indexes channel.Keys,
) (ended []cesium.ChannelKey, skip set.Set[cesium.ChannelKey], refusal error) {
	refusedGroups := set.New[channel.Key]()
	for _, index := range indexes {
		grp, ok := g.groups[index]
		if !ok || !grp.written {
			continue
		}
		if err := g.authorizeGroup(index); err != nil {
			refusal = err
			refusedGroups.Add(index)
			continue
		}
		if subject, held, ok := g.controls.heldIndexed(index, g.own); ok {
			refusal = errors.Wrapf(
				control.ErrUnauthorized,
				"cannot end the domain of index %v because writer %v holds channel %v",
				uint32(index),
				subject,
				cesium.Channel{Key: uint32(held)},
			)
			refusedGroups.Add(index)
			continue
		}
		ended = append(ended, uint32(index))
	}
	if len(refusedGroups) == 0 {
		return ended, nil, refusal
	}
	// A refused index still commits the channels of its group that the session
	// controls, and nothing else.
	skip = set.New[cesium.ChannelKey]()
	for _, key := range g.written {
		k := channel.Key(key)
		if !refusedGroups.Contains(g.indexOf(k)) {
			skip.Add(key)
		} else if gt, ok := g.byKey[k]; ok && g.authorize(gt) != nil {
			skip.Add(key)
		}
	}
	return ended, skip, refusal
}

// authorizeGroup returns the refusal of the first channel of the group of index that
// the
// session does not control.
func (g *gates) authorizeGroup(index channel.Key) error {
	if err := g.authorizeKey(index); err != nil {
		return err
	}
	for key := range g.groups[index].dataAuth {
		if err := g.authorizeKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (g *gates) authorizeKey(key channel.Key) error {
	gt, ok := g.byKey[key]
	if !ok {
		return nil
	}
	if err := g.authorize(gt); err != nil {
		return cesium.ChannelError{Key: uint32(key), Err: err}
	}
	return nil
}

// indexOf returns the index of the group that holds key, or zero.
func (g *gates) indexOf(key channel.Key) channel.Key {
	if _, ok := g.groups[key]; ok {
		return key
	}
	for index, grp := range g.groups {
		if _, ok := grp.dataAuth[key]; ok {
			return index
		}
	}
	return 0
}

// release releases every gate of the session and returns the transfers.
func (g *gates) release() []transfer {
	transfers := make([]transfer, 0, len(g.byKey))
	for key, gt := range g.byKey {
		_, t := gt.Release()
		transfers = append(transfers, t)
		g.controls.release(key)
	}
	g.byKey = nil
	return transfers
}

// nameHolder names, in a claim refusal of Cesium, the writer that controls the claimed
// time range, and returns true. A refused open then wraps control.ErrUnauthorized. It
// returns err and false when err holds no claim refusal.
func (cs *controls) nameHolder(err error) (error, bool) {
	var (
		claimErr cesium.ClaimError
		chErr    cesium.ChannelError
	)
	if !errors.As(err, &claimErr) || !errors.As(err, &chErr) {
		return err, false
	}
	holder := "a writer"
	if subject, ok := cs.holderAt(channel.Key(chErr.Key), claimErr.TimeRange); ok {
		holder = "writer " + subject.String()
	}
	if claimErr.Open {
		return cesium.ChannelError{Key: chErr.Key, Err: errors.Wrapf(
			control.ErrUnauthorized,
			"cannot access time range %v because %s holds it",
			claimErr.TimeRange,
			holder,
		)}, true
	}
	return cesium.ChannelError{Key: chErr.Key, Err: errors.Wrapf(
		validate.ErrValidation,
		"cannot write to time range %v because %s holds it",
		claimErr.TimeRange,
		holder,
	)}, true
}
