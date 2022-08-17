package freighter

import (
	"context"
	"github.com/arya-analytics/x/address"
)

// StreamTransport is an entity that implements streaming transport of messages between a
// client and a server. Stream is bidirectional, meaning that both client and server
// can exchange messages in an asynchronous manner. Unary transport should be
// preferred over Stream as it is simpler. Stream transport is useful for cases
// where the client and server/need to exchange many messages over extended
// periods of time in a non-blocking fashion. The semantics of Stream communication
// are more complex than Unary, and care should be taken when managing the Stream
// lifecycle.
type StreamTransport[RQ, RS Payload] interface {
	StreamServiceClient[RQ, RS]
	StreamServiceServer[RQ, RS]
}

type StreamServiceClient[RQ, RS Payload] interface {
	Transport
	// Stream opens a stream to the target server using the given context. If
	// the stream cannot be opened (ex. the server cannot be reached), Stream
	// will return an error. For the semantics of stream operation, see the
	// ClientStream interface.
	Stream(ctx context.Context, target address.Address) (ClientStream[RQ, RS], error)
}

type StreamServiceServer[RQ, RS Payload] interface {
	Transport
	// BindHandler is called by the server to handle a request from the client. If
	// the context is cancelled, the server is expected to discard unprocessed
	// requests, free resources related to the stream, and return an error to
	// the caller (ideally this error should wrap a context error in some form).
	//
	// Transient errors (errors that may be fatal to a request, but not to the stream)
	// should be returned as part of the response itself. This is typically in the
	// form of an 'Err' struct field in the response type RS.
	//
	// Fatal errors (errors that prevent the server from processing any future requests)
	// should be returned from the handle function itself (f). If the handle function
	// returns nil, the server will close the stream, returning a final freighter.EOF
	// error to the client. If the handle function returns an error, the server will
	// close the stream and return the error to the client.
	//
	// For details on the semantics of the handle function, see the ServerStream
	// interface.
	BindHandler(handler func(ctx context.Context, server ServerStream[RQ, RS]) error)
}

// ClientStream is the client side interface of Stream freighter. ClientStream
// differs from ServerStream in that the client has an explicit CloseSend method
// to let the server know that it is done sending messages.
type ClientStream[RQ, RS Payload] interface {
	// StreamReceiver - Receive blocks until a message is received from the
	// server or the stream is closed.
	//
	// Failure Behavior:
	//
	// 1. If the server closed the stream -> If the server closed the
	// stream with a nil error, returns freighter.EOF. Otherwise, returns
	// the error the server closed with.
	//
	// 2. If the client called CloseSend -> Has no effect on the behavior of Receive.
	//
	// 3. If the context is cancelled by either the client or server -> Returns
	// the context error.
	//
	// 4. If the transport fails -> Returns the error that caused the transport
	// to fail.
	//
	// Repeated calls to Receive will immediately return the same error
	// the stream closed with.
	StreamReceiver[RS]
	// StreamSenderCloser -
	//
	// Send sends a message to the server. Send is non-blocking,
	// meaning that the message is not guaranteed to be received by the server even if
	// send returns.
	//
	// Failure Behavior:
	//
	// 1. If the server closed the stream -> Returns a freighter.EOF error
	// regardless of the error the server exited with (even a nil error).
	// The caller can discover the error by calling Receive.
	//
	// 2. If the client called CloseSend -> Returns a freighter.StreamClosed
	// error.
	//
	// 3. If the transport fails -> Returns the error that caused the transport
	// to fail.
	//
	// If the client continues to call Send after receiving an error,
	// Send will continue to return the same error.
	//
	// CloseSend lets the server know it's time to shut down. If the client attempts
	// to call Send after CloseSend, Send will return StreamClosed. The
	// client is free to continue receiving messages from the server after
	// calling CloseSend.
	StreamSenderCloser[RQ]
}

// ServerStream is the server side interface of Stream freighter. ServerStream
// is provided to the caller within a Stream handle. As a result, ServerStream
// provides no `Close` method to the caller.
type ServerStream[RQ, RS Payload] interface {
	// StreamReceiver - Receive blocks until a message is received from the
	// client or the stream closes.
	//
	// Failure Behavior:
	//
	// 1. If the client called CloseSend -> Returns a freighter.EOF error.
	//
	// 2. If the server handler has returned -> This is most likely a programming
	// error where a separate goroutine is writing to the stream after the handler
	// returns. In this case, the server will return a context.Canceled error.
	//
	// 2. If the transport fails -> Returns the error that caused the transport
	//to fail.
	//
	// Repeated calls to Receive will immediately return the same error.
	StreamReceiver[RQ]
	// StreamSender - Send sends a message to the client. Send is non-blocking, meaning
	// the message is not guaranteed to be delivered even if Send returns.
	//
	// Failure Behavior:
	//
	// 1. If the client called CloseSend -> Has no effect on the behavior of Send.
	//
	// 2. If the server handler has returned -> This is most likely a programming
	// error where a separate goroutine is writing to the stream after the handler
	// returns. In this case, the server will return either a context.Canceled
	// or freighter.StreamClosed error.
	//
	// 3. If the transport fails -> Returns the error that caused the transport
	// to fail.
	//
	// Repeated calls to Receive will immediately return the same error.
	StreamSender[RS]
}

// StreamReceiver is an entity that can receive payloads.
type StreamReceiver[P Payload] interface {
	Receive() (P, error)
}

// StreamSender is an entity that can send payloads.
type StreamSender[P Payload] interface {
	Send(P) error
}

// StreamSenderCloser is a type that can send messages as well as close the
// sending end of a stream.
type StreamSenderCloser[P Payload] interface {
	StreamSender[P]
	CloseSend() error
}

// SenderEmptyCloser wraps a StreamSender so that it can satisfy the StreamSenderCloser
// interface. This is useful for types that deal with both ServerStream and ClientStream
// side applications. This allows a ServerStream. StreamSender to be used with client side
// code.
type SenderEmptyCloser[P Payload] struct{ StreamSender[P] }

// CloseSend implements the StreamCloser interface.
func (c SenderEmptyCloser[P]) CloseSend() error { return nil }
