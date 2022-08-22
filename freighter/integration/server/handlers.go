package server

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

func echo(ctx context.Context, stream ServerStream) error {
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

func sendMessageAfterClientClose(
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

func receiveAndExitWithErr(
	ctx context.Context,
	stream ServerStream,
) error {
	_, err := stream.Receive()
	if err != nil {
		return err
	}
	return TestError{Code: 1, Message: "unexpected error"}
}

func immediatelyExitWithErr(
	ctx context.Context,
	stream ServerStream,
) error {
	return TestError{Code: 1, Message: "unexpected error"}
}

func immediatelyExitNominally(
	ctx context.Context,
	stream ServerStream,
) error {
	return nil
}
