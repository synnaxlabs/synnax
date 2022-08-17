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

type Encoder interface {
	Encode(value interface{}) ([]byte, error)
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

type GobEncoderDecoder struct{}

func (e *GobEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	//if bv, ok := value.([]byte); ok {
	//	return bv, nil
	//}
	var buff bytes.Buffer
	err := gob.NewEncoder(&buff).Encode(value)
	b := buff.Bytes()
	return b, err
}

func (e *GobEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := e.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

func (e *GobEncoderDecoder) Decode(data []byte, value interface{}) error {
	return e.DecodeStream(bytes.NewReader(data), value)
}

func (e *GobEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := e.Decode(data, value); err != nil {
		panic(err)
	}
}

func (e *GobEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return gob.NewDecoder(r).Decode(value)
}

type JSONEncoderDecoder struct{}

func (j *JSONEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (j *JSONEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := j.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

func (j *JSONEncoderDecoder) Decode(data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
}

func (j *JSONEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := j.Decode(data, value); err != nil {
		panic(err)
	}
}

func (j *JSONEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return json.NewDecoder(r).Decode(value)
}

type MsgPackEncoderDecoder struct{}

func (m *MsgPackEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	return msgpack.Marshal(value)
}

func (m *MsgPackEncoderDecoder) EncodeStatic(value interface{}) []byte {
	b, err := m.Encode(value)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MsgPackEncoderDecoder) Decode(data []byte, value interface{}) error {
	return m.DecodeStream(bytes.NewReader(data), value)
}

func (m *MsgPackEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := m.Decode(data, value); err != nil {
		panic(err)
	}
}

func (m *MsgPackEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	return msgpack.NewDecoder(r).Decode(value)
}

// ByteCheckableEncoderDecoder wraps an EncoderDecoder and checks for values
// that are already encoded ([]byte) and returns them as is.
type ByteCheckableEncoderDecoder struct {
	EncoderDecoder
}

func (b *ByteCheckableEncoderDecoder) Encode(value interface{}) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return b.EncoderDecoder.Encode(value)
}

func (b *ByteCheckableEncoderDecoder) EncodeStatic(value interface{}) []byte {
	if bv, ok := value.([]byte); ok {
		return bv
	}
	return b.EncoderDecoder.EncodeStatic(value)
}

func (b *ByteCheckableEncoderDecoder) Decode(data []byte, value interface{}) error {
	return b.DecodeStream(bytes.NewReader(data), value)
}

func (b *ByteCheckableEncoderDecoder) DecodeStatic(data []byte, value interface{}) {
	if err := b.Decode(data, value); err != nil {
		panic(err)
	}
}

func (b *ByteCheckableEncoderDecoder) DecodeStream(r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return b.EncoderDecoder.DecodeStream(r, value)
}
