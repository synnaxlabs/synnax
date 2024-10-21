package codec

import (
	"encoding/binary"
	"io"
)

type Reader struct {
	ByteOrder binary.ByteOrder
	Buffer    io.Reader
}

func (b *Reader) Uint16() (v uint16, err error) {
	err = binary.Read(b.Buffer, b.ByteOrder, &v)
	return v, err
}

func (b *Reader) Uint32() (v uint32, err error) {
	err = binary.Read(b.Buffer, b.ByteOrder, &v)
	return v, err
}

func (b *Reader) Uint64() (v uint64, err error) {
	err = binary.Read(b.Buffer, b.ByteOrder, &v)
	return v, err
}

func (b *Reader) Int16() (v int16, err error) {
	err = binary.Read(b.Buffer, b.ByteOrder, &v)
	return v, err
}

func (b *Reader) Int32() (v int32, err error) {
	err = binary.Read(b.Buffer, b.ByteOrder, &v)
	return v, err
}
