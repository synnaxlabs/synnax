package api

import (
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

type Frame struct {
	Keys []string
	telem.Frame
}

func newFrameFromDistribution(frame framer.Frame) Frame {
	return Frame{
		Keys:  frame.Keys().Strings(),
		Frame: frame.Frame,
	}
}

func toDistributionFrame(f Frame) (framer.Frame, errors.Typed) {
	keys, err := channel.ParseKeys(f.Keys)
	if err != nil {
		return framer.Frame{}, errors.Parse(err)
	}
	return framer.NewFrame(keys, f.Arrays), errors.Nil
}

type SegmentService struct {
	loggingProvider
	authProvider
	Internal *framer.Service
}

func NewSegmentService(p Provider) *SegmentService {
	return &SegmentService{
		Internal:        p.Config.Segment,
		authProvider:    p.auth,
		loggingProvider: p.Logging,
	}
}
