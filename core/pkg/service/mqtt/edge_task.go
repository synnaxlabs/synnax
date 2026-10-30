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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// commandBacklog is the count of inbound commands that can wait to be written.
const commandBacklog = 64

// publishedTag is one tag of an edge node, ready to publish samples and to take
// commands.
type publishedTag struct {
	name string
	// index is the index channel of channel. Zero for a virtual channel.
	index   channel.Key
	channel channel.Key
	// command is the tag as a target of inbound commands. Nil for a tag that rejects
	// writes.
	command *readTag
}

// newPublishedTag validates cfg against channels, which holds its channels.
func newPublishedTag(
	node sparkplug.NodeID,
	cfg EdgeTag,
	channels map[channel.Key]channel.Channel,
) (publishedTag, sparkplug.Tag, error) {
	t := publishedTag{name: cfg.Name, channel: cfg.Channel}
	if cfg.Name == "" {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrap(
			validate.ErrValidation,
			"tag: required",
		)
	}
	if sparkplug.Reserved(cfg.Name) {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation, "tag %s: Sparkplug B reserves the name", cfg.Name,
		)
	}
	ch, ok := channels[cfg.Channel]
	if !ok {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: channel %d does not exist", cfg.Name, cfg.Channel,
		)
	}
	t.index = ch.Index()
	sparkplugType, ok := sparkplugTypes[cfg.TagType]
	if !ok {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: unknown Sparkplug B data type %s", cfg.Name, cfg.TagType,
		)
	}
	declared := sparkplug.Tag{Name: cfg.Name, DataType: sparkplugType}
	if !sparkplug.Compatible(ch.DataType, sparkplugType) {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrapf(
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
		return publishedTag{}, sparkplug.Tag{}, errors.Wrap(err, "command channel")
	}
	if !sparkplug.Compatible(command.dataType, sparkplugType) {
		return publishedTag{}, sparkplug.Tag{}, errors.Wrapf(
			validate.ErrValidation,
			"tag %s: command channel %s of type %s cannot take values of %s",
			cfg.Name, channels[cfg.CommandChannel].Name, command.dataType,
			sparkplugType,
		)
	}
	t.command = &command
	return t, declared, nil
}

// edgeHealth is the state that the Health of an edge node reports.
type edgeHealth struct {
	// commandErr is the text of the error of the last command. Empty when the command
	// was written.
	commandErr string
	// sessionErr is the text of the error of a session that failed to start. Empty
	// while connected.
	sessionErr string
	connected  bool
}

func (h edgeHealth) err() error {
	if h.sessionErr != "" {
		return errors.Mark(errors.New(h.sessionErr), driver.ErrTemporary)
	}
	if !h.connected {
		return errNotConnected
	}
	if h.commandErr != "" {
		return errors.Mark(errors.New(h.commandErr), driver.ErrTemporary)
	}
	return nil
}

// edgeNode is one Sparkplug B edge node: the sink of an edge task. It publishes the
// samples of its channels as NDATA messages on a client of its own, and writes the
// commands it receives to the command channels of its tags.
type edgeNode struct {
	ins alamos.Instrumentation
	// broker is the key of the broker device, whose edits the edge node follows.
	broker  device.Key
	devices *device.Service
	// command is the topic of the NCMD messages to the edge node.
	command   string
	tags      map[channel.Key][]publishedTag
	byName    map[string]publishedTag
	framer    *framer.Service
	task      task.Task
	authority control.Authority
	edge      *sparkplug.Edge
	// mu guards the sequence, which continues from one run to the next, and the state
	// of the current run. A sequenced publish encodes its message and hands it to the
	// client under mu, so messages leave the client in sequence order. The wait for the
	// broker happens outside the lock.
	mu struct {
		sync.Mutex
		seq    *sparkplug.Sequence
		client paho.Client
		// latest holds the last value of each tag, for the next birth. Every value fits
		// its tag.
		latest map[string]sparkplug.Metric
		// reported is the state that Health returned last.
		reported edgeHealth
		// commandErr is the text of the error of the last command. Empty when the
		// command was written.
		commandErr string
		// sessionErr is the error of the last session that failed to start. Nil after a
		// session that started.
		sessionErr error
		// cfg is the client config of the next connection attempt.
		cfg clientConfig
		// restart ends the current connection, so the next one takes cfg. Nil before
		// the first connection attempt.
		restart context.CancelFunc
	}
}

