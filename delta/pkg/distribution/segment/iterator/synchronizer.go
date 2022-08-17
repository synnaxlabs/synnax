package iterator

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"time"
)

type synchronizer struct {
	timeout time.Duration
	nodeIDs []core.NodeID
	confluence.UnarySink[Response]
}

func (a *synchronizer) sync(ctx context.Context, command Command) (bool, error) {
	ctx, cancel := signal.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	acknowledgements := make([]core.NodeID, 0, len(a.nodeIDs))
	for {
		select {
		case <-ctx.Done():
			return false, errors.Wrap(ctx.Err(), "[synchronizer] - timed out")
		case r, ok := <-a.In.Outlet():
			if r.Command != command {
				continue
			}
			if !ok {
				panic(
					"[iterator.synchronizer] - response pipe closed before all nodes acked command",
				)
			}
			if !lo.Contains(acknowledgements, r.NodeID) {
				// If any node does not consider the request as valid, then we consider
				// the entire command as invalid.
				if !r.Ack {
					return false, r.Error
				}
				acknowledgements = append(acknowledgements, r.NodeID)
			}
			if len(acknowledgements) == len(a.nodeIDs) {
				return true, nil
			}
		}
	}
}
