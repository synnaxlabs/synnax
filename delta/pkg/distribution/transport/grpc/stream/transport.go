package stream

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	sv1 "github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/stream/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"go/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	baseWriterTransport = fgrpc.StreamTransportCore[
		stream.WriteRequest,
		*sv1.WriteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	baseReaderTransport = fgrpc.StreamTransportCore[
		stream.ReadRequest,
		*sv1.ReadRequest,
		stream.ReadResponse,
		*sv1.ReadResponse,
	]
)

var (
	_ stream.Transport       = (*transport)(nil)
	_ stream.WriteTransport  = (*writerTransport)(nil)
	_ stream.ReadTransport   = (*readerTransport)(nil)
	_ sv1.WriteServiceServer = (*writerTransport)(nil)
	_ sv1.ReadServiceServer  = (*readerTransport)(nil)
)

// transport implements the stream.Transport interface.
type transport struct {
	reader *readerTransport
	writer *writerTransport
}

func (t *transport) Reader() stream.ReadTransport { return t.reader }

func (t *transport) Writer() stream.WriteTransport { return t.writer }

func (t *transport) BindTo(server grpc.ServiceRegistrar) {
	sv1.RegisterWriteServiceServer(server, t.writer)
	sv1.RegisterReadServiceServer(server, t.reader)
}

func New(pool *fgrpc.Pool) *transport {
	t := &transport{
		reader: &readerTransport{baseReaderTransport: baseReaderTransport{
			Pool:               pool,
			RequestTranslator:  readRequestTranslator{},
			ResponseTranslator: readResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface) (fgrpc.GRPCClientStream[*sv1.ReadRequest, *sv1.ReadResponse], error) {
				return sv1.NewReadServiceClient(conn).Read(ctx)
			},
		}},
		writer: &writerTransport{baseWriterTransport: baseWriterTransport{
			Pool:               pool,
			RequestTranslator:  writeRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface) (fgrpc.GRPCClientStream[*sv1.WriteRequest, *emptypb.Empty], error) {
				return sv1.NewWriteServiceClient(conn).Write(ctx)
			},
		}},
	}
	return t
}

type writerTransport struct{ baseWriterTransport }

func (w *writerTransport) Write(server sv1.WriteService_WriteServer) error {
	return w.Handler(server.Context(), w.Server(server))
}

type readerTransport struct{ baseReaderTransport }

func (r *readerTransport) Read(server sv1.ReadService_ReadServer) error {
	return r.Handler(server.Context(), r.Server(server))
}
