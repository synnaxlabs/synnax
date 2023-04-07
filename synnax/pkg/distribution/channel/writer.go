package channel

import (
	"context"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	proxy *leaseProxy
	tx    gorp.Tx
}

func (w Writer) Create(ctx context.Context, c *Channel) error {
	channels := []Channel{*c}
	err := w.proxy.create(ctx, w.tx, &channels)
	if err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

func (w Writer) CreateMany(ctx context.Context, channels *[]Channel) error {
	return w.proxy.create(ctx, w.tx, channels)
}