var _ driver.Sink = (*edgeNode)(nil)

// edgeSession is the driver.Writer of one run of an edge task: the goroutines of its
// connection and of the commands it receives.
type edgeSession struct {
	*edgeNode
	// commands carries the commands that the message handler received to writeCommands.
	commands chan sparkplug.Command
	// rebirths wakes the serve loop to publish the birth again.
	rebirths chan struct{}
	// edited wakes followBroker after an edit of the broker device.
	edited     chan struct{}
	disconnect observe.Disconnect
	// changed fires when the state that Health reports may have changed.
	changed chan struct{}
	cancel  context.CancelFunc
	stopped <-chan struct{}
}

var _ driver.Writer = (*edgeSession)(nil)

// Open implements driver.Sink. It connects with the current properties of the broker
// device, and returns after the first connection attempt. It returns an error when the
// edge node cannot take those properties.
func (n *edgeNode) Open(ctx context.Context) (driver.Writer, error) {
	e := &edgeSession{
		edgeNode: n,
		commands: make(chan sparkplug.Command, commandBacklog),
		rebirths: make(chan struct{}, 1),
		edited:   make(chan struct{}, 1),
		changed:  make(chan struct{}, 1),
	}
	// An edit after the read below wakes followBroker.
	e.disconnect = n.devices.Observe().OnChange(e.onDeviceChange)
	cfg, _, err := n.brokerConfig(ctx, clientConfig{})
	if err != nil {
		e.disconnect()
		return nil, err
	}
	n.mu.Lock()
	n.mu.cfg = cfg
	n.mu.restart = nil
	n.mu.commandErr = ""
	n.mu.sessionErr = nil
	n.mu.latest = make(map[string]sparkplug.Metric)
	n.mu.reported = edgeHealth{connected: true}
	n.mu.Unlock()
	var (
		settled    = make(chan struct{})
		settleOnce sync.Once
		settle     = func() { settleOnce.Do(func() { close(settled) }) }
	)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(e.ins))
	e.cancel = cancel
	sCtx.Go(func(ctx context.Context) error {
		defer settle()
		e.run(ctx, settle)
		return nil
	}, signal.RecoverWithErrOnPanic())
	sCtx.Go(func(ctx context.Context) error {
		e.writeCommands(ctx)
		return nil
	}, signal.RecoverWithErrOnPanic())
	sCtx.Go(func(ctx context.Context) error {
		e.followBroker(ctx)
		return nil
	}, signal.RecoverWithErrOnPanic())
	e.stopped = sCtx.Stopped()
	select {
	case <-settled:
	case <-ctx.Done():
		return nil, errors.Combine(ctx.Err(), e.Close())
	}
	// A broker that is down makes a warning, but a session that connected and failed to
	// start fails the start.
	n.mu.Lock()
	err = n.mu.sessionErr
	n.mu.Unlock()
	if err != nil {
		return nil, errors.Combine(err, e.Close())
	}
	return e, nil
}

// Close implements driver.Writer.
func (e *edgeSession) Close() error {
	e.disconnect()
	e.cancel()
	<-e.stopped
	return nil
}

// brokerConfig reads the broker device and returns the client config of the edge node.
// It returns false, and reads no TLS files, when the config matches current.
func (n *edgeNode) brokerConfig(
	ctx context.Context,
	current clientConfig,
) (clientConfig, bool, error) {
	var dev device.Device
	if err := n.devices.NewRetrieve().
		Where(device.MatchKeys(n.broker)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return clientConfig{}, false, err
	}
	props, err := parseProperties(dev)
	if err != nil {
		return clientConfig{}, false, err
	}
	cfg, err := newClientSettings(dev, props)
	if err != nil {
		return clientConfig{}, false, err
	}
	// The edge node is its own client, so it needs an ID of its own. Sparkplug B ties
	// the death message to the last will of one connection.
	cfg.clientID = deriveClientID(dev.Key + "/" + n.task.Key.String())
	if cfg.equal(current) {
		return current, false, nil
	}
	if cfg.tls, err = loadTLS(props); err != nil {
		return clientConfig{}, false, err
	}
	return cfg, true, nil
}

