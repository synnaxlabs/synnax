package segment

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	sv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/segment/v1"
	"github.com/synnaxlabs/x/telem"
)

// |||||| WRITER ||||||

var (
	_ fgrpc.Translator[writer.Request, *sv1.WriterRequest]       = (*writerRequestTranslator)(nil)
	_ fgrpc.Translator[writer.Response, *sv1.WriterResponse]     = (*writerResponseTranslator)(nil)
	_ fgrpc.Translator[iterator.Request, *sv1.IteratorRequest]   = (*iteratorRequestTranslator)(nil)
	_ fgrpc.Translator[iterator.Response, *sv1.IteratorResponse] = (*iteratorResponseTranslator)(nil)
)

type writerRequestTranslator struct{}

func (w writerRequestTranslator) Backward(req *sv1.WriterRequest) (writer.Request, error) {
	keys, err := channel.ParseKeys(req.OpenKeys)
	return writer.Request{
		Command: writer.Command(req.Command),
		Keys:    keys,
		Start:   telem.TimeStamp(req.Start),
		Frame:   tranFrameFwd(req.Frame),
	}, err
}

func (w writerRequestTranslator) Forward(req writer.Request) (*sv1.WriterRequest, error) {
	return &sv1.WriterRequest{
		Command:  int32(req.Command),
		OpenKeys: req.Keys.Strings(),
		Start:    int64(req.Start),
		Frame:    tranFrmBwd(req.Frame),
	}, nil
}

type writerResponseTranslator struct{}

func (w writerResponseTranslator) Backward(res *sv1.WriterResponse) (writer.Response, error) {
	return writer.Response{
		Command: writer.Command(res.Command),
		SeqNum:  int(res.Counter),
		Ack:     res.Ack,
		Err:     fgrpc.DecodeError(res.Error),
	}, nil
}

func (w writerResponseTranslator) Forward(res writer.Response) (*sv1.WriterResponse, error) {
	return &sv1.WriterResponse{
		Command: int32(res.Command),
		Counter: int32(res.SeqNum),
		Ack:     res.Ack,
		Error:   fgrpc.EncodeError(res.Err),
	}, nil
}

// |||||| ITERATOR ||||||

type iteratorRequestTranslator struct{}

func (w iteratorRequestTranslator) Backward(req *sv1.IteratorRequest) (iterator.Request, error) {
	keys, err := channel.ParseKeys(req.Keys)
	return iterator.Request{
		Command: iterator.Command(req.Command),
		Span:    telem.TimeSpan(req.Span),
		Bounds: telem.TimeRange{
			Start: telem.TimeStamp(req.Range.Start),
			End:   telem.TimeStamp(req.Range.End),
		},
		Stamp: telem.TimeStamp(req.Stamp),
		Keys:  keys,
	}, err
}

func (w iteratorRequestTranslator) Forward(req iterator.Request) (*sv1.IteratorRequest, error) {
	return &sv1.IteratorRequest{
		Command: int32(req.Command),
		Span:    int64(req.Span),
		Range: &sv1.TimeRange{
			Start: int64(req.Bounds.Start),
			End:   int64(req.Bounds.End),
		},
		Stamp: int64(req.Stamp),
		Keys:  req.Keys.Strings(),
	}, nil
}

type iteratorResponseTranslator struct{}

func (w iteratorResponseTranslator) Backward(res *sv1.IteratorResponse) (iterator.Response, error) {
	return iterator.Response{
		Variant: iterator.ResponseVariant(res.Variant),
		NodeID:  distribcore.NodeID(res.NodeId),
		Ack:     res.Ack,
		SeqNum:  int(res.Counter),
		Command: iterator.Command(res.Command),
		Err:     fgrpc.DecodeError(res.Error),
		Frame:   tranFrameFwd(res.Frame),
	}, nil
}

func (w iteratorResponseTranslator) Forward(res iterator.Response) (*sv1.IteratorResponse, error) {
	return &sv1.IteratorResponse{
		Variant: int32(res.Variant),
		NodeId:  int32(res.NodeID),
		Ack:     res.Ack,
		Counter: int32(res.SeqNum),
		Command: int32(res.Command),
		Error:   fgrpc.EncodeError(res.Err),
		Frame:   tranFrmBwd(res.Frame),
	}, nil
}

// |||||| SEGMENTS ||||||

func tranFrameFwd(frame *sv1.Frame) framer.Frame {
	keys := lo.Must(channel.ParseKeys(frame.Keys))
	arrays := tranArrayFwd(frame.Arrays)
	return framer.NewFrame(keys, arrays)
}

func tranFrmBwd(frame framer.Frame) *sv1.Frame {
	return &sv1.Frame{
		Keys:   frame.Keys().Strings(),
		Arrays: tranArrBwd(frame.Arrays),
	}
}

func tranArrayFwd(arrays []*sv1.Array) []telem.Array {
	tArrays := make([]telem.Array, len(arrays))
	for i, arr := range arrays {
		tArrays[i] = telem.Array{
			DataType: telem.DataType(arr.DataType),
			Range: telem.TimeRange{
				Start: telem.TimeStamp(arr.Range.Start),
				End:   telem.TimeStamp(arr.Range.End),
			},
			Data: arr.Data,
		}
	}
	return tArrays
}

func tranArrBwd(arrays []telem.Array) []*sv1.Array {
	tArrays := make([]*sv1.Array, len(arrays))
	for i, arr := range arrays {
		tArrays[i] = &sv1.Array{
			DataType: string(arr.DataType),
			Range: &sv1.TimeRange{
				Start: int64(arr.Range.Start),
				End:   int64(arr.Range.End),
			},
			Data: arr.Data,
		}
	}
	return tArrays
}
