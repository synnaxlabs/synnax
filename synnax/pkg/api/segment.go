package api

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/x/telem"
)

// Segment is an API-friendly version of the segment.MD type. It is simplified for
// use purely as a data container.
type Segment struct {
	ChannelKey string          `json:"channel_key" msgpack:"channel_key"`
	Start      telem.TimeStamp `json:"start" msgpack:"start"`
	Data       []byte          `json:"data" msgpack:"data"`
}

type SegmentService struct {
	loggingProvider
	authProvider
	Internal *segment.Service
}

func NewSegmentService(p Provider) *SegmentService {
	return &SegmentService{
		Internal:        p.Config.Segment,
		authProvider:    p.auth,
		loggingProvider: p.Logging,
	}
}
