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
	return cesium.WriteRequest{
		Command: cesium.WriterCommand(in.Command),
		Frame:   in.Frame.StorageFrame(),
	}, true, nil
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
