package channel

import "github.com/synnaxlabs/freighter"

type (
	CreateTransportClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	CreateTransportServer = freighter.UnaryServer[CreateMessage, CreateMessage]
)

type Transport interface {
	CreateClient() CreateTransportClient
	CreateServer() CreateTransportServer
}

type CreateMessage struct {
	Channels []Channel
}
