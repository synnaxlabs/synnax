package writer

import (
	"context"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct{ ServiceConfig }

func startServer(cfg ServiceConfig) *server {
	s := &server{ServiceConfig: cfg}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

func (sf *server) handle(ctx context.Context, server ServerStream) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	// The first request provides the parameters for opening the toStorage writer
	req, err := server.Receive()
	if err != nil {
		return err
	}

	// Senders and receivers must be set up to distribution requests and responses
	// to their storage counterparts.
	receiver := &freightfluence.TransformReceiver[cesium.WriteRequest, Request]{Receiver: server}
	receiver.Transform = newRequestTranslator()
	sender := &freightfluence.TransformSender[cesium.WriteResponse, Response]{Sender: freighter.SenderNopCloser[Response]{StreamSender: server}}
	sender.Transform = newResponseTranslator(sf.HostResolver.HostID())

	w, err := sf.TS.NewStreamWriter(req.Config.toStorage())
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment[cesium.WriteRequest, cesium.WriteResponse](pipe, "toStorage", w)
	plumber.SetSource[cesium.WriteRequest](pipe, "receiver", receiver)
	plumber.SetSink[cesium.WriteResponse](pipe, "sender", sender)
	plumber.MustConnect[cesium.WriteRequest](pipe, "receiver", "toStorage", 1)
	plumber.MustConnect[cesium.WriteResponse](pipe, "toStorage", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())

	return sCtx.Wait()
}
