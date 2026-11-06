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

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	// testConnectionCommand tests a connection to the broker that its arguments
	// describe, one stage at a time.
	testConnectionCommand = "test_connection"
	// browseCommand lists the topics that a broker device delivers for a time.
	browseCommand = "browse"
	// sampleCommand reads messages from topics of a broker device and infers their
	// shapes.
	sampleCommand = "sample"

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

// browsedField is one scalar of a browsed JSON payload.
type browsedField struct {
	Pointer string `json:"pointer"`
	// Type is the JSON type of the scalar: "number", "string", or "boolean".
	Type string `json:"type"`
	// Sample is the start of the JSON text of the scalar.
	Sample string `json:"sample"`
}

// browsedTopic is one topic that a browse heard, with its last payload.
type browsedTopic struct {
	Topic string `json:"topic"`
	// Count is the number of messages heard, with a retained message.
	Count int `json:"count"`
	// Rate is the number of messages per second heard, without a retained message.
	Rate float64 `json:"rate"`
	// LastSeen is the time that the last message arrived.
	LastSeen telem.TimeStamp `json:"last_seen"`
	// Payload is the start of the last payload. Empty when the payload is not text.
	Payload string `json:"payload"`
	// Format is the format of the whole last payload.
	Format driver.Format `json:"format"`
	// Fields are the scalars of the whole last payload. Empty when it is not JSON.
	Fields   []browsedField `json:"fields"`
	Retained bool           `json:"retained"`
}

type browseResult struct {
	Topics []browsedTopic `json:"topics"`
	// Truncated is true when the broker delivered more topics than the limit, or when
	// the topics do not all fit in the answer.
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
		var args testConnectionArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		steps, err := s.test(ctx, args)
		if err != nil {
			return nil, err
		}
		return driver.AnswerTest(steps)
	case browseCommand:
		var args browseArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		res, err := s.browse(ctx, args)
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(res)
	case sampleCommand:
		var args sampleArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		res, err := s.sample(ctx, args)
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

// unmarshalArgs decodes the arguments of cmd into args.
func unmarshalArgs(cmd task.Command, args any) error {
	if err := cmd.Args.Unmarshal(args); err != nil {
		return errors.Wrapf(validate.ErrValidation, "invalid arguments: %s", err)
	}
	return nil
}

// probe reports whether a client can connect to the broker of cfg.
func probe(ctx context.Context, ins alamos.Instrumentation, cfg clientConfig) error {
	_, disconnect, err := open(ctx, ins, paho.NewClient, newClientOptions(cfg))
	if err != nil {
		return err
	}
	disconnect()
	return nil
}

func (s *scanner) browse(ctx context.Context, args browseArgs) (browseResult, error) {
	heard, err := s.hear(ctx, args)
	if err != nil {
		return browseResult{}, err
	}
	topics := make([]browsedTopic, len(heard.topics))
	for i, t := range heard.topics {
		t.Rate = float64(t.Count-t.retained) / heard.duration.Seconds()
		topics[i] = t.browsedTopic
	}
	kept, truncated, err := driver.Fit(browseResult{}, [][]browsedTopic{topics})
	if err != nil {
		return browseResult{}, err
	}
	return browseResult{
		Topics:    kept[0],
		Truncated: heard.truncated || truncated,
	}, nil
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
	return driver.CutText(payload, browsePayloadLimit)
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
		Sample:  driver.CutText(value, browseSampleLimit),
	})
	return nil
}