// onDeviceChange runs in the commit of a device change, so it wakes followBroker and
// never waits.
func (e *edgeSession) onDeviceChange(
	_ context.Context,
	r gorp.TxReader[device.Key, device.Device],
) {
	for ch := range r {
		if ch.Variant != change.VariantSet || ch.Key != e.broker {
			continue
		}
		select {
		case e.edited <- struct{}{}:
		default:
		}
	}
}

// followBroker moves the edge node to the new properties of each edit of the broker
// device until ctx is cancelled. The edge node keeps its config through an edit that it
// cannot take.
func (e *edgeSession) followBroker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.edited:
		}
		e.mu.Lock()
		current := e.mu.cfg
		e.mu.Unlock()
		cfg, changed, err := e.brokerConfig(ctx, current)
		if err != nil {
			if ctx.Err() == nil {
				e.ins.L.Warn(
					"the edge node cannot take the properties of the broker device",
					zap.Error(err),
				)
			}
			continue
		}
		if !changed {
			continue
		}
		e.mu.Lock()
		e.mu.cfg = cfg
		restart := e.mu.restart
		e.mu.Unlock()
		if restart != nil {
			restart()
		}
	}
}

// run keeps the edge node connected until ctx is cancelled. Each restart ends the
// current connection with an NDEATH, and the next one takes the current config.
func (e *edgeSession) run(ctx context.Context, settle func()) {
	for ctx.Err() == nil {
		connCtx, restart := context.WithCancel(ctx)
		e.mu.Lock()
		e.mu.restart = restart
		e.mu.Unlock()
		e.connect(connCtx, settle)
		restart()
	}
}

func (e *edgeSession) connect(ctx context.Context, settle func()) {
	reconnect(ctx, e.ins, reconnectHooks{
		options: func(lost chan<- error) *paho.ClientOptions {
			will, err := e.death()
			if err != nil {
				e.ins.L.DPanic("failed to encode the NDEATH message", zap.Error(err))
			}
			e.mu.Lock()
			cfg := e.mu.cfg
			e.mu.Unlock()
			return newClientOptions(cfg).
				SetBinaryWill(will.Topic, will.Payload, will.QoS, will.Retained).
				SetOrderMatters(true).
				SetDefaultPublishHandler(e.onMessage).
				SetConnectionLostHandler(reportLost(lost))
		},
		serve: func(ctx context.Context, client paho.Client, lost <-chan error) {
			defer e.endSession()
			err := waitSubscribe(ctx, client.Subscribe(e.command, 0, nil))
			if err == nil {
				err = e.publishBirth(ctx, client)
			}
			if err != nil {
				e.clearClient(err)
				if ctx.Err() == nil {
					e.ins.L.Warn(
						"failed to start the edge node session",
						zap.Error(err),
					)
				}
				settle()
				return
			}
			e.notify()
			settle()
			e.serve(ctx, client, lost)
			e.clearClient(nil)
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

// serve answers rebirth requests until ctx is cancelled or the connection is lost.
func (e *edgeSession) serve(
	ctx context.Context,
	client paho.Client,
	lost <-chan error,
) {
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
		}
	}
}

// writeCommands writes each command that the message handler receives until ctx is
// cancelled. It does not wait for a connection, so no command outlives the one that
// carried it.
func (e *edgeSession) writeCommands(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-e.commands:
			err := e.writeCommand(ctx, cmd)
			if ctx.Err() != nil {
				return
			}
			e.mu.Lock()
			e.mu.commandErr = ""
			if err != nil {
				e.mu.commandErr = err.Error()
			}
			e.notify()
			e.mu.Unlock()
		}
	}
}

// clearClient stops data from going out until the next birth. err is the error of a
// session that failed to start, or nil.
func (e *edgeSession) clearClient(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mu.client = nil
	e.mu.sessionErr = err
	e.notify()
}

