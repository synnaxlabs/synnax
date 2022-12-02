package writer

import (
	"context"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
)

func newRequestTranslator() func(ctx context.Context, in Request) (cesium.WriteRequest, bool, error) {
	return func(ctx context.Context, in Request) (cesium.WriteRequest, bool, error) {
		return cesium.WriteRequest{
			Command: cesium.WriterCommand(in.Command), Frame: in.Frame.ToStorage(),
		}, true, nil
	}
}

func newResponseTranslator(host core.NodeID) func(ctx context.Context, in cesium.WriteResponse) (Response, bool, error) {
	return func(ctx context.Context, in cesium.WriteResponse) (Response, bool, error) {
		return Response{
			Command: Command(in.Command),
			Ack:     in.Ack,
			Err:     in.Err,
			SeqNum:  in.SeqNum,
			NodeID:  host,
		}, true, nil
	}
}
