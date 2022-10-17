package kv

import (
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	*gorp.KVWriter[[]byte, segment.MD]
}
