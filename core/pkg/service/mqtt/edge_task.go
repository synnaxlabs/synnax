// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"sync"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// commandBacklog is the count of inbound commands that can wait to be written.
const commandBacklog = 64

// edgeTag is one tag of an edge node, ready to publish samples and to take commands.
type edgeTag struct {
	name string
	// index is the index channel of channel. Zero for a virtual channel.
	index    channel.Key
	channel  channel.Key
	dataType telem.DataType
	// command is the tag as a target of inbound commands. Nil for a tag that rejects
	// writes.
	command *readTag
}

// newEdgeTag validates cfg against channels, which holds its channels.
func newEdgeTag(
	node sparkplug.NodeID,
	cfg EdgeTag,
	channels map[channel.Key]channel.Channel,
) (edgeTag, sparkplug.Tag, error) {
	t := edgeTag{name: cfg.Name, channel: cfg.Channel}
	if cfg.Name == "" {
		return t, sparkplug.Tag{}, errors.Wrap(validate.ErrValidation, "tag: required")
	}
	ch, ok := channels[cfg.Channel]
	if !ok {
		return t, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: channel %d does not exist", cfg.Name, cfg.Channel,
		)
	}
	t.index, t.dataType = ch.Index(), ch.DataType
	sparkplugType, ok := sparkplugTypes[cfg.SparkplugType]
	if !ok {
		return t, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: unknown Sparkplug B data type %s", cfg.Name, cfg.SparkplugType,
		)
	}
	declared := sparkplug.Tag{Name: cfg.Name, DataType: sparkplugType}
	// A channel and a type that never match fail now, not on the first sample.
	if _, err := sparkplug.NewEdge([]sparkplug.Tag{declared}).Data(
		[]sparkplug.Metric{{Name: cfg.Name, Value: zeroValue(ch.DataType)}},
		telem.Now(),
	); err != nil {
		return t, declared, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: channel %s of type %s cannot be sent as %s",
			cfg.Name, ch.Name, ch.DataType, sparkplugType,
		)
	}
	if cfg.CommandChannel == 0 {
		return t, declared, nil
	}
	command, err := newReadTag(SparkplugReadEntry{
		Group:    node.Group,
		EdgeNode: node.EdgeNode,
		Tag:      cfg.Name,
		Channel:  cfg.CommandChannel,
	}, channels)
	if err != nil {
		return t, declared, errors.Wrap(err, "command channel")
	}
	t.command = &command
	return t, declared, nil
}

// zeroValue returns the zero sample of dataType in the form that a metric takes.
func zeroValue(dataType telem.DataType) any {
	if dataType.IsVariable() {
		return ""
	}
	value, _ := sampleValue(dataType, telem.MakeSeries(dataType, 1), 0)
	return value
}

// edgeNode is one Sparkplug B edge node: the sink of an edge task. It publishes the
// samples of its channels as NDATA messages on a client of its own, and writes the
// commands it receives to the command channels of its tags.
type edgeNode struct {
	ins       alamos.Instrumentation
	cfg       clientConfig
	node      sparkplug.NodeID
	tags      map[channel.Key][]edgeTag
	byName    map[string]edgeTag
	framer    *framer.Service
	task      task.Task
	authority control.Authority
	// commands carries the commands that the message handler received to the serve
	// loop, which writes them.
	commands chan sparkplug.Command
	// rebirths wakes the serve loop to publish the birth again.
	rebirths chan struct{}
	cancel   context.CancelFunc
	stopped  chan struct{}
	mu       struct {
		sync.Mutex
		// edge holds the sequence numbers, so every sequenced publish holds the lock.
		edge   *sparkplug.Edge
		client paho.Client
		// latest holds the last value of each tag, for the next birth.
		latest map[string]sparkplug.Metric
		// reported is the state that Health returned last.
		reported error
		// commandErr is the error of the last command, or nil when it was written.
		commandErr error
		// changed fires when the connection state changes.
		changed chan struct{}
	}
}

var _ driver.Sink = (*edgeNode)(nil)

func (e *edgeNode) topic(t sparkplug.MessageType) string {
	return sparkplug.Topic{Type: t, Node: e.node}.String()
}

