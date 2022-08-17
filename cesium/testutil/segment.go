package testutil

import (
	"bytes"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/binary"
	"math/rand"
)

func RandFloat64Slice(n int) []float64 {
	s := make([]float64, n)
	for i := 0; i < n; i++ {
		s[i] = rand.Float64()
	}
	return s
}

func WriteFloat64Slice(s []float64) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)*8))
	buf.Reset()
	err := binary.Write(buf, s)
	return buf, err
}

func RandomFloat64Bytes(n int) []byte {
	s := RandFloat64Slice(n)
	buf, err := WriteFloat64Slice(s)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func NewFloat64Segment(cpk cesium.ChannelKey, factory func(n int)) cesium.Segment {
	return cesium.Segment{}
}

func RandomFloat64Segment(n int) []byte {
	return RandomFloat64Bytes(n)
}
