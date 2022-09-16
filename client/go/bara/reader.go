package main

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/freighter/fws"
	"github.com/synnaxlabs/x/httputil"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sync"
	"time"
)

func main() {
	numReceivers := 10
	client := fws.NewClient[api.StreamReaderRequest, api.StreamReaderResponse](httputil.MsgPackEncoderDecoder)
	l := lo.Must(zap.NewDevelopment())
	var wg sync.WaitGroup

	for j := 0; j < numReceivers; j++ {
		stream, err := client.Stream(context.Background(), "ws://localhost:3456/api/v1/stream/read")
		if err != nil {
			panic(err)
		}
		if err := stream.Send(api.StreamReaderRequest{ChannelKeys: []string{"1-1"}}); err != nil {
			panic(err)
		}
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for {
				i++
				t0 := time.Now()
				res, err := stream.Receive()
				if err != nil {
					panic(err)
				}
				if i%100 == 0 {
					l.Info("received samples", zap.Int("id", j), zap.Int("num_samples", len(res.Samples)), zap.Duration("duration", time.Since(t0)))
				}
			}
		}()
	}
	wg.Wait()

}
