package server

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"go.uber.org/zap"
)

func BindTo(f *fiber.App, logger *zap.SugaredLogger) {
	router := fhttp.NewRouter(fhttp.RouterConfig{Logger: logger})
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

	unaryEchoServer := fhttp.UnaryPostServer[Message, Message](router, "/unary/echo")
	unaryEchoServer.BindHandler(unaryEcho)

	router.BindTo(f)
}

func unaryEcho(ctx context.Context, req Message) (Message, error) {
	req.ID++
	return req, nil
}

func streamEcho(ctx context.Context, stream ServerStream) error {
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
	ctx context.Context,
	stream ServerStream,
) error {
	for i := 0; i < 10; i++ {
		if err := stream.Send(Message{Message: "hello", ID: i}); err != nil {
			return err
		}
	}
	return nil
}

func streamSendMessageAfterClientClose(
	ctx context.Context,
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
	ctx context.Context,
	stream ServerStream,
) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitWithErr(
	ctx context.Context,
	stream ServerStream,
) error {
	return TestError{Code: 1, Message: "unexpected error"}
}

func streamImmediatelyExitNominally(
	ctx context.Context,
	stream ServerStream,
) error {
	return nil
}
