package segment

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	sv1 "github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/segment/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
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
	return writer.Request{OpenKeys: keys, Segments: tranSegFwd(req.Segments)}, err
}

func (w writerRequestTranslator) Forward(req writer.Request) (*sv1.WriterRequest, error) {
	return &sv1.WriterRequest{OpenKeys: req.OpenKeys.Strings(), Segments: tranSegBwd(req.Segments)}, nil
}

type writerResponseTranslator struct{}

func (w writerResponseTranslator) Backward(res *sv1.WriterResponse) (writer.Response, error) {
	return writer.Response{Error: errors.New(res.Error)}, nil
}

func (w writerResponseTranslator) Forward(res writer.Response) (*sv1.WriterResponse, error) {
	return &sv1.WriterResponse{Error: res.Error.Error()}, nil
}

// |||||| ITERATOR ||||||

type iteratorRequestTranslator struct{}

func (w iteratorRequestTranslator) Backward(req *sv1.IteratorRequest) (iterator.Request, error) {
	keys, err := channel.ParseKeys(req.Keys)
	return iterator.Request{
		Command: iterator.Command(req.Command),
		Span:    telem.TimeSpan(req.Span),
		Range: telem.TimeRange{
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
			Start: int64(req.Range.Start),
			End:   int64(req.Range.End),
		},
		Stamp: int64(req.Stamp),
		Keys:  req.Keys.Strings(),
	}, nil
}

type iteratorResponseTranslator struct{}

func (w iteratorResponseTranslator) Backward(res *sv1.IteratorResponse) (iterator.Response, error) {
	return iterator.Response{Error: errors.New(res.Error), Segments: tranSegFwd(res.Segments)}, nil
}

func (w iteratorResponseTranslator) Forward(res iterator.Response) (*sv1.IteratorResponse, error) {
	return &sv1.IteratorResponse{Error: res.Error.Error(), Segments: tranSegBwd(res.Segments)}, nil
}

// |||||| SEGMENTS ||||||

func tranSegFwd(segments []*sv1.Segment) []segment.Segment {
	tSegments := make([]segment.Segment, len(segments))
	for i, seg := range segments {
		key, err := channel.ParseKey(seg.ChannelKey)
		if err != nil {
			panic(err)
		}
		tSegments[i] = segment.Segment{
			ChannelKey: key,
			Segment: cesium.Segment{
				ChannelKey: key.StorageKey(),
				Start:      telem.TimeStamp(seg.Start),
				Data:       seg.Data,
			},
		}
	}
	return tSegments
}

func tranSegBwd(segments []segment.Segment) []*sv1.Segment {
	tSegments := make([]*sv1.Segment, len(segments))
	for i, seg := range segments {
		tSegments[i] = &sv1.Segment{
			ChannelKey: seg.ChannelKey.String(),
			Start:      int64(seg.Segment.Start),
			Data:       seg.Segment.Data,
		}
	}
	return tSegments
}
