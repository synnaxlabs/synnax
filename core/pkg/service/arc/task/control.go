// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arccontrol "github.com/synnaxlabs/synnax/pkg/service/arc/control"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"go.uber.org/zap"
)

// controlConflict is a write channel currently controlled by a subject other than this
// task.
type controlConflict struct {
	channel channel.Channel
	holder  control.Subject
}

// controlDigestKeys resolves the control digest channel for every node that leases one of
// the write channels.
func controlDigestKeys(
	ctx context.Context,
	channels *channel.Service,
	writeKeys channel.Keys,
) (channel.Keys, error) {
	leaseholders := writeKeys.UniqueLeaseholders()
	names := make([]string, len(leaseholders))
	for i, n := range leaseholders {
		names[i] = fmt.Sprintf("sy_node_%v_control", n)
	}
	var digests []channel.Channel
	if err := channels.NewRetrieve().
		Where(channel.MatchNames(names...)).
		Entries(&digests).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	keys := make(channel.Keys, len(digests))
	for i, ch := range digests {
		keys[i] = ch.Key()
	}
	return keys, nil
}

// retrieveWriteChannels resolves the channel records for the given keys, used to name
// channels in the control warning.
func retrieveWriteChannels(
	ctx context.Context,
	channels *channel.Service,
	keys channel.Keys,
) ([]channel.Channel, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var out []channel.Channel
	if err := channels.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		Entries(&out).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return out, nil
}

// declaredWriteKeys collects the channels the program writes to across all nodes and
// authority overrides. Index channels are excluded so only explicit writes are named.
func declaredWriteKeys(prog arc.Arc) channel.Keys {
	var keys channel.Keys
	for _, n := range prog.Program.Nodes {
		for k := range n.Channels.Write {
			keys = append(keys, channel.Key(k))
		}
	}
	for k := range prog.Program.Authorities.Channels {
		keys = append(keys, channel.Key(k))
	}
	return keys
}

// configureControlConflicts reports the write channels already held by another writer at
// configure time, read from a synchronous control snapshot local to this node.
func (f *factory) configureControlConflicts(
	ctx context.Context,
	t task.Task,
	prog arc.Arc,
) ([]controlConflict, error) {
	writeKeys := declaredWriteKeys(prog).Unique()
	if len(writeKeys) == 0 {
		return nil, nil
	}
	writeChannels, err := retrieveWriteChannels(ctx, f.cfg.Channel, writeKeys)
	if err != nil {
		return nil, err
	}
	states := arccontrol.New()
	for _, s := range f.cfg.Framer.ControlStates(ctx).ToStorage().SeriesSlice() {
		if err := states.ApplySeries(s); err != nil {
			f.cfg.L.Warn("failed to decode control digest at configure", zap.Error(err))
		}
	}
	self := control.Subject{Name: prog.Name, Key: t.Key.String()}
	return evaluateControlConflicts(states, writeChannels, self), nil
}

// evaluateControlConflicts returns the write channels held by a subject other than self,
// sorted by channel key for stable output.
func evaluateControlConflicts(
	states *arccontrol.States,
	writes []channel.Channel,
	self control.Subject,
) []controlConflict {
	var conflicts []controlConflict
	for _, ch := range writes {
		holder, ok := states.Holder(ch.Key())
		if !ok || holder.Subject == self {
			continue
		}
		conflicts = append(conflicts, controlConflict{channel: ch, holder: holder.Subject})
	}
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].channel.Key() < conflicts[j].channel.Key()
	})
	return conflicts
}

// controlHolderName returns the holder's name, falling back to its key.
func controlHolderName(subject control.Subject) string {
	if subject.Name != "" {
		return subject.Name
	}
	return subject.Key
}

// controlWarning builds the summary message and holder-grouped description for a conflict
// set. conflicts must be sorted by channel key so the grouping is deterministic.
func controlWarning(conflicts []controlConflict) (message, description string) {
	channelNoun := "channel"
	if len(conflicts) != 1 {
		channelNoun = "channels"
	}
	var holders []control.Subject
	channelsByHolder := make(map[string][]string)
	for _, c := range conflicts {
		if _, seen := channelsByHolder[c.holder.Key]; !seen {
			holders = append(holders, c.holder)
		}
		channelsByHolder[c.holder.Key] = append(channelsByHolder[c.holder.Key], c.channel.Name)
	}
	writerPhrase := "another writer"
	if len(holders) > 1 {
		writerPhrase = "other writers"
	}
	message = fmt.Sprintf(
		"Authority held on %d %s by %s",
		len(conflicts), channelNoun, writerPhrase,
	)
	lines := make([]string, len(holders))
	for i, h := range holders {
		lines[i] = fmt.Sprintf("%s: %s", controlHolderName(h), strings.Join(channelsByHolder[h.Key], ", "))
	}
	return message, strings.Join(lines, "\n")
}

// controlWarner folds the control digest and emits task conflict warnings off the streamer
// callback, so status writes cannot backpressure the relay that feeds the digest.
type controlWarner struct {
	// task is the Arc task whose status is updated when the conflict set changes.
	task *impl
	// states is the control-state fold this warner reads conflicts from.
	states *arccontrol.States
	// writes is the set of channels the program writes, named in the warning.
	writes []channel.Channel
	// self identifies this task as a control subject, excluded from its own conflicts.
	self control.Subject
	// notify wakes run after a digest frame is folded. Buffered so observe never blocks.
	notify chan struct{}
	// done signals run to exit before the task context is cancelled.
	done chan struct{}
	// stopOnce guards done against a double close.
	stopOnce sync.Once
	// decodeErrLogged makes a malformed digest log once. Only touched by observe, which
	// runs on the single sink goroutine, so no synchronization is needed.
	decodeErrLogged bool
}

func newControlWarner(
	t *impl,
	states *arccontrol.States,
	writes []channel.Channel,
	self control.Subject,
) *controlWarner {
	return &controlWarner{
		task:   t,
		states: states,
		writes: writes,
		self:   self,
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
}

// sink returns the digest streamer sink. It only folds the digest and wakes run, never
// writing status inline.
func (w *controlWarner) sink() *confluence.UnarySink[framer.StreamerResponse] {
	return &confluence.UnarySink[framer.StreamerResponse]{
		Sink: func(_ context.Context, res framer.StreamerResponse) error {
			w.observe(res)
			return nil
		},
	}
}

// observe folds a digest frame and wakes run. It performs no status I/O and never blocks.
func (w *controlWarner) observe(res framer.StreamerResponse) {
	for _, s := range res.Frame.ToStorage().SeriesSlice() {
		if err := w.states.ApplySeries(s); err != nil && !w.decodeErrLogged {
			w.decodeErrLogged = true
			w.task.factoryCfg.L.Error("failed to fold control digest", zap.Error(err))
		}
	}
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// run re-evaluates conflicts on each digest wake-up and hands them to the status composer.
// It runs in its own goroutine so a slow status write never stalls the digest streamer.
func (w *controlWarner) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-w.notify:
			conflicts := evaluateControlConflicts(w.states, w.writes, w.self)
			w.task.setConflicts(ctx, conflicts)
		}
	}
}

// stop signals run to exit before the task context is cancelled. It is safe to call more
// than once.
func (w *controlWarner) stop() { w.stopOnce.Do(func() { close(w.done) }) }
