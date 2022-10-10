package freighter

import (
	"context"
	"github.com/synnaxlabs/x/address"
)

// UnaryClient is the client side interface of the Unary transport.
type UnaryClient[RQ, RS Payload] interface {
	Transport
	// Send sends a request to the target server using the given context. The context
	// should be canceled if the client expects the server to discard the request
	// and return an error upon receiving it.
	Send(ctx context.Context, target address.Address, req RQ) (res RS, err error)
}

type UnaryServer[RQ, RS Payload] interface {
	Transport
	// BindHandler binds a handle that processes a request from a client. The server
	// is expected to send a response along with any errors encountered during
	// processing. If the provided context is invalid, the server is expected to
	// abort the request and respond with an error (ideally this error should
	// wrap a context error in some form).
	BindHandler(handle func(ctx context.Context, req RQ) (res RS, err error))
}
