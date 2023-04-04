package channel

import (
	"context"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	proxy *leaseProxy
	txn   gorp.Txn
}

func (w Writer) Create(ctx context.Context, c *Channel) error {
	channels := []Channel{*c}
	err := w.proxy.create(context.TODO(), w.txn, &channels)
	if err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

func (w Writer) CreateMany(ctx context.Context, channels *[]Channel) error {
	return w.proxy.create(ctx, w.txn, channels)
}
