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
)

// The stages of a connection test.
const (
	stageDial    = "dial"
	stageConnect = "connect"
)

// TestRequest describes a broker to test a connection to. The broker need not be a
// stored device yet.
type TestRequest struct {
	// Device is the key of the broker device that the test edits. Empty for a broker
	// that is not a device yet.
	Device device.Key `json:"device" msgpack:"device"`
	// Location is the host of the broker.
	Location string `json:"location" msgpack:"location"`
	// Properties are the connection properties of the broker device.
	Properties msgpack.EncodedJSON `json:"properties" msgpack:"properties"`
}

// TestConnection tests a connection to the broker of req one stage at a time: dial,
// TLS, and connect, where the broker checks the credentials. It stops after the first
// stage that fails. A broker whose tasks hold a connection with the same settings
// reports that connection instead. It returns a validation error when the properties of
// req are not valid, and an error when a certificate file fails to load.
func (f *Factory) TestConnection(
	ctx context.Context,
	req TestRequest,
) ([]driver.TestStep, error) {
	return f.scanner().test(ctx, req)
}

// Browse lists the topics that the broker device of req delivers while a client of its
// own listens, sorted by name.
func (f *Factory) Browse(ctx context.Context, req BrowseRequest) (BrowseResult, error) {
	heard, err := f.scanner().hear(ctx, req)
	if err != nil {
		return BrowseResult{}, err
	}
	res := BrowseResult{
		Topics:    make([]BrowsedTopic, len(heard.topics)),
		Truncated: heard.truncated,
	}
	for i, t := range heard.topics {
		res.Topics[i] = BrowsedTopic{
			Topic:    t.topic,
			Count:    t.count,
			Rate:     float64(t.count-t.retained) / heard.duration.Seconds(),
			LastSeen: t.lastSeen,
			Payload:  t.last[:min(len(t.last), browsePayloadLimit)],
			Format:   driver.FormatOf(t.last),
			Retained: t.lastRetained,
		}
	}
	return res, nil
}

// Sample listens to the topics of req until each holds req.Count messages, or until
// req.Duration passes when a topic is a wildcard filter, and infers the shapes of the
// payloads.
func (f *Factory) Sample(ctx context.Context, req SampleRequest) (SampleResult, error) {
	return f.scanner().sample(ctx, req)
}

func (f *Factory) scanner() *scanner {
	return &scanner{ins: f.cfg.Instrumentation, pool: f.pool, device: f.cfg.Device}
}

