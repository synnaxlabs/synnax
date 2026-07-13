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
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

func BindTo(f *fiber.App) error {
	router, err := http.NewRouter(http.RouterConfig{
		Instrumentation:     testutil.Instrumentation("freighter-integration"),
		StreamWriteDeadline: 50 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	echoServer := http.NewStreamServer[Message, Message](router, "/stream/echo")
	echoServer.BindHandler(streamEcho)

	streamSendMessageAfterClientCloseServer := http.NewStreamServer[Message, Message](router, "/stream/sendMessageAfterClientClose")
	streamSendMessageAfterClientCloseServer.BindHandler(streamSendMessageAfterClientClose)

	streamReceiveAndExitWithErrServer := http.NewStreamServer[Message, Message](router, "/stream/receiveAndExitWithErr")
	streamReceiveAndExitWithErrServer.BindHandler(streamReceiveAndExitWithErr)

	streamImmediatelyExitWithErrServer := http.NewStreamServer[Message, Message](router, "/stream/immediatelyExitWithErr")
	streamImmediatelyExitWithErrServer.BindHandler(streamImmediatelyExitWithErr)

	streamImmediatelyExitNominallyServer := http.NewStreamServer[Message, Message](router, "/stream/immediatelyExitNominally")
	streamImmediatelyExitNominallyServer.BindHandler(streamImmediatelyExitNominally)

	streamRespondWithTenMessagesServer := http.NewStreamServer[Message, Message](router, "/stream/respondWithTenMessages")
	streamRespondWithTenMessagesServer.BindHandler(streamRespondWithTenMessages)

	unaryGetEchoServer := http.NewUnaryServer[Message, Message](router, "/unary/echo")
	unaryGetEchoServer.BindHandler(unaryEcho)

	unaryParamEchoServer := http.NewUnaryServer[Message, Message](router, "/unary/paramEcho")
	unaryParamEchoServer.BindHandler(unaryParamEcho)

	unaryMiddlewareCheckServer := http.NewUnaryServer[Message, Message](router, "/unary/middlewareCheck")
	unaryMiddlewareCheckServer.BindHandler(unaryEcho)
	unaryMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamMiddlewareCheckServer := http.NewStreamServer[Message, Message](router, "/stream/middlewareCheck")
	streamMiddlewareCheckServer.BindHandler(streamEcho)
	streamMiddlewareCheckServer.Use(freighter.MiddlewareFunc(checkMiddleware))

	streamSlamMessagesServer := http.NewStreamServer[Message, Message](router, "/stream/slamMessages")
	streamSlamMessagesServer.BindHandler(streamSlamMessages)
	slamMessagesTimeoutCheck := http.NewUnaryServer[Message, Message](router, "/unary/slamMessagesTimeoutCheck")
	slamMessagesTimeoutCheck.BindHandler(slamMessagesTimeoutCheckHandler)

	streamEventuallyResponseWithMessageServer := http.NewStreamServer[Message, Message](router, "/stream/eventuallyResponseWithMessage")
	streamEventuallyResponseWithMessageServer.BindHandler(streamEventuallyResponseWithMessage)

	router.BindTo(f)

	// Bound as a raw fiber route rather than a freighter unary server because freighter
	// maps every handler error to 400, and this endpoint needs to return 503 to model a
	// transient, retryable transport failure (a 400 validation error would not, and
	// must not, be retried).
	f.Post("/unary/flakyUnavailable", flakyUnavailable)

	// Bound as a raw fiber route so it can return a 200 with a genuinely empty body,
	// which a freighter unary server would never produce; lets clients verify they
	// reject an empty successful response rather than accepting it.
	f.Post("/unary/emptyResponse", func(c fiber.Ctx) error {
		c.Status(fiber.StatusOK)
		return nil
	})

	return nil
}

func checkMiddleware(
	ctx freighter.Context,
	next freighter.Next,
) (freighter.Context, error) {
	if ctx.Params["Test"] != "test" {
		return ctx, TestError{Message: "test param not found", Code: 1}
	}
	return next(ctx)
}

func unaryEcho(_ context.Context, req Message) (Message, error) {
	req.ID++
	return req, nil
}

// unaryParamEcho echoes the request params named in req.Message (comma-separated)
// back sorted and joined by "|", with absent params echoed as empty strings. Values
// are sorted so the response is deterministic regardless of the order the keys are
// requested in. Lets clients verify which out-of-band request params reached the
// handler and with what values.
func unaryParamEcho(ctx context.Context, req Message) (Message, error) {
	keys := strings.Split(req.Message, ",")
	values := make([]string, len(keys))
	for i, k := range keys {
		if v, ok := freighter.MDFromContext(ctx).Get(k); ok {
			values[i], _ = v.(string)
		}
	}
	slices.Sort(values)
	return Message{Message: strings.Join(values, "|")}, nil
}

var (
	flakyMu   sync.Mutex
	flakySeen = set.New[string]()
)

// flakyUnavailable responds with 503 Service Unavailable the first time it sees a given
// message, then echoes nominally on every subsequent request for that message. A 503
// models a transient transport failure that a retry policy is meant to recover from
// (unlike a validation error), so it lets clients exercise retry behavior
// deterministically: a caller that retries recovers on the second attempt, while a
// caller that does not retry surfaces the 503.
func flakyUnavailable(c fiber.Ctx) error {
	var req Message
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "application/json")
	flakyMu.Lock()
	defer flakyMu.Unlock()
	if !flakySeen.Contains(req.Message) {
		flakySeen.Add(req.Message)
		c.Status(fiber.StatusServiceUnavailable)
		return c.SendString(`{"type":"integration.error","data":"1,transient unavailable"}`)
	}
	req.ID++
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(body)
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
	timeouts  = set.New[string]()
)

func streamSlamMessages(_ context.Context, stream ServerStream) error {
	msg, err := stream.Receive()
	if err != nil {
		return err
	}
	for i := range 1_000_000 {
		if err := stream.Send(Message{Message: "hello", ID: i}); err != nil {
			timeoutMu.Lock()
			timeouts.Add(msg.Message)
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

func slamMessagesTimeoutCheckHandler(_ context.Context, msg Message) (Message, error) {
	timeoutMu.Lock()
	defer timeoutMu.Unlock()
	if timeouts.Contains(msg.Message) {
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
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitWithErr(context.Context, ServerStream) error {
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitNominally(context.Context, ServerStream) error { return nil }
