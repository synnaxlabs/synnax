package channel

import "github.com/arya-analytics/freighter"

type CreateTransport = freighter.Unary[CreateMessage, CreateMessage]

type CreateMessage struct {
	Channels []Channel
}
