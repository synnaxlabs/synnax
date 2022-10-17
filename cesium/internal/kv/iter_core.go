package kv

import (
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

type UnaryMDIterator = gorp.KVIterator[segment.MD]

func NewMDIterator(db kv.Reader, opts kv.IteratorOptions) *UnaryMDIterator {
	return gorp.WrapKVIter[segment.MD](
		db.NewIterator(opts),
		gorp.WithoutTypePrefix(),
		gorp.WithEncoderDecoder(segmentEncoderDecoder),
	)
}

// segmentEncoderDecoder is the binary.EncoderDecoder that headers are encoded in.
// this allows us to iterate across time ranges by byte value.
var segmentEncoderDecoder = &binary.PassThroughEncoderDecoder{
	EncoderDecoder: &binary.GobEncoderDecoder{},
}
