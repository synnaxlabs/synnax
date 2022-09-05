package seg

import (
	"bytes"
	"encoding/binary"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/telem"
	"math/rand"
)

type DataFactory interface {
	Generate(n int) []byte
}

func New(c cesium.Channel, fac DataFactory, start telem.TimeStamp, span telem.TimeSpan) cesium.Segment {
	return cesium.Segment{
		ChannelKey: c.Key,
		Data:       generateSpan(c, fac, span),
		Start:      start,
	}
}

func NewSet(c cesium.Channel, fac DataFactory, start telem.TimeStamp, span telem.TimeSpan, n int) []cesium.Segment {
	s := make([]cesium.Segment, n)
	for i := 0; i < n; i++ {
		s[i] = New(c, fac, start, span)
		start = start.Add(span)
	}
	return s
}

func GenSlice[T any](n int, fac func(int) T) []T {
	s := make([]T, n)
	for i := 0; i < n; i++ {
		s[i] = fac(i)
	}
	return s

}

func sequentialFloat64Slice(n int) []float64 {
	return GenSlice[float64](n, func(i int) float64 { return float64(i) })
}

func randomFloat64(n int) []float64 {
	return GenSlice[float64](n, func(i int) float64 { return rand.Float64() })
}

type SequentialFloat64Factory struct {
	Cache  bool
	values []byte
}

func (s *SequentialFloat64Factory) Generate(n int) []byte {
	if s.Cache && len(s.values) >= n {
		if len(s.values) == n {
			return s.values
		}
		return s.values[:n]
	}
	b := writeBytes(sequentialFloat64Slice(n))
	if s.Cache {
		s.values = b
	}
	return b
}

type RandomFloat64Factory struct {
	Cache  bool
	values []byte
}

func (s *RandomFloat64Factory) Generate(n int) []byte {
	if s.Cache && len(s.values) >= n*8 {
		return s.values[:n*8]
	}
	b := writeBytes(randomFloat64(n))
	if s.Cache {
		s.values = b
	}
	return b
}

func writeBytes(data interface{}) []byte {
	b := bytes.NewBuffer(nil)
	err := binary.Write(b, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func generateSpan(c cesium.Channel, fac DataFactory, span telem.TimeSpan) []byte {
	sc := c.Rate.SampleCount(span)
	return fac.Generate(sc)
}

type MultiSequentialFactory struct {
	factories []SequentialFactory
}

func (m *MultiSequentialFactory) Next() (s []cesium.Segment) {
	for _, f := range m.factories {
		s = append(s, f.Next()...)
	}
	return s
}

func (m *MultiSequentialFactory) NextN(n int) (s []cesium.Segment) {
	for _, f := range m.factories {
		s = append(s, f.NextN(n)...)
	}
	return s
}

type sequentialFactory struct {
	FirstTS telem.TimeStamp
	PrevTS  telem.TimeStamp
	Factory DataFactory
	Span    telem.TimeSpan
	Channel cesium.Channel
}

type SequentialFactory interface {
	Next() []cesium.Segment
	NextN(n int) []cesium.Segment
}

func NewSequentialFactory(fac DataFactory, span telem.TimeSpan, c ...cesium.Channel) SequentialFactory {
	if len(c) == 0 {
		panic("no channels provided to sequential factory")
	}
	if len(c) == 1 {
		return newSingleSequentialFactory(fac, span, c[0])

	}
	multi := &MultiSequentialFactory{factories: make([]SequentialFactory, len(c))}
	for i, ch := range c {
		multi.factories[i] = newSingleSequentialFactory(fac, span, ch)
	}
	return multi
}

func newSingleSequentialFactory(fac DataFactory, span telem.TimeSpan, c cesium.Channel) SequentialFactory {
	return &sequentialFactory{
		FirstTS: telem.TimeStampMin,
		PrevTS:  telem.TimeStampMin,
		Factory: fac,
		Span:    span,
		Channel: c,
	}
}

func (sf *sequentialFactory) Next() []cesium.Segment {
	s := New(sf.Channel, sf.Factory, sf.PrevTS, sf.Span)
	sf.PrevTS = sf.PrevTS.Add(sf.Span)
	return []cesium.Segment{s}
}

func (sf *sequentialFactory) NextN(n int) []cesium.Segment {
	s := NewSet(sf.Channel, sf.Factory, sf.PrevTS, sf.Span, n)
	sf.PrevTS = s[n-1].Start.Add(sf.Span)
	return s
}
