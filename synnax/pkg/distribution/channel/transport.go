package channel

import "github.com/synnaxlabs/freighter"

type CreateTransport = freighter.Unary[CreateMessage, CreateMessage]

type CreateMessage struct {
	Channels []Channel
}
