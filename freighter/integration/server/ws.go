package server

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/fws"
	"github.com/arya-analytics/x/httputil"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Websocket struct {
	Logger  *zap.SugaredLogger
	Encoder httputil.EncoderDecoder
}

func (w Websocket) BindTo(f fiber.Router) {
	router := f.Group("/ws")
	echoServer := fws.NewServer[Message, Message](nil, w.Logger)
	echoServer.BindTo(router, "/echo")
	echoServer.BindHandler(wsEcho)

	sendMessageAfterClientCloseServer := fws.NewServer[Message, Message](nil, w.Logger)
	sendMessageAfterClientCloseServer.BindTo(router, "/sendMessageAfterClientClose")
	sendMessageAfterClientCloseServer.BindHandler(wsSendMessageAfterClientClose)

	receiveAndExitWithErrServer := fws.NewServer[Message, Message](nil, w.Logger)
	receiveAndExitWithErrServer.BindTo(router, "/receiveAndExitWithErr")
	receiveAndExitWithErrServer.BindHandler(wsReceiveAndExitWithErr)

	immediatelyExitWithErrServer := fws.NewServer[Message, Message](nil, w.Logger)
	immediatelyExitWithErrServer.BindTo(router, "/immediatelyExitWithErr")
	immediatelyExitWithErrServer.BindHandler(wsImmediatelyExitWithErr)

	immediatelyExitNominallyServer := fws.NewServer[Message, Message](nil, w.Logger)
	immediatelyExitNominallyServer.BindTo(router, "/immediatelyExitNominally")
	immediatelyExitNominallyServer.BindHandler(wsImmediatelyExitNominally)

	respondWithTenMessagesServer := fws.NewServer[Message, Message](nil, w.Logger)
	respondWithTenMessagesServer.BindTo(router, "/respondWithTenMessages")
	respondWithTenMessagesServer.BindHandler(wsRespondWithTenMessages)
}

func wsEcho(ctx context.Context, stream ServerStream) error {
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

func wsRespondWithTenMessages(
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

func wsSendMessageAfterClientClose(
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

func wsReceiveAndExitWithErr(
	ctx context.Context,
	stream ServerStream,
) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	return TestError{Code: 1, Message: "unexpected error"}
}

func wsImmediatelyExitWithErr(
	ctx context.Context,
	stream ServerStream,
) error {
	return TestError{Code: 1, Message: "unexpected error"}
}

func wsImmediatelyExitNominally(
	ctx context.Context,
	stream ServerStream,
) error {
	return nil
}
