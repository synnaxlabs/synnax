package binary

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"io"
	"reflect"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// sugarEncodingErr adds additional context to encoding errors.
func sugarEncodingErr(value interface{}, err error) error {
	val := reflect.ValueOf(value)
	return errors.Wrapf(err, "failed to encode value: kind=%s, type=%s, value=%+v", val.Kind(), val.Type(), value)
}

// sugarDecodingErr adds additional context to decoding errors.
func sugarDecodingErr(data []byte, value interface{}, err error) error {
	val := reflect.ValueOf(value)
	return errors.Wrapf(err, "failed to decode into value: kind=%s, type=%s, data=%x", val.Kind(), val.Type(), data)
}

// Codec is an interface that encodes and decodes values.
type Codec interface {
	Encoder
	Decoder
}

// Encoder encodes values into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along
	// with any errors encountered.
	Encode(ctx context.Context, value interface{}) ([]byte, error)
}

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(ctx context.Context, data []byte, value interface{}) error
	// DecodeStream decodes data from the given reader into a pointer value.;
	DecodeStream(ctx context.Context, r io.Reader, value interface{}) error
}

var (
	_ Codec = (*GobCodec)(nil)
	_ Codec = (*JSONCodec)(nil)
	_ Codec = (*MsgPackCodec)(nil)
)

// GobCodec is a gob implementation of the Codec interface.
type GobCodec struct{}

// Encode implements the Encoder interface.
func (e *GobCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	var (
		buff bytes.Buffer
		err  = gob.NewEncoder(&buff).Encode(value)
		b    = buff.Bytes()
	)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, nil
}

// Decode implements the Decoder interface.
func (e *GobCodec) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := e.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (e *GobCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := gob.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// JSONCodec is a JSON implementation of Codec.
type JSONCodec struct {
	// Pretty indicates whether the JSON should be pretty printed.
	Pretty bool
}

// Encode implements the Encoder interface.
func (j *JSONCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	var b []byte
	var err error
	if j.Pretty {
		b, err = json.MarshalIndent(value, "", "  ")
	} else {
		b, err = json.Marshal(value)
	}
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, nil
}

// Decode implements the Decoder interface.
func (j *JSONCodec) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := json.Unmarshal(data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (j *JSONCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := json.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// MsgPackCodec is a msgpack implementation of Codec.
type MsgPackCodec struct{}

// Encode implements the Encoder interface.
func (m *MsgPackCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, nil
}

// Decode implements the Decoder interface.
func (m *MsgPackCodec) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := m.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := msgpack.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// PassThroughCodec wraps a Codec and checks for values
// that are already encoded ([]byte) and returns them as is.
type PassThroughCodec struct{ Codec }

// Encode implements the Encoder interface.
func (enc *PassThroughCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return enc.Codec.Encode(ctx, value)
}

// Decode implements the Decoder interface.
func (enc *PassThroughCodec) Decode(ctx context.Context, data []byte, value interface{}) error {
	return enc.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.Codec.DecodeStream(ctx, r, value)
}

// TracingCodec wraps a Codec and traces the encoding and decoding
// operations.
type TracingCodec struct {
	alamos.Instrumentation
	Level alamos.Environment
	Codec
}

// Encode implements the Encoder interface.
func (enc *TracingCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	ctx, span := enc.T.Trace(ctx, "encode", enc.Level)
	b, err := enc.Codec.Encode(ctx, value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, span.EndWith(err)
}

// Decode implements the Decoder interface.
func (enc *TracingCodec) Decode(ctx context.Context, data []byte, value interface{}) error {
	ctx, span := enc.T.Trace(ctx, "decode", enc.Level)
	err := enc.Codec.Decode(ctx, data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// DecodeStream implements the Decoder interface.
func (enc *TracingCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	ctx, span := enc.T.Trace(ctx, "decode_stream", enc.Level)
	err := enc.Codec.DecodeStream(ctx, r, value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

func UnmarshalStringInt64(b []byte) (n int64, err error) {
	if err = json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err = json.Unmarshal(b, &str); err != nil {
		return n, err
	}
	n, err = strconv.ParseInt(str, 10, 64)
	return n, err
}

func UnmarshalStringUint64(b []byte) (n uint64, err error) {
	if err = json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err = json.Unmarshal(b, &str); err != nil {
		return n, err
	}
	n, err = strconv.ParseUint(str, 10, 64)
	return n, err
}

// FallbackCodec wraps a set of Codecs. When the first Codec in the chain fails to
// encode/decode a value, it falls back to the next Codec in the chain.
type FallbackCodec struct {
	// Codecs is the list of codecs to fallback on.
	Codecs []Codec
	// FallbackOnEncode indicates whether to fallback on encoding.
	FallbackOnEncode bool
	// FallbackOnDecode indicates whether to fallback on decoding.
	FallbackOnDecode bool
}

var _ Codec = (*FallbackCodec)(nil)

// Encode implements the Encoder interface.
func (f *FallbackCodec) Encode(ctx context.Context, value interface{}) (b []byte, err error) {
	if len(f.Codecs) == 0 {
		panic("[binary] - no codecs provided to FallbackCodec")
	}
	if !f.FallbackOnEncode {
		return f.Codecs[0].Encode(ctx, value)
	}
	if f.FallbackOnEncode {
		for _, c := range f.Codecs {
			b, err = c.Encode(ctx, value)
			if err == nil {
				return
			}
		}
	}
	return
}

// Decode implements the Decoder interface.
func (f *FallbackCodec) Decode(ctx context.Context, data []byte, value interface{}) (err error) {
	if len(f.Codecs) == 0 {
		panic("[binary] - no codecs provided to FallbackCodec")
	}
	if !f.FallbackOnDecode {
		return f.Codecs[0].Decode(ctx, data, value)
	}
	if f.FallbackOnDecode {
		for _, c := range f.Codecs {
			err = c.Decode(ctx, data, value)
			if err == nil {
				return
			}
		}
	}
	return
}

// DecodeStream implements the Decoder interface.
func (f *FallbackCodec) DecodeStream(ctx context.Context, r io.Reader, value interface{}) (err error) {
	if len(f.Codecs) == 0 {
		panic("[binary] - no codecs provided to FallbackCodec")
	}
	if !f.FallbackOnDecode {
		return f.Codecs[0].DecodeStream(ctx, r, value)
	}
	if f.FallbackOnDecode {
		for _, c := range f.Codecs {
			err = c.DecodeStream(ctx, r, value)
			if err == nil {
				return
			}
		}
	}
	return
}
