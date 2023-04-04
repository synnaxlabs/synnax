package signal

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	xio "github.com/synnaxlabs/x/io"
	"io"
)

func NewShutdown(
	wg WaitGroup,
	cancel context.CancelFunc,
) io.Closer {
	return xio.CloserFunc(func() error {
		cancel()
		err := wg.Wait()
		return lo.Ternary(errors.Is(err, context.Canceled), nil, err)
	})
}
