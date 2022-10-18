package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

type coreMDIterator = gorp.KVIterator[core.SegmentMD]

func newCoreMDIterator(db kv.Reader, opts kv.IteratorOptions) *coreMDIterator {
	return gorp.WrapKVIter[core.SegmentMD](
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
