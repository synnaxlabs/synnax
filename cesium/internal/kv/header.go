package kv

import (
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
)

type Header struct {
	db *gorp.DB
}

// headerEncoderDecoder is the binary.EncoderDecoder that headers are encoded in.
// this allows us to iterate across time ranges by byte value.
var headerEncoderDecoder = &binary.PassThroughEncoderDecoder{
	EncoderDecoder: &binary.GobEncoderDecoder{},
}

func NewHeader(kve kv.DB) *Header {
	return &Header{
		db: gorp.Wrap(
			kve,
			gorp.WithoutTypePrefix(),
			gorp.WithEncoderDecoder(headerEncoderDecoder),
		),
	}
}

func (s *Header) Set(header segment.Header) error {
	return s.SetMultiple([]segment.Header{header})
}

func (s *Header) SetMultiple(headers []segment.Header) error {
	return gorp.NewCreate[[]byte, segment.Header]().Entries(&headers).Exec(s.db)
}
