package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/ui"
	"go.uber.org/zap"
	"time"
)

type API struct {
	auth    *authService
	segment *segmentService
	ui      *uiService
	channel *channelService
	stream  *streamService
	logger  *zap.Logger
}

func (a *API) Route(router fiber.Router) {
	logger.New()
	router.Use(logMiddleware(a.logger))
	router.Use(pprof.New())
	router.Get("/metrics", monitor.New(monitor.Config{
		Refresh: 500 * time.Millisecond,
	}))
	a.ui.Route(router)
	apiRouter := router.Group("/api/v1")
	a.auth.Route(apiRouter)
	a.segment.Route(apiRouter)
	a.channel.Route(apiRouter)
	a.stream.Route(apiRouter)
}

func Wrap(from api.API) API {
	fiberAPI := API{logger: from.Config.Logger}
	fiberAPI.auth = &authService{AuthService: *from.Auth}
	fiberAPI.segment = &segmentService{SegmentService: *from.Segment}
	fiberAPI.ui = &uiService{Dist: ui.Dist}
	fiberAPI.channel = &channelService{ChannelService: *from.Channel}
	fiberAPI.stream = &streamService{StreamService: *from.Stream}
	return fiberAPI
}