// Start implements driver.Sink. It returns after the first connection attempt.
func (e *edgeNode) Start(ctx context.Context) error {
	e.commands = make(chan sparkplug.Command, commandBacklog)
	e.rebirths = make(chan struct{}, 1)
	e.mu.latest = make(map[string]sparkplug.Metric)
	e.mu.changed = make(chan struct{}, 1)
	e.mu.reported, e.mu.commandErr = nil, nil
	e.stopped = make(chan struct{})
	var (
		settled    = make(chan struct{})
		settleOnce sync.Once
		settle     = func() { settleOnce.Do(func() { close(settled) }) }
	)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(e.ins))
	e.cancel = cancel
	sCtx.Go(func(ctx context.Context) error {
		defer close(e.stopped)
		defer settle()
		e.run(ctx, settle)
		return nil
	}, signal.RecoverWithErrOnPanic())
	select {
	case <-settled:
		return nil
	case <-ctx.Done():
		cancel()
		<-e.stopped
		return ctx.Err()
	}
}

// Stop implements driver.Sink.
func (e *edgeNode) Stop() error {
	e.cancel()
	<-e.stopped
	return nil
}

func (e *edgeNode) run(ctx context.Context, settle func()) {
	reconnect(ctx, e.ins, reconnectHooks{
		options: func(lost chan<- error) *paho.ClientOptions {
			will, err := e.mu.edge.Death()
			if err != nil {
				e.ins.L.DPanic("failed to encode the NDEATH message", zap.Error(err))
			}
			return newClientOptions(e.cfg).
				SetBinaryWill(e.topic(sparkplug.NDeath), will, 1, false).
				SetOrderMatters(true).
				SetDefaultPublishHandler(e.onMessage).
				SetConnectionLostHandler(reportLost(lost))
		},
		serve: func(ctx context.Context, client paho.Client, lost <-chan error) {
			defer e.mu.edge.EndSession()
			err := wait(ctx, client.Subscribe(e.topic(sparkplug.NCmd), 0, nil))
			if err == nil {
				err = e.publishBirth(ctx, client)
			}
			if err != nil {
				if ctx.Err() == nil {
					e.ins.L.Warn(
						"failed to start the edge node session",
						zap.Error(err),
					)
				}
				return
			}
			e.setClient(client)
			settle()
			e.serve(ctx, client, lost)
			e.setClient(nil)
			if ctx.Err() != nil {
				// The broker sends the last will only for a connection that drops.
				deathCtx, cancel := context.WithTimeout(
					context.Background(), stateTimeout,
				)
				e.publishDeath(deathCtx, client)
				cancel()
			}
		},
		failed: func() {
			e.notify()
			settle()
		},
	})
}

// serve writes commands and answers rebirth requests until ctx is cancelled or the
// connection is lost.
func (e *edgeNode) serve(ctx context.Context, client paho.Client, lost <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-lost:
			e.ins.L.Warn("lost the connection to the MQTT broker", zap.Error(err))
			return
		case <-e.rebirths:
			if err := e.publishBirth(ctx, client); err != nil && ctx.Err() == nil {
				e.ins.L.Warn("failed to publish the NBIRTH message", zap.Error(err))
			}
		case cmd := <-e.commands:
			err := e.writeCommand(ctx, cmd)
			if ctx.Err() != nil {
				return
			}
			e.mu.Lock()
			e.mu.commandErr = err
			e.notify()
			e.mu.Unlock()
		}
	}
}

func (e *edgeNode) setClient(client paho.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mu.client = client
	e.notify()
}

// health returns the state that the task reports. The caller holds e.mu.
func (e *edgeNode) health() error {
	if e.mu.client == nil {
		return errNotConnected
	}
	if e.mu.commandErr != nil {
		return errors.Wrap(driver.ErrTemporary, e.mu.commandErr.Error())
	}
	return nil
}

func (e *edgeNode) notify() {
	select {
	case e.mu.changed <- struct{}{}:
	default:
	}
}

// Health implements driver.Sink. It reports each loss and recovery of the connection,
// and each command that could not be written.
func (e *edgeNode) Health(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.mu.changed:
			e.mu.Lock()
			err := e.health()
			same := errors.Is(err, e.mu.reported) && errors.Is(e.mu.reported, err)
			e.mu.reported = err
			e.mu.Unlock()
			if !same {
				return err
			}
		}
	}
}

// publishBirth publishes the NBIRTH message. It holds the lock through the publish,
// so that no NDATA message can overtake the birth that starts its sequence.
func (e *edgeNode) publishBirth(ctx context.Context, client paho.Client) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	payload, err := e.mu.edge.Birth(e.mu.latest, telem.Now())
	if err != nil {
		return err
	}
	return wait(ctx, client.Publish(e.topic(sparkplug.NBirth), 0, false, payload))
}

func (e *edgeNode) publishDeath(ctx context.Context, client paho.Client) {
	payload, err := e.mu.edge.Death()
	if err == nil {
		err = wait(ctx, client.Publish(e.topic(sparkplug.NDeath), 1, false, payload))
	}
	if err != nil && ctx.Err() == nil {
		e.ins.L.Warn("failed to publish the NDEATH message", zap.Error(err))
	}
}

