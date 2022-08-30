package binary

import (
	"bytes"
	"encoding/binary"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"math"
)

type UnaryParser struct {
	ByteOrder binary.ByteOrder
}

func (p UnaryParser) Uint16(b []byte) uint16 { return p.ByteOrder.Uint16(b) }

func (p UnaryParser) Uint32(b []byte) uint32 { return p.ByteOrder.Uint32(b) }

func (p UnaryParser) Uint64(b []byte) uint64 { return p.ByteOrder.Uint64(b) }

func (p UnaryParser) Byte(b []byte) byte { return b[0] }

func (p UnaryParser) Int8(b []byte) int8 { return int8(b[0]) }

func (p UnaryParser) Int16(b []byte) int16 { return int16(p.Uint16(b)) }

func (p UnaryParser) Int32(b []byte) int32 { return int32(p.Uint32(b)) }

func (p UnaryParser) Int64(b []byte) int64 { return int64(p.Uint64(b)) }

func (p UnaryParser) Float32(b []byte) float32 { return math.Float32frombits(p.Uint32(b)) }

func (p UnaryParser) Float64(b []byte) float64 { return math.Float64frombits(p.Uint64(b)) }

type BufferParser struct {
	UnaryParser
}

func NewBufferParser(b binary.ByteOrder) BufferParser {
	return BufferParser{UnaryParser{b}}
}

func (p BufferParser) Uint16(b bytes.Buffer) ([]uint16, error) {
	return parse[uint16](b, 2, p.UnaryParser.Uint16)
}

func (p BufferParser) Uint32(b bytes.Buffer) ([]uint32, error) {
	return parse[uint32](b, 4, p.UnaryParser.Uint32)
}

func (p BufferParser) Uint64(b bytes.Buffer) ([]uint64, error) {
	return parse[uint64](b, 8, p.UnaryParser.Uint64)
}

func (p BufferParser) Byte(b bytes.Buffer) ([]byte, error) {
	return parse[byte](b, 1, p.UnaryParser.Byte)
}

func (p BufferParser) Int8(b bytes.Buffer) ([]int8, error) {
	return parse[int8](b, 1, p.UnaryParser.Int8)
}

func (p BufferParser) Int16(b bytes.Buffer) ([]int16, error) {
	return parse[int16](b, 2, p.UnaryParser.Int16)
}

func (p BufferParser) Int32(b bytes.Buffer) ([]int32, error) {
	return parse[int32](b, 4, p.UnaryParser.Int32)
}

func (p BufferParser) Int64(b bytes.Buffer) ([]int64, error) {
	return parse[int64](b, 8, p.UnaryParser.Int64)
}

func (p BufferParser) Float32(b bytes.Buffer) ([]float32, error) {
	return parse[float32](b, 4, p.UnaryParser.Float32)
}

func (p BufferParser) Float64(b bytes.Buffer) ([]float64, error) {
	return parse[float64](b, 8, p.UnaryParser.Float64)
}

var InvalidBufferLength = errors.New("invalid buffer length for given data type")

func parse[V any](b bytes.Buffer, n int, parse func(b []byte) V) ([]V, error) {
	_len, err := calculateLength(b, n)
	if err != nil {
		return nil, err
	}
	out := make([]V, _len)
	for i := 0; i < _len; i++ {
		out[i] = parse(b.Next(n))
	}
	return out, nil
}

func checkValidLen(_len int, n int) error {
	return lo.Ternary(_len%n != 0, InvalidBufferLength, nil)
}

func calculateLength(b bytes.Buffer, n int) (int, error) {
	_len := b.Len()
	return _len / n, checkValidLen(_len, n)
}
