package fiber

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter/fws"
	"github.com/synnaxlabs/synnax/pkg/api"
)

type segmentService struct{ api.SegmentService }

func (ss *segmentService) Route(parent fiber.Router) {
	router := parent.Group("/segment")
	writerServer := fws.NewServer[api.SegmentWriterRequest, api.SegmentWriterResponse](
		nil, ss.Logger,
	)
	iterServer := fws.NewServer[api.IteratorRequest, api.IteratorResponse](
		nil, ss.Logger,
	)

	// We wrap closures around the following handlers to guarantee a typed error
	// goodResponse.
	writerServer.BindHandler(func(ctx context.Context, stream api.SegmentWriterStream) error {
		return ss.Write(ctx, stream)
	})
	iterServer.BindHandler(func(ctx context.Context, stream api.IteratorStream) error {
		return ss.Iterate(ctx, stream)
	})

	iterServer.BindTo(router, "/iterate")
	writerServer.BindTo(router, "/write")
}