// onMessage takes an NCMD message. It must not block: the client delivers messages in
// order on one goroutine.
func (e *edgeNode) onMessage(_ paho.Client, m paho.Message) {
	if m.Topic() != e.topic(sparkplug.NCmd) {
		return
	}
	cmd, err := e.mu.edge.DecodeCommand(m.Payload())
	if err != nil {
		e.ins.L.Debug("dropped a Sparkplug B command", zap.Error(err))
		return
	}
	if cmd.Rebirth {
		select {
		case e.rebirths <- struct{}{}:
		default:
		}
	}
	if len(cmd.Metrics) == 0 {
		return
	}
	select {
	case e.commands <- cmd:
	default:
		e.ins.L.Warn("dropped a Sparkplug B command: the command backlog is full")
	}
}

// writeCommand writes the values of cmd to the command channels of their tags, with
// the arrival time as their timestamp. Each command opens its own writer, because a
// command names any subset of the tags, and a writer needs every channel of an index
// in each frame.
func (e *edgeNode) writeCommand(ctx context.Context, cmd sparkplug.Command) error {
	var (
		fr   framer.Frame
		keys channel.Keys
		now  = telem.Now()
	)
	for _, m := range cmd.Metrics {
		t, ok := e.byName[m.Name]
		if !ok || t.command == nil {
			e.ins.L.Warn(
				"rejected a command for a tag that takes no writes",
				zap.String("tag", m.Name),
			)
			continue
		}
		data, err := t.command.appendSample(nil, m.Value)
		if err != nil {
			e.ins.L.Warn(
				"rejected a command value", zap.String("tag", m.Name), zap.Error(err),
			)
			continue
		}
		if fr.Get(t.command.channel).Len() > 0 {
			continue
		}
		fr = fr.Append(t.command.channel, telem.Series{
			DataType: t.command.dataType, Data: data,
		})
		keys = append(keys, t.command.channel)
		if t.command.index != 0 && fr.Get(t.command.index).Len() == 0 {
			fr = fr.Append(t.command.index, telem.NewSeriesV(now))
			keys = append(keys, t.command.index)
		}
	}
	if fr.Empty() {
		return nil
	}
	authorities := make([]control.Authority, len(keys))
	for i := range authorities {
		authorities[i] = e.authority
	}
	w, err := e.framer.OpenWriter(ctx, framer.WriterConfig{
		ControlSubject:    control.Subject{Key: e.task.Key.String(), Name: e.task.Name},
		Keys:              keys,
		Authorities:       authorities,
		Start:             now,
		ErrOnUnauthorized: new(true),
		Sync:              new(true),
	})
	if err != nil {
		return err
	}
	_, err = w.Write(fr)
	return errors.Combine(err, w.Close())
}

// Write implements driver.Sink. It publishes the samples of fr in one NDATA message.
// A sample carries the timestamp of its index when the frame holds it.
func (e *edgeNode) Write(ctx context.Context, fr framer.Frame) error {
	var metrics []sparkplug.Metric
	for key, series := range fr.Entries() {
		for _, t := range e.tags[key] {
			stamps := indexOf(fr, t.index, series)
			for i := range int(series.Len()) {
				value, err := sampleValue(t.dataType, series, i)
				if err != nil {
					return errors.Wrapf(driver.ErrTemporary, "tag %s: %v", t.name, err)
				}
				m := sparkplug.Metric{Name: t.name, Value: value}
				if stamps != nil {
					m.Timestamp = stamps.ValueAt[telem.TimeStamp](i)
				}
				metrics = append(metrics, m)
			}
		}
	}
	if len(metrics) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, m := range metrics {
		e.mu.latest[m.Name] = m
	}
	if e.mu.client == nil {
		return errNotConnected
	}
	payload, err := e.mu.edge.Data(metrics, telem.Now())
	if err != nil {
		return errors.Wrap(driver.ErrTemporary, err.Error())
	}
	err = wait(ctx, e.mu.client.Publish(e.topic(sparkplug.NData), 0, false, payload))
	if err != nil {
		return errors.Wrap(driver.ErrTemporary, err.Error())
	}
	return nil
}

// indexOf returns the series of index in fr that lines up with series, or nil.
func indexOf(fr framer.Frame, index channel.Key, series telem.Series) *telem.Series {
	if index == 0 {
		return nil
	}
	for key, s := range fr.Entries() {
		if key == index && s.Alignment == series.Alignment && s.Len() == series.Len() {
			return &s
		}
	}
	return nil
}
