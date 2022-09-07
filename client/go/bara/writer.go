package main

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/freighter/fws"
	"github.com/arya-analytics/x/httputil"
	"github.com/arya-analytics/x/telem"
	"go/types"
	"time"
)

func main() {
	client := fws.NewClient[api.StreamWriterRequest, types.Nil](httputil.MsgPackEncoderDecoder)
	stream, err := client.Stream(context.Background(), "ws://localhost:3456/api/v1/stream/write")
	if err != nil {
		panic(err)
	}
	numSamples := 3000
	requestInterval := 10 * time.Millisecond

	t := time.NewTicker(requestInterval)
	for range t.C {
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
