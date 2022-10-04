package segment

import (
	"context"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/segment/v1"
	"google.golang.org/grpc"
)

type (
	writerClient = fgrpc.StreamClientCore[
		writer.Request,
		*segmentv1.WriterRequest,
		writer.Response,
		*segmentv1.WriterResponse,
	]
	writerServerCore = fgrpc.StreamServerCore[
		writer.Request,
		*segmentv1.WriterRequest,
		writer.Response,
		*segmentv1.WriterResponse,
	]
	iteratorClient = fgrpc.StreamClientCore[
		iterator.Request,
		*segmentv1.IteratorRequest,
		iterator.Response,
		*segmentv1.IteratorResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		iterator.Request,
		*segmentv1.IteratorRequest,
		iterator.Response,
		*segmentv1.IteratorResponse,
	]
)

var (
	_ segmentv1.WriterServiceServer   = (*writerServer)(nil)
	_ writer.TransportServer          = (*writerServer)(nil)
	_ writer.TransportClient          = (*writerClient)(nil)
	_ segmentv1.IteratorServiceServer = (*iteratorServer)(nil)
	_ iterator.TransportServer        = (*iteratorServer)(nil)
	_ iterator.TransportClient        = (*iteratorClient)(nil)
	_ segment.Transport               = (*transport)(nil)
)

func New(pool *fgrpc.Pool) *transport {
	return &transport{
		wc: &writerClient{
			Pool:               pool,
			RequestTranslator:  writerRequestTranslator{},
			ResponseTranslator: writerResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*segmentv1.WriterRequest, *segmentv1.WriterResponse], error) {
				return segmentv1.NewWriterServiceClient(conn).Write(ctx)
			},
		},
		ws: &writerServer{writerServerCore: writerServerCore{
			RequestTranslator:  writerRequestTranslator{},
			ResponseTranslator: writerResponseTranslator{},
			ServiceDesc:        &segmentv1.WriterService_ServiceDesc,
		}},
		is: &iteratorServer{iteratorServerCore: iteratorServerCore{
			RequestTranslator:  iteratorRequestTranslator{},
			ResponseTranslator: iteratorResponseTranslator{},
			ServiceDesc:        &segmentv1.IteratorService_ServiceDesc,
		}},
		ic: &iteratorClient{
			Pool:               pool,
			RequestTranslator:  iteratorRequestTranslator{},
			ResponseTranslator: iteratorResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*segmentv1.IteratorRequest, *segmentv1.IteratorResponse], error) {
				return segmentv1.NewIteratorServiceClient(conn).Iterate(ctx)
			},
		},
	}
}

type writerServer struct{ writerServerCore }

func (w *writerServer) Write(server segmentv1.WriterService_WriteServer) error {
	return w.Handler(server.Context(), w.Server(server))
}

type iteratorServer struct{ iteratorServerCore }

func (t *iteratorServer) Iterate(server segmentv1.IteratorService_IterateServer) error {
	return t.Handler(server.Context(), t.Server(server))
}

type transport struct {
	ws *writerServer
	is *iteratorServer
	wc writer.TransportClient
	ic iterator.TransportClient
}

func (t *transport) WriterServer() writer.TransportServer { return t.ws }

func (t *transport) WriterClient() writer.TransportClient { return t.wc }

func (t *transport) IteratorServer() iterator.TransportServer { return t.is }

func (t *transport) IteratorClient() iterator.TransportClient { return t.ic }

func (t *transport) BindTo(server grpc.ServiceRegistrar) {
	segmentv1.RegisterWriterServiceServer(server, t.ws)
	segmentv1.RegisterIteratorServiceServer(server, t.is)
}
