package channel

import (
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	proxy  *leaseProxy
	writer gorp.WriteContext
}

func (w Writer) Create(c *Channel) error {
	channels := []Channel{*c}
	err := w.proxy.create(w.writer, &channels)
	if err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

func (w Writer) CreateMany(channels *[]Channel) error {
	return w.proxy.create(w.writer, channels)
}
