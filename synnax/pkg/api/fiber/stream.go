package fiber

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter/fws"
	"github.com/synnaxlabs/synnax/pkg/api"
	"go/types"
)

type streamService struct{ api.StreamService }

func (ss *streamService) Route(parent fiber.Router) {
	router := parent.Group("/stream")
	writerServer := fws.NewServer[api.StreamWriterRequest, types.Nil](
		nil, ss.Logger,
	)
	readerServer := fws.NewServer[api.StreamReaderRequest, api.StreamReaderResponse](
		nil, ss.Logger,
	)
	writerServer.BindHandler(func(ctx context.Context, stream api.SampleWriterStream) error {
		return ss.Write(ctx, stream)
	})
	readerServer.BindHandler(func(ctx context.Context, stream api.SampleReaderStream) error {
		return ss.Read(ctx, stream)
	})
	writerServer.BindTo(router, "/write")
	readerServer.BindTo(router, "/read")
}
