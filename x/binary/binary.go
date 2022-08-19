package binary

import (
	"bytes"
	"encoding/binary"
	"io"
)

func Write(w io.Writer, data interface{}) (err error) {
	return binary.Write(w, Encoding(), data)
}

func Read(r io.Reader, data interface{}) (err error) {
	return binary.Read(r, Encoding(), data)
}

func Marshal(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := Write(buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Encoding() binary.ByteOrder {
	return binary.BigEndian
}

func MakeCopy(bytes []byte) []byte {
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	return copied
}
