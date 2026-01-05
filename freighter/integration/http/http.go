// Copyright 2026 Synnax Labs, Inc.
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
	"go/types"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

func BindTo(f *fiber.App) {
	router := fhttp.NewRouter(fhttp.RouterConfig{
		Instrumentation:     testutil.Instrumentation("freighter-integration"),
		StreamWriteDeadline: 50 * time.Millisecond,
	})
	echoServer := fhttp.StreamServer[Message, Message](router, "/stream/echo")
	echoServer.BindHandler(streamEcho)

	streamSendMessageAfterClientCloseServer := fhttp.StreamServer[Message, Message](router, "/stream/sendMessageAfterClientClose")
	streamSendMessageAfterClientCloseServer.BindHandler(streamSendMessageAfterClientClose)

	streamReceiveAndExitWithErrServer := fhttp.StreamServer[Message, Message](router, "/stream/receiveAndExitWithErr")
	streamReceiveAndExitWithErrServer.BindHandler(streamReceiveAndExitWithErr)

	streamImmediatelyExitWithErrServer := fhttp.StreamServer[Message, Message](router, "/stream/immediatelyExitWithErr")
	streamImmediatelyExitWithErrServer.BindHandler(streamImmediatelyExitWithErr)

	streamImmediatelyExitNominallyServer := fhttp.StreamServer[Message, Message](router, "/stream/immediatelyExitNominally")
	streamImmediatelyExitNominallyServer.BindHandler(streamImmediatelyExitNominally)

	streamRespondWithTenMessagesServer := fhttp.StreamServer[Message, Message](router, "/stream/respondWithTenMessages")
	streamRespondWithTenMessagesServer.BindHandler(streamRespondWithTenMessages)

	unaryGetEchoServer := fhttp.UnaryServer[Message, Message](router, "/unary/echo")
	unaryGetEchoServer.BindHandler(unaryEcho)

	unaryMiddlewareCheckServer := fhttp.UnaryServer[Message, Message](router, "/unary/middlewareCheck")
	unaryMiddlewareCheckServer.BindHandler(unaryEcho)
	unaryMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamMiddlewareCheckServer := fhttp.StreamServer[Message, Message](router, "/stream/middlewareCheck")
	streamMiddlewareCheckServer.BindHandler(streamEcho)
	streamMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamSlamMessagesServer := fhttp.StreamServer[Message, Message](router, "/stream/slamMessages")
	streamSlamMessagesServer.BindHandler(streamSlamMessages)
	slamMessagesTimeoutCheck := fhttp.UnaryServer[Message, Message](router, "/unary/slamMessagesTimeoutCheck")
	slamMessagesTimeoutCheck.BindHandler(slamMessagesTimeoutCheckHandler)

	streamEventuallyResponseWithMessageServer := fhttp.StreamServer[Message, Message](router, "/stream/eventuallyResponseWithMessage")
	streamEventuallyResponseWithMessageServer.BindHandler(streamEventuallyResponseWithMessage)

	router.BindTo(f)
}

func checkMiddleware(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
	if ctx.Params["Test"] != "test" {
		return ctx, TestError{Message: "test param not found", Code: 1}
	}
	return next(ctx)
}

func unaryEcho(_ context.Context, req Message) (Message, error) {
	req.ID++
	return req, nil
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

func streamRespondWithTenMessages(
	_ context.Context,
	stream ServerStream,
) error {
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

func streamSlamMessages(
	_ context.Context,
	stream ServerStream,
) error {
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

func streamEventuallyResponseWithMessage(
	_ context.Context,
	stream ServerStream,
) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return stream.Send(Message{Message: "hello", ID: 1})
}

func slamMessagesTimeoutCheckHandler(
	_ context.Context,
	msg Message,
) (Message, error) {
	timeoutMu.Lock()
	defer timeoutMu.Unlock()
	if _, ok := timeouts[msg.Message]; ok {
		return Message{Message: "timeout"}, nil
	}
	return Message{Message: "success"}, nil
}

func streamSendMessageAfterClientClose(
	_ context.Context,
	stream ServerStream,
) error {
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

func streamReceiveAndExitWithErr(
	_ context.Context,
	stream ServerStream,
) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitWithErr(
	context.Context,
	ServerStream,
) error {
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitNominally(
	context.Context,
	ServerStream,
) error {
	return nil
}
