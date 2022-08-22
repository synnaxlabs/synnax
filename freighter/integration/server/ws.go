package server

import (
	"github.com/arya-analytics/freighter/fws"
	"github.com/arya-analytics/x/httputil"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Websocket struct {
	Logger  *zap.SugaredLogger
	Encoder httputil.EncoderDecoder
}

func (w Websocket) BindTo(f fiber.Router) {
	echoServer := fws.NewServer[Message, Message](nil, w.Logger)
	echoServer.BindTo(f, "/echo")
	echoServer.BindHandler(echo)

	sendMessageAfterClientCloseServer := fws.NewServer[Message, Message](nil, w.Logger)
	sendMessageAfterClientCloseServer.BindTo(f, "/sendMessageAfterClientClose")
	sendMessageAfterClientCloseServer.BindHandler(sendMessageAfterClientClose)

	receiveAndExitWithErrServer := fws.NewServer[Message, Message](nil, w.Logger)
	receiveAndExitWithErrServer.BindTo(f, "/receiveAndExitWithErr")
	receiveAndExitWithErrServer.BindHandler(receiveAndExitWithErr)

	immediatelyExitWithErrServer := fws.NewServer[Message, Message](nil, w.Logger)
	immediatelyExitWithErrServer.BindTo(f, "/immediatelyExitWithErr")
	immediatelyExitWithErrServer.BindHandler(immediatelyExitWithErr)

	immediatelyExitNominallyServer := fws.NewServer[Message, Message](nil, w.Logger)
	immediatelyExitNominallyServer.BindTo(f, "/immediatelyExitNominally")
	immediatelyExitNominallyServer.BindHandler(immediatelyExitNominally)
}
