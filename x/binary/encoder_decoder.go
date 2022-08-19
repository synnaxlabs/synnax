package binary

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/vmihailenco/msgpack/v5"
	"io"
)

type EncoderDecoder interface {
	Encoder
	Decoder
}

// Encoder is an entity that can encode a value into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along
	// with any errors encountered.
	Encode(value interface{}) ([]byte, error)
	// EncodeStatic encodes the value into binary. It panics if any errors are
	// encountered. This is useful for situations where the encoded value is
	// 
	EncodeStatic(value interface{}) []byte
}

type Decoder interface {
	Decode(data []byte, value interface{}) error
	DecodeStatic(data []byte, value interface{})
	DecodeStream(r io.Reader, value interface{}) error
}

var (
	_ EncoderDecoder = (*GobEncoderDecoder)(nil)
	_ EncoderDecoder = (*JSONEncoderDecoder)(nil)
	_ EncoderDecoder = (*MsgPackEncoderDecoder)(nil)
)

// GobEncoderDecoder is a gob implementation of the EncoderDecoder interface.
type GobEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (e *GobEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	var buff bytes.Buffer
	err := gob.NewEncoder(&buff).Encode(value)
	b := buff.Bytes()
	return b, err
}

// EncodeStatic implements the Encoder interface.
func (e *GobEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := e.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements the Decoder interface.
func (e *GobEncoderDecoder) Decode(data []byte, value interface{}) error {
	return e.DecodeStream(bytes.NewReader(data), value)
}

// DecodeStatic implements the Decoder interface.
func (e *GobEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := e.Decode(data, value); err != nil {
		panic(err)
	}
}

// DecodeStream implements the Decoder interface.
func (e *GobEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return gob.NewDecoder(r).Decode(value)
}

// JSONEncoderDecoder is a JSON implementation of EncoderDecoder.
type JSONEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (j *JSONEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// EncodeStatic implements the Encoder interface.
func (j *JSONEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := j.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements the Decoder interface.
func (j *JSONEncoderDecoder) Decode(data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
}

// DecodeStatic implements the Decoder interface.
func (j *JSONEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := j.Decode(data, value); err != nil {
		panic(err)
	}
}

// DecodeStream implements the Decoder interface.
func (j *JSONEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return json.NewDecoder(r).Decode(value)
}

// MsgPackEncoderDecoder is a msgpack implementation of EncoderDecoder.
type MsgPackEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (m *MsgPackEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	return msgpack.Marshal(value)
}

// EncodeStatic implements the Encoder interface.
func (m *MsgPackEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := m.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements the Decoder interface.
func (m *MsgPackEncoderDecoder) Decode(data []byte, value interface{}) error {
	return m.DecodeStream(bytes.NewReader(data), value)
}

// DecodeStatic implements the Decoder interface.
func (m *MsgPackEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := m.Decode(data, value); err != nil {
		panic(err)
	}
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return msgpack.NewDecoder(r).Decode(value)
}

// PassThroughEncoderDecoder wraps an EncoderDecoder and checks for values
// that are already encoded ([]byte) and returns them as is.
type PassThroughEncoderDecoder struct {
	EncoderDecoder
}

// Encode implements the Encoder interface.
func (enc *PassThroughEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return enc.EncoderDecoder.Encode(value)
}

// EncodeStatic implements the Encoder interface.
func (enc *PassThroughEncoderDecoder) EncodeStatic(value interface{}) []byte {
	if bv, ok := value.([]byte); ok {
		return bv
	}
	return enc.EncoderDecoder.EncodeStatic(value)
}

// Decode implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) Decode(data []byte, value interface{}) error {
	return enc.DecodeStream(bytes.NewReader(data), value)
}

// DecodeStatic implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := enc.Decode(data, value); err != nil {
		panic(err)
	}
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.EncoderDecoder.DecodeStream(r, value)
}
