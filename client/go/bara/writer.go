package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/freighter/fws"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/telem"
	"go/types"
	"time"
)

func main() {
	client := fws.NewClient[api.StreamWriterRequest, types.Nil](httputil.MsgPackEncoderDecoder)
	stream, err := client.Stream(context.Background(), "ws://localhost:3456/api/v1/stream/write")
	if err != nil {
		panic(err)
	}
	numSamples := 30
	requestInterval := 1000 * time.Millisecond

	t := time.NewTicker(requestInterval)
	for range t.C {
		logrus.Info("Sending request")
		samples := make([]api.Sample, numSamples)
		for i := 0; i < numSamples; i++ {
			samples[i] = api.Sample{
				ChannelKey: "1-1",
				Stamp:      telem.Now(),
				Value:      []byte("test"),
			}
		}
		err := stream.Send(api.StreamWriterRequest{Samples: samples})
		if err != nil {
			panic(err)
		}
	}
	go func() {
		for {
			_, err := stream.Receive()
			if err != nil {
				panic(err)
			}
		}
	}()
}
