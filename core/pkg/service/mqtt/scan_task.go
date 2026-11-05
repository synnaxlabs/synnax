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
	"bytes"
	"context"
	"encoding/json/jsontext"
	"io"
	"slices"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
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
	// browseFieldLimit is the largest count of fields of a browsed topic.
	browseFieldLimit = 256
	// browseSampleLimit is the longest field sample, in bytes, of a browsed topic.
	browseSampleLimit = 64
)

var errReconnecting = errors.New("the connection is down and tries again on its own")

// testConnectionArgs describes a broker that may not be a stored device yet, so that
// the Console can test a connection before it saves the device.
type testConnectionArgs struct {
	Properties msgpack.EncodedJSON `json:"properties"`
	Location   string              `json:"location"`
	// Device is the key of the broker device that the test edits. Empty for a broker
	// that is not a device yet.
	Device device.Key `json:"device"`
}

// browsedField is one scalar of a browsed JSON payload.
type browsedField struct {
	Pointer string `json:"pointer"`
	// Type is the JSON type of the scalar: "number", "string", or "boolean".
	Type string `json:"type"`
	// Sample is the start of the JSON text of the scalar.
	Sample string `json:"sample"`
}

// browsedTopic is one topic that a browse saw, with its last payload.
type browsedTopic struct {
	Topic string `json:"topic"`
	// Payload is the start of the last payload. Empty when the payload is not text.
	Payload string `json:"payload"`
	// Fields are the scalars of the whole last payload. Empty when it is not JSON.
	Fields   []browsedField `json:"fields"`
	Retained bool           `json:"retained"`
}

type browseResult struct {
	Topics []browsedTopic `json:"topics"`
	// Truncated is true when the broker delivered more topics than the limit.
	Truncated bool `json:"truncated"`
}

// scanner is the driver.Scanner of the MQTT scan task.
type scanner struct {
	ins    alamos.Instrumentation
	pool   *pool
	device *device.Service
}

var _ driver.Scanner = (*scanner)(nil)

// Check implements driver.Scanner. A broker with running tasks reports the state of
// their shared connection, and the error of the last edit of its properties that the
// connection could not take. Any other broker gets a test connection.
func (s *scanner) Check(ctx context.Context, dev device.Device) error {
	pooled, connected, err := s.pool.state(dev.Key)
	if err != nil || connected {
		return err
	}
	if pooled {
		return errReconnecting
	}
	cfg, err := newClientConfig(dev)
	if err != nil {
		return err
	}
	// A task can connect while the probe runs, and a broker drops the older of two
	// sessions with one client ID.
	cfg.clientID = deriveClientID(dev.Key + "/probe")
	return probe(ctx, s.ins, cfg)
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
	// A broker drops the older of two sessions with one client ID, so a probe with the
	// settings of the connection that tasks share would drop that connection.
	if live, connected := s.pool.live(args.Device, cfg); live {
		if !connected {
			return errReconnecting
		}
		return nil
	}
	return probe(ctx, s.ins, cfg)
}

// probe reports whether a client can connect to the broker of cfg.
func probe(ctx context.Context, ins alamos.Instrumentation, cfg clientConfig) error {
	_, disconnect, err := open(ctx, ins, newClientOptions(cfg))
	if err != nil {
		return err
	}
	disconnect()
	return nil
}

func (s *scanner) browse(ctx context.Context, cmd task.Command) (browseResult, error) {
	var req BrowseRequest
	if err := cmd.Args.Unmarshal(&req); err != nil {
		return browseResult{}, errors.Wrapf(
			validate.ErrValidation, "invalid arguments: %s", err.Error(),
		)
	}
	heard, err := s.hear(ctx, req)
	if err != nil {
		return browseResult{}, err
	}
	res := browseResult{
		Topics:    make([]browsedTopic, len(heard.topics)),
		Truncated: heard.truncated,
	}
	for i, t := range heard.topics {
		res.Topics[i] = browsedTopic{
			Topic:    t.topic,
			Payload:  payloadSample(t.last),
			Fields:   payloadFields(t.last),
			Retained: t.lastRetained,
		}
	}
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

// browseConfig returns the broker device of a browse, its properties, and the config
// of its client. A browse has its own client and client ID. A wildcard subscription
// on the shared connection would overlap the subscriptions of the tasks.
func (s *scanner) browseConfig(
	ctx context.Context,
	key device.Key,
) (device.Device, properties, clientConfig, error) {
	var dev device.Device
	if err := s.device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return device.Device{}, properties{}, clientConfig{}, err
	}
	props, err := parseProperties(dev)
	if err != nil {
		return device.Device{}, properties{}, clientConfig{}, err
	}
	cfg, err := newClientConfigOf(dev, props)
	if err != nil {
		return device.Device{}, properties{}, clientConfig{}, err
	}
	// A broker drops the older of two sessions with one client ID, so each browse and
	// sample derives an ID of its own.
	cfg.clientID = deriveClientID(uuid.New().String())
	// A browse is never the host application, so it sets no STATE last will.
	cfg.hostID = ""
	return dev, props, cfg, nil
}

// payloadSample returns the start of a text payload.
func payloadSample(payload []byte) string {
	if !utf8.Valid(payload) {
		return ""
	}
	return cutText(payload, browsePayloadLimit)
}

// cutText returns at most limit bytes from the start of text, cut at a rune boundary.
func cutText(text []byte, limit int) string {
	if len(text) <= limit {
		return string(text)
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return string(text[:limit])
}

// payloadFields returns the scalars of a JSON payload in document order, at most
// browseFieldLimit of them. Arrays and nulls give no field. It returns no fields when
// the payload is not JSON, is not UTF-8, or repeats a name in an object.
func payloadFields(payload []byte) []browsedField {
	fields := []browsedField{}
	dec := jsontext.NewDecoder(bytes.NewReader(payload))
	if err := appendFields(dec, nil, &fields); err != nil {
		return []browsedField{}
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return []browsedField{}
	}
	return fields
}

// appendFields appends the scalars of the next value of dec, at pointer, to fields.
func appendFields(
	dec *jsontext.Decoder,
	pointer xjson.Pointer,
	fields *[]browsedField,
) error {
	var t xjson.Type
	switch dec.PeekKind() {
	case '{':
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for dec.PeekKind() != '}' {
			name, err := dec.ReadToken()
			if err != nil {
				return err
			}
			child := append(slices.Clip(pointer), name.String())
			if err = appendFields(dec, child, fields); err != nil {
				return err
			}
		}
		_, err := dec.ReadToken()
		return err
	case '"':
		t = xjson.String
	case '0':
		t = xjson.Number
	case 't', 'f':
		t = xjson.Boolean
	default:
		return dec.SkipValue()
	}
	value, err := dec.ReadValue()
	if err != nil || len(*fields) == browseFieldLimit {
		return err
	}
	*fields = append(*fields, browsedField{
		Pointer: pointer.String(),
		Type:    t.String(),
		Sample:  cutText(value, browseSampleLimit),
	})
	return nil
}
