package framer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	framerv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/framer/v1"
	"google.golang.org/grpc"
)

type (
	writerClient = fgrpc.StreamClientCore[
		writer.Request,
		*framerv1.WriterRequest,
		writer.Response,
		*framerv1.WriterResponse,
	]
	writerServerCore = fgrpc.StreamServerCore[
		writer.Request,
		*framerv1.WriterRequest,
		writer.Response,
		*framerv1.WriterResponse,
	]
	iteratorClient = fgrpc.StreamClientCore[
		iterator.Request,
		*framerv1.IteratorRequest,
		iterator.Response,
		*framerv1.IteratorResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		iterator.Request,
		*framerv1.IteratorRequest,
		iterator.Response,
		*framerv1.IteratorResponse,
	]
)

var (
	_ framerv1.WriterServiceServer   = (*writerServer)(nil)
	_ writer.TransportServer         = (*writerServer)(nil)
	_ writer.TransportClient         = (*writerClient)(nil)
	_ framerv1.IteratorServiceServer = (*iteratorServer)(nil)
	_ iterator.TransportServer       = (*iteratorServer)(nil)
	_ iterator.TransportClient       = (*iteratorClient)(nil)
	_ framer.Transport               = (*transport)(nil)
	_ fgrpc.BindableTransport        = (*transport)(nil)
)

func New(pool *fgrpc.Pool) *transport {
	return &transport{
		writer: &writerTransport{
			client: &writerClient{
				Pool:               pool,
				RequestTranslator:  writerRequestTranslator{},
				ResponseTranslator: writerResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*framerv1.WriterRequest, *framerv1.WriterResponse], error) {
					return framerv1.NewWriterServiceClient(conn).Write(ctx)
				},
			},
			server: &writerServer{writerServerCore: writerServerCore{
				RequestTranslator:  writerRequestTranslator{},
				ResponseTranslator: writerResponseTranslator{},
				ServiceDesc:        &framerv1.WriterService_ServiceDesc,
			}},
		},
		iterator: &iteratorTransport{
			server: &iteratorServer{iteratorServerCore: iteratorServerCore{
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ServiceDesc:        &framerv1.IteratorService_ServiceDesc,
			}},
			client: &iteratorClient{
				Pool:               pool,
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*framerv1.IteratorRequest, *framerv1.IteratorResponse], error) {
					return framerv1.NewIteratorServiceClient(conn).Iterate(ctx)
				},
			},
		},
	}
}

type writerServer struct{ writerServerCore }

func (w *writerServer) Write(server framerv1.WriterService_WriteServer) error {
	return w.Handler(server.Context(), w.Server(server))
}

type iteratorServer struct{ iteratorServerCore }

func (t *iteratorServer) Iterate(server framerv1.IteratorService_IterateServer) error {
	return t.Handler(server.Context(), t.Server(server))
}

type transport struct {
	writer   *writerTransport
	iterator *iteratorTransport
}

func (t *transport) Writer() writer.Transport { return t.writer }

func (t *transport) Iterator() iterator.Transport { return t.iterator }

func (t *transport) BindTo(server grpc.ServiceRegistrar, mw ...freighter.Middleware) {
	framerv1.RegisterWriterServiceServer(server, t.writer.server)
	framerv1.RegisterIteratorServiceServer(server, t.iterator.server)
	t.iterator.server.Use(mw...)
	t.writer.server.Use(mw...)
}

type writerTransport struct {
	client *writerClient
	server *writerServer
}

func (t *writerTransport) Client() writer.TransportClient { return t.client }

func (t *writerTransport) Server() writer.TransportServer { return t.server }

type iteratorTransport struct {
	client *iteratorClient
	server *iteratorServer
}

func (t *iteratorTransport) Client() iterator.TransportClient { return t.client }

func (t *iteratorTransport) Server() iterator.TransportServer { return t.server }
