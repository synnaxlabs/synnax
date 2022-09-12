package writer

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
)

type requestTranslator struct {
	host core.NodeID
	confluence.LinearTransform[Request, cesium.WriteRequest]
	confluence.TransientProvider
}

func newRequestTranslator(host core.NodeID, transient confluence.Inlet[error]) confluence.Segment[Request, cesium.WriteRequest] {
	rt := &requestTranslator{host: host}
	rt.Transform = rt.translate
	return confluence.InjectTransient[Request, cesium.WriteRequest](transient, rt)
}

func (rt *requestTranslator) translate(ctx context.Context, in Request) (cesium.WriteRequest, bool, error) {
	req := cesium.WriteRequest{Segments: make([]cesium.Segment, 0, len(in.Segments))}
	for _, seg := range in.Segments {
		if seg.ChannelKey.NodeID() != rt.host {
			if err := signal.SendUnderContext(ctx, rt.Transient(), unspecifiedChannelError(seg.ChannelKey)); err != nil {
				return cesium.WriteRequest{}, false, err
			}
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
	return Response(in), true, nil
}

func newResponseTranslator() *responseTranslator {
	rt := &responseTranslator{}
	rt.Transform = rt.translate
	return rt
}
