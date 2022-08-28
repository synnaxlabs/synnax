package fiber

import (
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/delta/pkg/ui"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"go.uber.org/zap"
)

type API struct {
	auth    *authService
	segment *segmentService
	ui      *uiService
	channel *channelService
	logger  *zap.Logger
}

func (a *API) Route(router fiber.Router) {
	logger.New()
	router.Use(logMiddleware(a.logger))
	router.Use(pprof.New())
	a.ui.Route(router)
	apiRouter := router.Group("/api/v1")
	a.auth.Route(apiRouter)
	a.segment.Route(apiRouter)
	a.channel.Route(apiRouter)
}

func Wrap(from api.API) API {
	fiberAPI := API{logger: from.Config.Logger}
	fiberAPI.auth = &authService{AuthService: *from.Auth}
	fiberAPI.segment = &segmentService{SegmentService: *from.Segment}
	fiberAPI.ui = &uiService{Dist: ui.Dist}
	fiberAPI.channel = &channelService{ChannelService: *from.Channel}
	return fiberAPI
}