// health returns the state that the task reports. The caller holds e.mu.
func (e *edgeSession) health() edgeHealth {
	h := edgeHealth{connected: e.mu.client != nil, commandErr: e.mu.commandErr}
	if !h.connected && e.mu.sessionErr != nil {
		h.sessionErr = e.mu.sessionErr.Error()
	}
	return h
}

func (e *edgeSession) notify() {
	select {
	case e.changed <- struct{}{}:
	default:
	}
}

// Health implements driver.Writer. It reports each loss and recovery of the connection,
// and each command that could not be written.
func (e *edgeSession) Health(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.changed:
			e.mu.Lock()
			h := e.health()
			same := h == e.mu.reported
			e.mu.reported = h
			e.mu.Unlock()
			if !same {
				return h.err()
			}
		}
	}
}

// publishBirth publishes the NBIRTH message that starts a new sequence on client, and
// makes client carry the data that follows it.
func (e *edgeSession) publishBirth(ctx context.Context, client paho.Client) error {
	token, err := e.queueBirth(client)
	if err != nil {
		return err
	}
	return wait(ctx, token)
}

func (e *edgeSession) queueBirth(client paho.Client) (paho.Token, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pub, err := e.mu.seq.Birth(e.mu.latest, telem.Now())
	if err != nil {
		return nil, err
	}
	e.mu.client = client
	return client.Publish(pub.Topic, pub.QoS, pub.Retained, pub.Payload), nil
}

func (n *edgeNode) death() (sparkplug.Publication, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.mu.seq.Death()
}

func (n *edgeNode) endSession() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.seq.EndSession()
}

func (e *edgeSession) publishDeath(ctx context.Context, client paho.Client) {
	pub, err := e.death()
	if err == nil {
		err = wait(ctx, client.Publish(pub.Topic, pub.QoS, pub.Retained, pub.Payload))
	}
	if err != nil && ctx.Err() == nil {
		e.ins.L.Warn("failed to publish the NDEATH message", zap.Error(err))
	}
}

// onMessage takes an NCMD message. It must not block: the client delivers messages in
// order on one goroutine.
func (e *edgeSession) onMessage(_ paho.Client, m paho.Message) {
	var msg sparkplug.Message
	topic, err := sparkplug.ParseTopic(m.Topic())
	if err == nil {
		msg, err = sparkplug.Decode(topic, m.Payload())
	}
	if err != nil {
		e.ins.L.Debug("dropped a Sparkplug B command", zap.Error(err))
		return
	}
	cmd := e.edge.Command(msg)
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
func (e *edgeSession) writeCommand(ctx context.Context, cmd sparkplug.Command) error {
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
		data, err := sparkplug.AppendSample(nil, t.command.dataType, m.Value)
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

// Write implements driver.Writer. It publishes the samples of fr in one NDATA message.
// A sample carries the timestamp of its index when the frame holds it. A sample that
// does not fit its tag is left out, and the temporary error names it.
func (e *edgeSession) Write(ctx context.Context, fr framer.Frame) error {
	var metrics []sparkplug.Metric
	for key, series := range fr.Entries() {
		for _, t := range e.tags[key] {
			stamps := indexOf(fr, t.index, series)
			for i := range int(series.Len()) {
				m := sparkplug.Metric{Name: t.name, Value: sparkplug.ValueAt(series, i)}
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
	token, err := e.queueData(metrics)
	if token == nil {
		return err
	}
	if wErr := wait(ctx, token); wErr != nil {
		return errors.Mark(wErr, driver.ErrTemporary)
	}
	return err
}

// queueData records the metrics that fit their tags as the latest values, and hands an
// NDATA message that carries them to the client. The error names each metric that does
// not fit. The token is nil when no message went to the client.
func (e *edgeSession) queueData(metrics []sparkplug.Metric) (paho.Token, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pub, carried, err := e.mu.seq.Data(metrics, telem.Now())
	for _, m := range carried {
		e.mu.latest[m.Name] = m
	}
	err = errors.Mark(err, driver.ErrTemporary)
	if e.mu.client == nil {
		return nil, errNotConnected
	}
	if pub.Payload == nil {
		return nil, err
	}
	return e.mu.client.Publish(pub.Topic, pub.QoS, pub.Retained, pub.Payload), err
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
