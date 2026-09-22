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
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const (
	// testConnectionCommand connects to the broker that its arguments describe.
	testConnectionCommand = "test_connection"
	// browseCommand lists the topics that a broker device delivers for a time.
	browseCommand = "browse"

	defaultBrowseFilter   = "#"
	defaultBrowseDuration = 3 * time.Second
	maxBrowseDuration     = 20 * time.Second
	defaultBrowseLimit    = 500
	maxBrowseLimit        = 5000
	// browsePayloadLimit is the longest payload sample, in bytes, of a browsed topic.
	browsePayloadLimit = 256
)

// testConnectionArgs describes a broker that may not be a stored device yet, so that
// the Console can test a connection before it saves the device.
type testConnectionArgs struct {
	Properties msgpack.EncodedJSON `json:"properties"`
	Location   string              `json:"location"`
}

type browseArgs struct {
	// Device is the key of the broker device to browse.
	Device device.Key `json:"device"`
	// Filter is the topic filter to subscribe to. Empty selects every topic.
	Filter string `json:"filter"`
	// Duration is the time to listen for, in milliseconds. Zero selects 3 s.
	Duration int `json:"duration"`
	// Limit is the largest count of topics to return. Zero selects 500.
	Limit int `json:"limit"`
}

// browsedTopic is one topic that a browse saw, with its last payload.
type browsedTopic struct {
	Topic string `json:"topic"`
	// Payload is the start of the last payload. Empty when the payload is not text.
	Payload  string `json:"payload"`
	Retained bool   `json:"retained"`
}

type browseResult struct {
	Topics []browsedTopic `json:"topics"`
	// Truncated is true when the broker delivered more topics than the limit.
	Truncated bool `json:"truncated"`
}

// scanner is the driver.Scanner of the MQTT scan task.
type scanner struct {
	pool   *pool
	device *device.Service
}

var _ driver.Scanner = (*scanner)(nil)

// Check implements driver.Scanner. A broker with running tasks reports the state of
// their shared connection. Any other broker gets a test connection.
func (s *scanner) Check(ctx context.Context, dev device.Device) error {
	pooled, connected, err := s.pool.state(dev)
	if err != nil || connected {
		return err
	}
	if pooled {
		return errors.New("the connection is down and tries again on its own")
	}
	cfg, err := newClientConfig(dev)
	if err != nil {
		return err
	}
	return probe(ctx, cfg)
}

// Exec implements driver.Scanner.
func (s *scanner) Exec(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	switch cmd.Type {
	case testConnectionCommand:
		return nil, s.testConnection(ctx, cmd)
	case browseCommand:
		res, err := s.browse(ctx, cmd)
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(res)
	case browseSparkplugCommand:
		res, err := s.browseSparkplug(ctx, cmd)
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(res)
	}
	return nil, driver.ErrUnsupportedCommand
}

func (s *scanner) testConnection(ctx context.Context, cmd task.Command) error {
	var args testConnectionArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"invalid arguments: %s",
			err.Error(),
		)
	}
	cfg, err := newClientConfig(device.Device{
		Key:        cmd.Key,
		Location:   args.Location,
		Properties: args.Properties,
	})
	if err != nil {
		return err
	}
	return probe(ctx, cfg)
}

// probe reports whether a client can connect to the broker of cfg.
func probe(ctx context.Context, cfg clientConfig) error {
	_, disconnect, err := open(ctx, newClientOptions(cfg))
	if err != nil {
		return err
	}
	disconnect()
	return nil
}

func (s *scanner) browse(ctx context.Context, cmd task.Command) (browseResult, error) {
	var (
		args browseArgs
		res  = browseResult{Topics: []browsedTopic{}}
	)
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return res, errors.Wrapf(
			validate.ErrValidation, "invalid arguments: %s", err.Error(),
		)
	}
	if args.Filter == "" {
		args.Filter = defaultBrowseFilter
	}
	duration := browseDuration(args.Duration)
	if args.Limit <= 0 {
		args.Limit = defaultBrowseLimit
	}
	args.Limit = min(args.Limit, maxBrowseLimit)
	_, cfg, err := s.browseConfig(ctx, args.Device)
	if err != nil {
		return res, err
	}
	var (
		mu     sync.Mutex
		topics = make(map[string]browsedTopic)
	)
	opts := newClientOptions(cfg)
	opts.SetDefaultPublishHandler(func(_ paho.Client, m paho.Message) {
		mu.Lock()
		defer mu.Unlock()
		if _, seen := topics[m.Topic()]; !seen && len(topics) >= args.Limit {
			res.Truncated = true
			return
		}
		topics[m.Topic()] = browsedTopic{
			Topic:    m.Topic(),
			Payload:  payloadSample(m.Payload()),
			Retained: m.Retained(),
		}
	})
	client, disconnect, err := open(ctx, opts)
	if err != nil {
		return res, err
	}
	defer disconnect()
	if err = wait(ctx, client.Subscribe(args.Filter, 0, nil)); err != nil {
		return res, err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case <-timer.C:
	}
	mu.Lock()
	defer mu.Unlock()
	for _, t := range topics {
		res.Topics = append(res.Topics, t)
	}
	sort.Slice(res.Topics, func(i, j int) bool {
		return res.Topics[i].Topic < res.Topics[j].Topic
	})
	return res, nil
}

// browseDuration returns the time that a browse listens for, from its argument in
// milliseconds.
func browseDuration(ms int) time.Duration {
	duration := time.Duration(ms) * time.Millisecond
	if duration <= 0 {
		duration = defaultBrowseDuration
	}
	return min(duration, maxBrowseDuration)
}

// browseConfig returns the broker device of a browse and the config of its client. A
// browse has its own client and client ID. A wildcard subscription on the shared
// connection would overlap the subscriptions of the tasks.
func (s *scanner) browseConfig(
	ctx context.Context,
	key device.Key,
) (device.Device, clientConfig, error) {
	var dev device.Device
	if err := s.device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return dev, clientConfig{}, err
	}
	cfg, err := newClientConfig(dev)
	cfg.clientID = deriveClientID(dev.Key + "/browse")
	// A browse is never the host application, so it sets no STATE last will.
	cfg.hostID = ""
	return dev, cfg, err
}

// payloadSample returns the start of a text payload, cut at a rune boundary.
func payloadSample(payload []byte) string {
	if !utf8.Valid(payload) {
		return ""
	}
	if len(payload) <= browsePayloadLimit {
		return string(payload)
	}
	cut := browsePayloadLimit
	for cut > 0 && !utf8.RuneStart(payload[cut]) {
		cut--
	}
	return string(payload[:cut])
}
