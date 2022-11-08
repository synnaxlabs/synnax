package writer

import (
	"context"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/confluence"
	"go.uber.org/zap"
)

type requestTranslator struct {
	host core.NodeID
	confluence.LinearTransform[Request, cesium.WriteRequest]
	logger *zap.Logger
}

func newRequestTranslator(host core.NodeID, logger *zap.Logger) confluence.Segment[Request, cesium.WriteRequest] {
	rt := &requestTranslator{host: host, logger: logger}
	rt.Transform = rt.translate
	return rt
}

func (rt *requestTranslator) translate(ctx context.Context, in Request) (cesium.WriteRequest, bool, error) {
	req := cesium.WriteRequest{
		Command:  cesium.WriterCommand(in.Command),
		Segments: make([]cesium.Segment, 0, len(in.Segments)),
	}
	for _, seg := range in.Segments {
		if seg.ChannelKey.NodeID() != rt.host {
			rt.logger.DPanic("received write request for channel whose lease is not the current node")
			continue
		}
		seg.Segment.ChannelKey = seg.ChannelKey.StorageKey()
		req.Segments = append(req.Segments, seg.Segment)
	}
	return req, true, nil
}

type responseTranslator struct {
	confluence.LinearTransform[cesium.WriteResponse, Response]
}

func (rt *responseTranslator) translate(_ context.Context, in cesium.WriteResponse) (Response, bool, error) {
	return Response{Command: Command(in.Command), Ack: in.Ack, Err: in.Err, SeqNum: in.SeqNum}, true, nil
}

func newResponseTranslator() *responseTranslator {
	rt := &responseTranslator{}
	rt.Transform = rt.translate
	return rt
}
