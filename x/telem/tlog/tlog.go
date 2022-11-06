package tlog

import (
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

func TimeRange(key string, tr telem.TimeRange) zap.Field {
	return zap.Stringer(key, tr)
}
