package iterator

import (
	"context"
	"github.com/synnaxlabs/cesium"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
)

func newStorageResponseTranslator(host dcore.NodeID) func(ctx context.Context, in storage.TSIteratorResponse) (Response, bool, error) {
	return func(ctx context.Context, res storage.TSIteratorResponse) (Response, bool, error) {
		return Response{
			Ack:     res.Ack,
			Variant: ResponseVariant(res.Variant),
			SeqNum:  res.SeqNum,
			NodeID:  host,
			Err:     res.Err,
			Command: Command(res.Command),
			Frame:   core.NewFrameFromStorage(res.Frame),
		}, true, nil
	}
}

func newStorageRequestTranslator() func(ctx context.Context, in Request) (storage.TSIteratorRequest, bool, error) {
	return func(ctx context.Context, req Request) (storage.TSIteratorRequest, bool, error) {
		return cesium.IteratorRequest{
			Command: cesium.IteratorCommand(req.Command),
			Span:    req.Span,
			Stamp:   req.Stamp,
			Bounds:  req.Bounds,
		}, true, nil
	}
}
