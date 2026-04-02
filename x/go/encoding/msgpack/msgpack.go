// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package msgpack

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"strconv"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// Codec is a msgpack implementation of encoding.Codec.
var Codec = &codec{}

type codec struct{}

// ContentType implements http.Codec to return the http content type for the codec.
func (c *codec) ContentType() string { return "application/msgpack" }

// Encode implements the encoding.Encoder interface.
func (c *codec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	return b, encoding.SugarEncodingErr(value, err)
}

// Decode implements the encoding.Decoder interface.
func (c *codec) Decode(ctx context.Context, data []byte, value any) error {
	err := c.DecodeStream(ctx, bytes.NewReader(data), value)
	return encoding.SugarDecodingErr(data, value, err)
}

// DecodeStream implements the encoding.Decoder interface.
func (c *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := msgpack.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the encoding.Encoder interface.
func (c *codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return encoding.SugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return encoding.SugarEncodingErr(value, err)
}

// EncodedJSON is a map[string]any that handles backwards-compatible msgpack
// decoding. When existing data was stored as a JSON string (the old format), it
// unmarshals the string into a map. When new data arrives as a map, it uses it directly.
type EncodedJSON map[string]any

func (e *EncodedJSON) DecodeMsgpack(dec *msgpack.Decoder) error {
	v, err := dec.DecodeInterface()
	if err != nil {
		return err
	}
	if v == nil {
		*e = nil
		return nil
	}
	switch val := v.(type) {
	case string:
		m := make(map[string]any)
		if len(val) != 0 {
			if err = json.Unmarshal([]byte(val), &m); err != nil {
				return errors.Wrapf(err, "failed to unmarshal JSON string into EncodedJSON")
			}
		}
		*e = m
	case map[string]any:
		*e = val
	case map[any]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			ks, ok := k.(string)
			if !ok {
				return errors.Newf("EncodedJSON: non-string key %T in map", k)
			}
			m[ks] = v
		}
		*e = m
	default:
		return errors.Newf("EncodedJSON: unsupported type %T", v)
	}
	return nil
}

// Unmarshal decodes the map into the provided struct using JSON marshal/unmarshal.
func (e EncodedJSON) Unmarshal(into any) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, into)
}

// UnmarshalUint64 decodes a msgpack value into a uint64, handling type coercion
// from various numeric types, floats, and strings.
func UnmarshalUint64(dec *msgpack.Decoder) (uint64, error) {
	v, err := dec.DecodeInterface()
	if err != nil {
		return 0, err
	}
	switch val := v.(type) {
	case uint64:
		return val, nil
	case uint32:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case int64:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case int32:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case int16:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case int8:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case int:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case float64:
		if val < 0 {
			return 0, errors.Newf("negative value %f cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case float32:
		if val < 0 {
			return 0, errors.Newf("negative value %f cannot be converted to uint64", val)
		}
		return uint64(val), nil
	case string:
		return strconv.ParseUint(val, 10, 64)
	default:
		return 0, errors.Newf("cannot unmarshal %T into uint64", v)
	}
}

// UnmarshalUint32 decodes a msgpack value into a uint32, handling type coercion
// from various numeric types, floats, and strings.
func UnmarshalUint32(dec *msgpack.Decoder) (uint32, error) {
	v, err := dec.DecodeInterface()
	if err != nil {
		return 0, err
	}
	switch val := v.(type) {
	case uint64:
		if val > math.MaxUint32 {
			return 0, errors.Newf("value %d exceeds uint32 max", val)
		}
		return uint32(val), nil
	case uint32:
		return val, nil
	case uint16:
		return uint32(val), nil
	case uint8:
		return uint32(val), nil
	case int64:
		if val < 0 || val > math.MaxUint32 {
			return 0, errors.Newf("value %d out of uint32 range", val)
		}
		return uint32(val), nil
	case int32:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint32", val)
		}
		return uint32(val), nil
	case int16:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint32", val)
		}
		return uint32(val), nil
	case int8:
		if val < 0 {
			return 0, errors.Newf("negative value %d cannot be converted to uint32", val)
		}
		return uint32(val), nil
	case int:
		if val < 0 || val > math.MaxUint32 {
			return 0, errors.Newf("value %d out of uint32 range", val)
		}
		return uint32(val), nil
	case float64:
		if val < 0 || val > math.MaxUint32 {
			return 0, errors.Newf("value %f out of uint32 range", val)
		}
		return uint32(val), nil
	case float32:
		if val < 0 || val > math.MaxUint32 {
			return 0, errors.Newf("value %f out of uint32 range", val)
		}
		return uint32(val), nil
	case string:
		n, err := strconv.ParseUint(val, 10, 32)
		return uint32(n), err
	default:
		return 0, errors.Newf("cannot unmarshal %T into uint32", v)
	}
}
