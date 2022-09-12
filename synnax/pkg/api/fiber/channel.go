package fiber

import (
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/gofiber/fiber/v2"
)

type channelService struct{ api.ChannelService }

func (ch *channelService) Route(parent fiber.Router) {
	router := parent.Group("/channel")
	router.Post("/create", ch.create)
	router.Get("/retrieve", ch.retrieve)
}

func (ch *channelService) create(c *fiber.Ctx) error {
	var req api.ChannelCreateRequest
	if err := parseBody(c, &req); err.Occurred() {
		return errorResponse(c, err)
	}
	res, err := ch.Create(c.Context(), req)
	return maybeGoodResponse(c, err, res)
}

func (ch *channelService) retrieve(c *fiber.Ctx) error {
	var req api.ChannelRetrieveRequest
	if err := parseQueryParams(c, &req); err.Occurred() {
		return errorResponse(c, err)
	}
	res, err := ch.Retrieve(c.Context(), req)
	return maybeGoodResponse(c, err, res)
}
