package stream

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/google/uuid"
	"sync"
)

type server struct {
	Config
	write   Inlet
	delta   *confluence.DynamicDeltaMultiplier[[]Sample]
	demands *demandCoordinator
}

func newServer(
	cfg Config,
	write Inlet,
	delta *confluence.DynamicDeltaMultiplier[[]Sample],
	demands *demandCoordinator,
) *server {
	sf := &server{
		Config:  cfg,
		write:   write,
		delta:   delta,
		demands: demands,
	}
	sf.Transport.Writer().BindHandler(sf.Write)
	sf.Transport.Reader().BindHandler(sf.Read)
	return sf
}

func (sf *server) Write(ctx context.Context, server WriteServerStream) error {
	for {
		req, err := server.Receive()
		if err != nil {
			return err
		}
		if err := signal.SendUnderContext(
			ctx, sf.write.Inlet(), req.Samples,
		); err != nil {
			return err
		}
	}
}

func (sf *server) Read(_ctx context.Context, server ReadServerStream) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	addr := address.Address(uuid.New().String())
	var (
		mu   sync.Mutex
		keys map[channel.Key]struct{}
	)

	go func() {
		req, err := server.Receive()
		if err != nil {
			sf.demands.clear(addr)
			cancel()
			return
		}
		mu.Lock()
		keys = make(map[channel.Key]struct{})
		for _, key := range req.Keys {
			keys[key] = struct{}{}
		}
		sf.demands.set(addr, req.Keys)
		mu.Unlock()
	}()

	samples := confluence.NewStream[[]Sample]()
	samples.SetInletAddress(addr)
	sf.delta.Connect(samples)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-samples.Outlet():
			var outSamples []Sample
			mu.Lock()
			for _, sample := range s {
				if _, ok := keys[sample.ChannelKey]; ok {
					outSamples = append(outSamples, sample)
				}
			}
			mu.Unlock()
			if err := server.Send(ReadResponse{Samples: outSamples}); err != nil {
				return err
			}
		}
	}
}
