package binary

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/vmihailenco/msgpack/v5"
	"io"
)

// EncoderDecoder is an interface that encodes and decodes values.
type EncoderDecoder interface {
	Encoder
	Decoder
}

// Encoder encodes values into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along
	// with any errors encountered.
	Encode(value interface{}) ([]byte, error)
}

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(data []byte, value interface{}) error
	// DecodeStream decodes data from the given reader into a pointer value.;
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

// Decode implements the Decoder interface.
func (e *GobEncoderDecoder) Decode(data []byte, value interface{}) error {
	return e.DecodeStream(bytes.NewReader(data), value)
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

// Decode implements the Decoder interface.
func (j *JSONEncoderDecoder) Decode(data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
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

// Decode implements the Decoder interface.
func (m *MsgPackEncoderDecoder) Decode(data []byte, value interface{}) error {
	return m.DecodeStream(bytes.NewReader(data), value)
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

// Decode implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) Decode(data []byte, value interface{}) error {
	return enc.DecodeStream(bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.EncoderDecoder.DecodeStream(r, value)
}
