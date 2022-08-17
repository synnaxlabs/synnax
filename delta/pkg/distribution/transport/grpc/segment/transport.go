package segment

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/segment/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"google.golang.org/grpc"
)

type (
	baseWriterTransport = fgrpc.StreamTransportCore[
		writer.Request,
		*segmentv1.WriterRequest,
		writer.Response,
		*segmentv1.WriterResponse,
	]
	baseIteratorTransport = fgrpc.StreamTransportCore[
		iterator.Request,
		*segmentv1.IteratorRequest,
		iterator.Response,
		*segmentv1.IteratorResponse,
	]
)

var (
	_ segmentv1.WriterServiceServer   = (*writerTransport)(nil)
	_ writer.Transport                = (*writerTransport)(nil)
	_ segmentv1.IteratorServiceServer = (*iteratorTransport)(nil)
	_ iterator.Transport              = (*iteratorTransport)(nil)
	_ segment.Transport               = (*transport)(nil)
)

func New(pool *fgrpc.Pool) *transport {
	t := &transport{
		wt: &writerTransport{baseWriterTransport: baseWriterTransport{
			Pool:               pool,
			RequestTranslator:  writerRequestTranslator{},
			ResponseTranslator: writerResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*segmentv1.WriterRequest, *segmentv1.WriterResponse], error) {
				return segmentv1.NewWriterServiceClient(conn).Write(ctx)
			},
		}},
		it: &iteratorTransport{baseIteratorTransport: baseIteratorTransport{
			Pool:               pool,
			RequestTranslator:  iteratorRequestTranslator{},
			ResponseTranslator: iteratorResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*segmentv1.IteratorRequest, *segmentv1.IteratorResponse], error) {
				return segmentv1.NewIteratorServiceClient(conn).Iterate(ctx)
			},
		}},
	}
	return t
}

type writerTransport struct{ baseWriterTransport }

func (w *writerTransport) Write(server segmentv1.WriterService_WriteServer) error {
	return w.Handler(server.Context(), w.Server(server))
}

type iteratorTransport struct{ baseIteratorTransport }

func (t *iteratorTransport) Iterate(server segmentv1.IteratorService_IterateServer) error {
	return t.Handler(server.Context(), t.Server(server))
}

type transport struct {
	wt *writerTransport
	it *iteratorTransport
}

func (t *transport) Writer() writer.Transport { return t.wt }

func (t *transport) Iterator() iterator.Transport { return t.it }

func (t *transport) BindTo(server grpc.ServiceRegistrar) {
	segmentv1.RegisterIteratorServiceServer(server, t.it)
	segmentv1.RegisterWriterServiceServer(server, t.wt)
}
