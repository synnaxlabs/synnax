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
	"cmp"
	"context"
	"fmt"
	"maps"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/net/proxy"
)

const (
	defaultSampleCount    = 20
	maxSampleCount        = 1000
	defaultSampleDuration = 5 * time.Second
	// samplePayloadLimit is the most bytes of a payload that a sample returns.
	samplePayloadLimit = 16 << 10
	// maxSampleTopics is the most topics that a sample keeps.
	maxSampleTopics = 500
	// sampleByteLimit is the most payload bytes that a sample holds while it reads.
	sampleByteLimit = 16 << 20
)

// The stages of a connection test.
const (
	stageDial    = "dial"
	stageConnect = "connect"
)

// testConnectionArgs describe a broker that need not be a stored device yet, so that
// the Console can test a connection before it saves the device.
type testConnectionArgs struct {
	// Device is the key of the broker device that the test edits. Empty for a broker
	// that is not a device yet.
	Device device.Key `json:"device"`
	// Location is the host of the broker.
	Location string `json:"location"`
	// Properties are the connection properties of the broker device.
	Properties msgpack.EncodedJSON `json:"properties"`
}

// test tests a connection to the broker of args one stage at a time: dial, TLS, and
// connect, where the broker checks the credentials. It stops after the first stage that
// fails. A broker whose tasks hold a connection with the same settings reports that
// connection instead. It returns a validation error when the properties of args are not
// valid, and an error when a certificate file fails to load.
func (s *scanner) test(
	ctx context.Context,
	args testConnectionArgs,
) ([]task.TestStep, error) {
	dev := device.Device{
		Key:        args.Device,
		Location:   args.Location,
		Properties: args.Properties,
	}
	props, err := parseProperties(dev)
	if err != nil {
		return nil, err
	}
	cfg, err := newClientConfigOf(dev, props)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(cfg.url)
	if err != nil {
		return nil, err
	}
	addr := u.Host
	if live, connected := s.pool.live(args.Device, cfg); live {
		if !connected {
			return []task.TestStep{failure(
				stageConnect, addr, "Connection down",
				"The tasks of this broker hold a connection with these settings. It "+
					"is down and tries again on its own.",
			)}, nil
		}
		return []task.TestStep{success(stageConnect, addr, "Connected")}, nil
	}
	// A broker drops the older of two sessions with one client ID, so each test derives
	// an ID of its own. A set ID is kept, because a broker can allow only it.
	if props.ClientID == "" {
		cfg.clientID = deriveClientID(uuid.New().String())
	}
	dialCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	conn, err := proxy.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return []task.TestStep{failure(
			stageDial, addr, "Failed to reach broker", fmt.Sprintf(
				"Could not reach %s: %s. Check the host and the port, and that the "+
					"network of the Core can reach the broker.",
				addr, err,
			),
		)}, nil
	}
	steps := []task.TestStep{success(stageDial, addr, "Reached broker")}
	if cfg.tls != nil {
		steps = append(steps, driver.Handshake(dialCtx, conn, addr, cfg.tls))
	}
	if err = conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return nil, err
	}
	if steps[len(steps)-1].Variant == status.VariantError {
		return steps, nil
	}
	if err = probe(ctx, s.ins, cfg); err != nil {
		return append(steps, connectFailure(addr, err)), nil
	}
	return append(steps, success(stageConnect, addr, "Connected")), nil
}

func success(stage, address, message string) task.TestStep {
	return task.TestStep{
		Stage:   stage,
		Address: address,
		Variant: status.VariantSuccess,
		Message: message,
	}
}

func failure(stage, address, message, description string) task.TestStep {
	return task.TestStep{
		Stage:       stage,
		Address:     address,
		Variant:     status.VariantError,
		Message:     message,
		Description: description,
	}
}

func connectFailure(addr string, err error) task.TestStep {
	switch {
	case errors.Is(err, packets.ErrorRefusedNotAuthorised),
		errors.Is(err, packets.ErrorRefusedBadUsernameOrPassword):
		return failure(stageConnect, addr, "Authentication failed", fmt.Sprintf(
			"%s. Check the username and the password.", err,
		))
	case errors.Is(err, packets.ErrorRefusedIDRejected):
		return failure(stageConnect, addr, "Client ID rejected", fmt.Sprintf(
			"%s. Set a client ID that the broker accepts.", err,
		))
	case errors.Is(err, packets.ErrorRefusedServerUnavailable):
		return failure(stageConnect, addr, "Broker not available", fmt.Sprintf(
			"%s. Try again later.", err,
		))
	default:
		return failure(stageConnect, addr, "Failed to connect", fmt.Sprintf(
			"%s. Check that the TLS setting matches the port of the broker.", err,
		))
	}
}