func (s *scanner) test(
	ctx context.Context,
	req TestRequest,
) ([]driver.TestStep, error) {
	dev := device.Device{
		Key:        req.Device,
		Location:   req.Location,
		Properties: req.Properties,
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
	if live, connected := s.pool.live(req.Device, cfg); live {
		if !connected {
			return []driver.TestStep{failure(
				stageConnect, addr, "Connection down",
				"The tasks of this broker hold a connection with these settings. It "+
					"is down and tries again on its own.",
			)}, nil
		}
		return []driver.TestStep{success(stageConnect, addr, "Connected")}, nil
	}
	// A broker drops the older of two sessions with one client ID. A derived ID gets a
	// suffix of its own, but a set ID is kept, because a broker can allow only it.
	if props.ClientID == "" {
		cfg.clientID = deriveClientID(cmp.Or(req.Device, uuid.New().String()) + "/test")
	}
	dialCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	conn, err := proxy.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return []driver.TestStep{failure(
			stageDial, addr, "Failed to reach broker", fmt.Sprintf(
				"Could not reach %s: %s. Check the host and the port, and that the "+
					"network of the Core can reach the broker.",
				addr, err,
			),
		)}, nil
	}
	steps := []driver.TestStep{success(stageDial, addr, "Reached broker")}
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

func success(stage, address, message string) driver.TestStep {
	return driver.TestStep{
		Stage:   stage,
		Address: address,
		Variant: status.VariantSuccess,
		Message: message,
	}
}

func failure(stage, address, message, description string) driver.TestStep {
	return driver.TestStep{
		Stage:       stage,
		Address:     address,
		Variant:     status.VariantError,
		Message:     message,
		Description: description,
	}
}

func connectFailure(addr string, err error) driver.TestStep {
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

// BrowseRequest selects what a browse listens to.
type BrowseRequest struct {
	// Device is the key of the broker device to browse.
	Device device.Key `json:"device" msgpack:"device"`
	// Filter is the topic filter to subscribe to. Empty selects every topic.
	Filter string `json:"filter" msgpack:"filter"`
	// Duration is the time to listen for, in milliseconds. Zero selects 3 s, and the
	// most is 20 s.
	Duration int `json:"duration" msgpack:"duration"`
	// Limit is the most topics to return. Zero selects 500, and the most is 5,000.
	Limit int `json:"limit" msgpack:"limit"`
}

// BrowsedTopic is one topic that a browse heard.
type BrowsedTopic struct {
	// Topic is the name of the topic.
	Topic string `json:"topic" msgpack:"topic"`
	// Count is the number of messages heard, with a retained message.
	Count int `json:"count" msgpack:"count"`
	// Rate is the number of messages per second heard, without a retained message.
	Rate float64 `json:"rate" msgpack:"rate"`
	// LastSeen is the time that the last message arrived.
	LastSeen telem.TimeStamp `json:"last_seen" msgpack:"last_seen"`
	// Payload is the start of the last payload, at most 256 bytes.
	Payload []byte `json:"payload" msgpack:"payload"`
	// Format is the format of the last payload.
	Format driver.Format `json:"format" msgpack:"format"`
	// Retained is true when the last message is the retained message of the topic.
	Retained bool `json:"retained" msgpack:"retained"`
}

// BrowseResult is what a browse heard.
type BrowseResult struct {
	// Topics are the topics heard, sorted by name.
	Topics []BrowsedTopic `json:"topics" msgpack:"topics"`
	// Truncated is true when the broker delivered more topics than the limit.
	Truncated bool `json:"truncated" msgpack:"truncated"`
}

// heardTopic is what a listen heard on one topic.
type heardTopic struct {
	topic           string
	count, retained int
	last            []byte
	lastSeen        telem.TimeStamp
	lastRetained    bool
}

// heard is what a browse heard on each topic.
type heard struct {
	topics    []*heardTopic
	truncated bool
	duration  time.Duration
}

// hear listens to the filter of req for its duration, and keeps the last message of
// each topic.
func (s *scanner) hear(ctx context.Context, req BrowseRequest) (heard, error) {
	if req.Filter == "" {
		req.Filter = defaultBrowseFilter
	}
	if req.Limit <= 0 {
		req.Limit = defaultBrowseLimit
	}
	req.Limit = min(req.Limit, maxBrowseLimit)
	var (
		mu  sync.Mutex
		res = heard{duration: browseDuration(req.Duration)}
		by  = make(map[string]*heardTopic)
	)
	if err := s.listen(
		ctx,
		req.Device,
		[]string{req.Filter},
		res.duration,
		func(m paho.Message) bool {
			mu.Lock()
			defer mu.Unlock()
			t, seen := by[m.Topic()]
			if !seen {
				if len(by) >= req.Limit {
					res.truncated = true
					return false
				}
				t = &heardTopic{topic: m.Topic()}
				by[m.Topic()] = t
			}
			t.count++
			if m.Retained() {
				t.retained++
			}
			t.last, t.lastSeen, t.lastRetained = m.Payload(), telem.Now(), m.Retained()
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
		return strings.Compare(a.topic, b.topic)
	})
	return res, nil
}

// SampleRequest selects the messages that a sample reads.
type SampleRequest struct {
	// Device is the key of the broker device to sample.
	Device device.Key `json:"device" msgpack:"device"`
	// Topics are the topic filters to subscribe to.
	Topics []string `json:"topics" msgpack:"topics"`
	// Count is the most messages to keep from each topic. Zero selects 20, and the most
	// is 1,000.
	Count int `json:"count" msgpack:"count"`
	// Duration is the longest time to listen for, in milliseconds. Zero selects 5 s,
	// and the most is 20 s.
	Duration int `json:"duration" msgpack:"duration"`
}

// SampledMessage is one message that a sample read.
type SampledMessage struct {
	// Time is the time that the message arrived.
	Time telem.TimeStamp `json:"time" msgpack:"time"`
	// Payload is the start of the payload, at most 16 KiB.
	Payload []byte `json:"payload" msgpack:"payload"`
	// Retained is true when the message is the retained message of its topic.
	Retained bool `json:"retained" msgpack:"retained"`
}

// SampledTopic is the sample of one topic.
type SampledTopic struct {
	// Topic is the name of the topic.
	Topic string `json:"topic" msgpack:"topic"`
	// Messages are the messages read, in the order that they arrived.
	Messages []SampledMessage `json:"messages" msgpack:"messages"`
}

// SampleResult is what a sample read.
type SampleResult struct {
	// Topics are the topics read, sorted by name.
	Topics []SampledTopic `json:"topics" msgpack:"topics"`
	// Shapes are the shapes of the JSON payloads. The source of a shape is a topic.
	Shapes []xjson.Shape `json:"shapes" msgpack:"shapes"`
}

type sampledTopic struct {
	SampledTopic
	payloads [][]byte
}

func (s *scanner) sample(ctx context.Context, req SampleRequest) (SampleResult, error) {
	if len(req.Topics) == 0 {
		return SampleResult{}, errors.Wrap(
			validate.ErrValidation, "topics: must be non-empty",
		)
	}
	count := min(max(cmp.Or(req.Count, defaultSampleCount), 1), maxSampleCount)
	duration := min(
		cmp.Or(time.Duration(req.Duration)*time.Millisecond, defaultSampleDuration),
		maxBrowseDuration,
	)
	// Only a set of exact topics can be complete before the duration ends.
	literal := !slices.ContainsFunc(req.Topics, func(t string) bool {
		return strings.ContainsAny(t, "+#")
	})
	var (
		mu   sync.Mutex
		by   = make(map[string]*sampledTopic)
		full int
	)
	if err := s.listen(
		ctx,
		req.Device,
		req.Topics,
		duration,
		func(m paho.Message) bool {
			mu.Lock()
			defer mu.Unlock()
			t, ok := by[m.Topic()]
			if !ok {
				t = &sampledTopic{Topic: m.Topic()}
				by[m.Topic()] = t
			}
			if len(t.Messages) == count {
				return false
			}
			payload := m.Payload()
			t.Messages = append(t.Messages, SampledMessage{
				Time:     telem.Now(),
				Payload:  payload[:min(len(payload), samplePayloadLimit)],
				Retained: m.Retained(),
			})
			t.payloads = append(t.payloads, payload)
			if len(t.Messages) == count {
				full++
			}
			return literal && full >= len(req.Topics)
		},
	); err != nil {
		return SampleResult{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	var (
		res  = SampleResult{Topics: make([]SampledTopic, 0, len(by))}
		docs []xjson.Document
	)
	for _, name := range slices.Sorted(maps.Keys(by)) {
		t := by[name]
		res.Topics = append(res.Topics, t.SampledTopic)
		for i, p := range t.payloads {
			docs = append(docs, xjson.Document{
				Source: name,
				Data:   p,
				Time:   t.Messages[i].Time,
			})
		}
	}
	res.Shapes = xjson.Infer(docs)
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
