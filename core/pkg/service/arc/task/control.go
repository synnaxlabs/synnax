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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// controlConflict is a write channel currently controlled by a different subject.
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
		states.ApplySeries(s)
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

// conflictKey is a stable identity for a conflict set, used to debounce repeated warnings.
func conflictKey(conflicts []controlConflict) string {
	parts := make([]string, len(conflicts))
	for i, c := range conflicts {
		parts[i] = fmt.Sprintf("%d:%s", c.channel.Key(), c.holder.Key)
	}
	return strings.Join(parts, ",")
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

// controlWarner mirrors control state and emits task warnings off the streamer callback,
// so status writes cannot backpressure the relay that feeds the digest.
type controlWarner struct {
	task     *impl
	states   *arccontrol.States
	writes   []channel.Channel
	self     control.Subject
	notify   chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	mu       sync.Mutex
	// lastKey identifies the last published conflict set, so repeats are not re-emitted.
	lastKey string
	// latest is the conflict set the writer goroutine should publish next.
	latest []controlConflict
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

// sink returns the digest streamer sink. It only mutates the in-memory mirror and flags a
// status update, never writing status inline.
func (w *controlWarner) sink() *confluence.UnarySink[framer.StreamerResponse] {
	return &confluence.UnarySink[framer.StreamerResponse]{
		Sink: func(_ context.Context, res framer.StreamerResponse) error {
			w.observe(res)
			return nil
		},
	}
}

// observe folds a digest frame into the mirror and signals the writer when the conflict
// set changes. It performs no I/O and never blocks on the status write.
func (w *controlWarner) observe(res framer.StreamerResponse) {
	for _, s := range res.Frame.ToStorage().SeriesSlice() {
		w.states.ApplySeries(s)
	}
	conflicts := evaluateControlConflicts(w.states, w.writes, w.self)
	key := conflictKey(conflicts)
	w.mu.Lock()
	changed := key != w.lastKey
	if changed {
		w.lastKey = key
		w.latest = conflicts
	}
	w.mu.Unlock()
	if !changed {
		return
	}
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// run publishes conflict-set changes as task statuses until ctx is cancelled. It runs in
// its own goroutine so a slow status write never stalls the digest streamer.
func (w *controlWarner) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-w.notify:
			w.mu.Lock()
			conflicts := w.latest
			w.mu.Unlock()
			w.task.setControlStatus(ctx, conflicts)
		}
	}
}

// stop signals run to exit before the task context is cancelled. It is safe to call more
// than once.
func (w *controlWarner) stop() { w.stopOnce.Do(func() { close(w.done) }) }

// setControlStatus emits a warning naming the conflicting channels, or restores a healthy
// running status when there are none.
func (t *impl) setControlStatus(ctx context.Context, conflicts []controlConflict) {
	stat := task.Status{
		Key:     task.OntologyID(t.task.Key).String(),
		Variant: status.VariantSuccess,
		Message: "Task running",
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.task.Key, Running: true},
	}
	if len(conflicts) > 0 {
		stat.Variant = status.VariantWarning
		stat.Message, stat.Description = controlWarning(conflicts)
	}
	if err := status.NewWriter[task.StatusDetails](t.factoryCfg.Status, nil).Set(ctx, &stat); err != nil {
		t.factoryCfg.L.Error("failed to set control status for Arc task", zap.Error(err))
	}
}
