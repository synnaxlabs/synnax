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
	_ Codec = (*GobEncoderDecoder)(nil)
	_ Codec = (*JSONEncoderDecoder)(nil)
	_ Codec = (*MsgPackEncoderDecoder)(nil)
)

// GobEncoderDecoder is a gob implementation of the Codec interface.
type GobEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (e *GobEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
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
func (e *GobEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := e.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (e *GobEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := gob.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// JSONEncoderDecoder is a JSON implementation of Codec.
type JSONEncoderDecoder struct {
	// Pretty indicates whether the JSON should be pretty printed.
	Pretty bool
}

// Encode implements the Encoder interface.
func (j *JSONEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
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
func (j *JSONEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := json.Unmarshal(data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (j *JSONEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := json.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// MsgPackEncoderDecoder is a msgpack implementation of Codec.
type MsgPackEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (m *MsgPackEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, nil
}

// Decode implements the Decoder interface.
func (m *MsgPackEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	err := m.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	err := msgpack.NewDecoder(r).Decode(value)
	if err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// PassThroughEncoderDecoder wraps a Codec and checks for values
// that are already encoded ([]byte) and returns them as is.
type PassThroughEncoderDecoder struct{ Codec }

// Encode implements the Encoder interface.
func (enc *PassThroughEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return enc.Codec.Encode(ctx, value)
}

// Decode implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	return enc.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.Codec.DecodeStream(ctx, r, value)
}

// TracingEncoderDecoder wraps a Codec and traces the encoding and decoding
// operations.
type TracingEncoderDecoder struct {
	alamos.Instrumentation
	Level alamos.Environment
	Codec
}

// Encode implements the Encoder interface.
func (enc *TracingEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	ctx, span := enc.T.Trace(ctx, "encode", enc.Level)
	b, err := enc.Codec.Encode(ctx, value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, span.EndWith(err)
}

// Decode implements the Decoder interface.
func (enc *TracingEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	ctx, span := enc.T.Trace(ctx, "decode", enc.Level)
	err := enc.Codec.Decode(ctx, data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// DecodeStream implements the Decoder interface.
func (enc *TracingEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
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