// browseArgs select what a browse listens to.
type browseArgs struct {
	// Device is the key of the broker device to browse.
	Device device.Key `json:"device"`
	// Filter is the topic filter to subscribe to. Empty selects every topic.
	Filter string `json:"filter"`
	// Duration is the time to listen for, in milliseconds. Zero selects 3 s, and the
	// most is 20 s.
	Duration int `json:"duration"`
	// Limit is the most topics to return. Zero selects 500, and the most is 5,000.
	Limit int `json:"limit"`
}

// heardTopic is what a browse heard on one topic. It holds what the answer shows of the
// last message, not the message.
type heardTopic struct {
	browsedTopic
	// retained is the number of retained messages heard.
	retained int
}

// heard is what a browse heard on each topic.
type heard struct {
	topics    []*heardTopic
	truncated bool
	duration  time.Duration
}

// hear listens to the filter of args for its duration, and keeps what the answer shows
// of the last message of each topic.
func (s *scanner) hear(ctx context.Context, args browseArgs) (heard, error) {
	if args.Filter == "" {
		args.Filter = defaultBrowseFilter
	}
	if args.Limit <= 0 {
		args.Limit = defaultBrowseLimit
	}
	args.Limit = min(args.Limit, maxBrowseLimit)
	var (
		mu  sync.Mutex
		res = heard{duration: browseDuration(args.Duration)}
		by  = make(map[string]*heardTopic)
	)
	if err := s.listen(
		ctx,
		args.Device,
		[]string{args.Filter},
		res.duration,
		func(m paho.Message) bool {
			mu.Lock()
			defer mu.Unlock()
			t, seen := by[m.Topic()]
			if !seen {
				if len(by) >= args.Limit {
					res.truncated = true
					return false
				}
				t = &heardTopic{Topic: m.Topic()}
				by[m.Topic()] = t
			}
			t.Count++
			if m.Retained() {
				t.retained++
			}
			p := m.Payload()
			t.LastSeen, t.Retained = telem.Now(), m.Retained()
			t.Payload, t.Format, t.Fields = payloadSample(p), driver.FormatOf(p),
				payloadFields(p)
			return false
		},
	); err != nil {
		return heard{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	for _, t := range by {
		res.topics = append(res.topics, t)
	}
	slices.SortFunc(res.topics, func(a, b *heardTopic) int {
		return strings.Compare(a.Topic, b.Topic)
	})
	return res, nil
}

// sampleArgs select the messages that a sample reads.
type sampleArgs struct {
	// Device is the key of the broker device to sample.
	Device device.Key `json:"device"`
	// Topics are the topic filters to subscribe to.
	Topics []string `json:"topics"`
	// Count is the most messages to keep from each topic. Zero selects 20, and the most
	// is 1,000.
	Count int `json:"count"`
	// Duration is the longest time to listen for, in milliseconds. Zero selects 5 s,
	// and the most is 20 s.
	Duration int `json:"duration"`
}

// sampledMessage is one message that a sample read.
type sampledMessage struct {
	// Time is the time that the message arrived.
	Time telem.TimeStamp `json:"time"`
	// Format is the format of the whole payload.
	Format driver.Format `json:"format"`
	// Payload is the start of the payload, from at most 16 KiB: text when Format is
	// json or text, and hex digits otherwise.
	Payload string `json:"payload"`
	// Retained is true when the message is the retained message of its topic.
	Retained bool `json:"retained"`
}

// sampledTopic is the sample of one topic.
type sampledTopic struct {
	// Topic is the name of the topic.
	Topic string `json:"topic"`
	// Messages are the messages read, in the order that they arrived.
	Messages []sampledMessage `json:"messages"`
}

// sampleResult is the data that answers a sample.
type sampleResult struct {
	// Topics are the topics read, sorted by name.
	Topics []sampledTopic `json:"topics"`
	// Shapes are the shapes of the JSON payloads. The source of a shape is a topic.
	Shapes []xjson.Shape `json:"shapes"`
	// Truncated is true when the broker delivered more topics or payload bytes than a
	// sample holds, or when the messages read do not all fit in the answer. The shapes
	// still come from every message read.
	Truncated bool `json:"truncated"`
}

// heardMessage is a message that a sample read, with its whole payload.
type heardMessage struct {
	sampledMessage
	payload []byte
}

// sample listens to the topics of args until each holds args.Count messages, or until
// args.Duration passes when a topic is a wildcard filter, and infers the shapes of the
// payloads. The answer keeps the messages that driver.Fit keeps, and the shapes come
// from every message read.
func (s *scanner) sample(ctx context.Context, args sampleArgs) (sampleResult, error) {
	if len(args.Topics) == 0 {
		return sampleResult{}, errors.Wrap(
			validate.ErrValidation, "topics: must be non-empty",
		)
	}
	count := min(max(cmp.Or(args.Count, defaultSampleCount), 1), maxSampleCount)
	duration := min(
		cmp.Or(time.Duration(args.Duration)*time.Millisecond, defaultSampleDuration),
		maxBrowseDuration,
	)
	// Only a set of exact topics can be complete before the duration ends.
	literal := !slices.ContainsFunc(args.Topics, func(t string) bool {
		return strings.ContainsAny(t, "+#")
	})
	var (
		mu         sync.Mutex
		by         = make(map[string][]heardMessage)
		full, held int
		truncated  bool
	)
	if err := s.listen(
		ctx,
		args.Device,
		args.Topics,
		duration,
		func(m paho.Message) bool {
			mu.Lock()
			defer mu.Unlock()
			messages, seen := by[m.Topic()]
			payload := m.Payload()
			switch {
			case seen && len(messages) == count:
				return false
			case !seen && len(by) == maxSampleTopics:
				truncated = true
				return false
			case held+len(payload) > sampleByteLimit:
				truncated = true
				return true
			}
			held += len(payload)
			msg := sampledMessage{Time: telem.Now(), Retained: m.Retained()}
			msg.Format, msg.Payload = driver.Display(payload, samplePayloadLimit)
			by[m.Topic()] = append(
				messages,
				heardMessage{sampledMessage: msg, payload: payload},
			)
			if len(by[m.Topic()]) == count {
				full++
			}
			return literal && full >= len(args.Topics)
		},
	); err != nil {
		return sampleResult{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	var (
		names   = slices.Sorted(maps.Keys(by))
		sampled = make([][]sampledMessage, len(names))
		docs    []xjson.Document
	)
	for i, name := range names {
		for _, m := range by[name] {
			sampled[i] = append(sampled[i], m.sampledMessage)
			docs = append(
				docs,
				xjson.Document{Source: name, Data: m.payload, Time: m.Time},
			)
		}
	}
	res := sampleResult{
		Topics: make([]sampledTopic, len(names)),
		Shapes: xjson.Infer(docs),
	}
	for i, name := range names {
		res.Topics[i].Topic = name
	}
	kept, cut, err := driver.Fit(res, sampled)
	if err != nil {
		return sampleResult{}, err
	}
	for i := range res.Topics {
		res.Topics[i].Messages = kept[i]
	}
	res.Truncated = truncated || cut
	return res, nil
}

// listen subscribes a client of its own to filters on the broker device key, and passes
// each message to handle until duration passes, ctx ends, or handle returns true.
func (s *scanner) listen(
	ctx context.Context,
	key device.Key,
	filters []string,
	duration time.Duration,
	handle func(paho.Message) (stop bool),
) error {
	_, _, cfg, err := s.browseConfig(ctx, key)
	if err != nil {
		return err
	}
	var (
		stopped = make(chan struct{})
		once    sync.Once
	)
	opts := newClientOptions(cfg)
	opts.SetDefaultPublishHandler(func(_ paho.Client, m paho.Message) {
		if handle(m) {
			once.Do(func() { close(stopped) })
		}
	})
	client, disconnect, err := open(ctx, s.ins, opts)
	if err != nil {
		return err
	}
	defer disconnect()
	subs := make(map[string]byte, len(filters))
	for _, f := range filters {
		subs[f] = 0
	}
	if err = waitSubscribe(ctx, client.SubscribeMultiple(subs, nil)); err != nil {
		return err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	case <-stopped:
	}
	return nil
}
