// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"fmt"
	"go/types"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/freighter/integration/payload"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

type Message = payload.Message
type ServerStream = payload.ServerStream

func BindTo(app *fiber.App) {
	router := lo.Must(fhttp.NewRouter(fhttp.RouterConfig{
		Instrumentation:     testutil.Instrumentation("freighter-integration"),
		StreamWriteDeadline: 50 * time.Millisecond,
	}))
	echoServer := fhttp.NewStreamServer[Message, Message](router, "/stream/echo")
	echoServer.BindHandler(streamEcho)

	streamSendMessageAfterClientCloseServer := fhttp.NewStreamServer[Message, Message](router, "/stream/sendMessageAfterClientClose")
	streamSendMessageAfterClientCloseServer.BindHandler(streamSendMessageAfterClientClose)

	streamReceiveAndExitWithErrServer := fhttp.NewStreamServer[Message, Message](router, "/stream/receiveAndExitWithErr")
	streamReceiveAndExitWithErrServer.BindHandler(streamReceiveAndExitWithErr)

	streamImmediatelyExitWithErrServer := fhttp.NewStreamServer[Message, Message](router, "/stream/immediatelyExitWithErr")
	streamImmediatelyExitWithErrServer.BindHandler(streamImmediatelyExitWithErr)

	streamImmediatelyExitNominallyServer := fhttp.NewStreamServer[Message, Message](router, "/stream/immediatelyExitNominally")
	streamImmediatelyExitNominallyServer.BindHandler(streamImmediatelyExitNominally)

	streamRespondWithTenMessagesServer := fhttp.NewStreamServer[Message, Message](router, "/stream/respondWithTenMessages")
	streamRespondWithTenMessagesServer.BindHandler(streamRespondWithTenMessages)

	unaryGetEchoServer := fhttp.NewUnaryServer[Message, Message](router, "/unary/echo")
	unaryGetEchoServer.BindHandler(unaryEcho)

	unaryGetReaderServer := fhttp.NewUnaryServer[Message, io.Reader](router, "/unary/getReader")
	unaryGetReaderServer.BindHandler(unaryGetReader)

	unaryTextResponseServer := fhttp.NewUnaryServer[Message, *UnaryTextResponse](
		router,
		"/unary/textResponse",
		fhttp.WithResponseEncoders(map[string]func() binary.Encoder{
			fhttp.MIMETextPlain: func() binary.Encoder { return binary.StringCodec },
		}),
	)
	unaryTextResponseServer.BindHandler(unaryTextResponseHandler)

	unaryMiddlewareCheckServer := fhttp.NewUnaryServer[Message, Message](router, "/unary/middlewareCheck")
	unaryMiddlewareCheckServer.BindHandler(unaryEcho)
	unaryMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamMiddlewareCheckServer := fhttp.NewStreamServer[Message, Message](router, "/stream/middlewareCheck")
	streamMiddlewareCheckServer.BindHandler(streamEcho)
	streamMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamSlamMessagesServer := fhttp.NewStreamServer[Message, Message](router, "/stream/slamMessages")
	streamSlamMessagesServer.BindHandler(streamSlamMessages)
	slamMessagesTimeoutCheck := fhttp.NewUnaryServer[Message, Message](router, "/unary/slamMessagesTimeoutCheck")
	slamMessagesTimeoutCheck.BindHandler(slamMessagesTimeoutCheckHandler)

	streamEventuallyResponseWithMessageServer := fhttp.NewStreamServer[Message, Message](router, "/stream/eventuallyResponseWithMessage")
	streamEventuallyResponseWithMessageServer.BindHandler(streamEventuallyResponseWithMessage)

	router.BindTo(app)
}

func checkMiddleware(
	ctx freighter.Context,
	next freighter.MiddlewareHandler,
) (freighter.Context, error) {
	if ctx.Params["Test"] != "test" {
		return ctx, payload.Error{Message: "test param not found", Code: 1}
	}
	return next(ctx)
}

func unaryEcho(_ context.Context, req Message) (Message, error) {
	req.ID++
	return req, nil
}

func unaryGetReader(_ context.Context, req Message) (io.Reader, error) {
	return strings.NewReader(req.Message), nil
}

type readingEnum int

const (
	readingID readingEnum = iota + 1
	readingMessage
	readingDone
)

type UnaryTextResponse struct {
	readingEnum
	req Message
}

var _ fhttp.UnaryReadable = (*UnaryTextResponse)(nil)

func newUnaryTextResponse(req Message) *UnaryTextResponse {
	return &UnaryTextResponse{
		readingEnum: readingID,
		req:         req,
	}
}

func (u *UnaryTextResponse) Read() (any, error) {
	switch u.readingEnum {
	case readingID:
		u.readingEnum = readingMessage
		return fmt.Sprintf("ID: %d ", u.req.ID), nil
	case readingMessage:
		u.readingEnum = readingDone
		if u.req.Message == "error" {
			return nil, errors.New("failed to read message")
		}
		return fmt.Sprintf("Message: %s", u.req.Message), nil
	case readingDone:
		return nil, io.EOF
	}
	panic("unreachable")
}

func unaryTextResponseHandler(_ context.Context, req Message) (*UnaryTextResponse, error) {
	return newUnaryTextResponse(req), nil
}

func streamEcho(_ context.Context, stream ServerStream) error {
	for {
		msg, err := stream.Receive()
		if err != nil {
			return err
		}
		msg.ID++
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}

func streamRespondWithTenMessages(_ context.Context, stream ServerStream) error {
	for i := range 10 {
		if err := stream.Send(Message{Message: "hello", ID: i}); err != nil {
			return err
		}
	}
	return nil
}

var (
	timeoutMu sync.Mutex
	timeouts  map[string]types.Nil = make(map[string]types.Nil)
)

func streamSlamMessages(_ context.Context, stream ServerStream) error {
	msg, err := stream.Receive()
	if err != nil {
		return err
	}
	for i := range 1_000_000 {
		if err := stream.Send(Message{Message: "hello", ID: i}); err != nil {
			timeoutMu.Lock()
			timeouts[msg.Message] = types.Nil{}
			timeoutMu.Unlock()
			return err
		}
	}
	return nil
}

func streamEventuallyResponseWithMessage(_ context.Context, stream ServerStream) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return stream.Send(Message{Message: "hello", ID: 1})
}

func slamMessagesTimeoutCheckHandler(_ context.Context, msg payload.Message) (payload.Message, error) {
	timeoutMu.Lock()
	defer timeoutMu.Unlock()
	if _, ok := timeouts[msg.Message]; ok {
		return Message{Message: "timeout"}, nil
	}
	return Message{Message: "success"}, nil
}

func streamSendMessageAfterClientClose(_ context.Context, stream ServerStream) error {
	for {
		msg, err := stream.Receive()
		if errors.Is(err, freighter.EOF) {
			return stream.Send(Message{Message: "Close Acknowledged"})
		}
		if err != nil {
			return err
		}
		zap.S().Warnw("server received unexpected message", "msg", msg)
	}
}

func streamReceiveAndExitWithErr(_ context.Context, stream ServerStream) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	return payload.Error{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitWithErr(context.Context, ServerStream) error {
	return payload.Error{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitNominally(context.Context, ServerStream) error { return nil }
