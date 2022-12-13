package aspen

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/aspen/transport"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errutil"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
)

type (
	Transport    = transport.Transport
	Cluster      = cluster.Cluster
	Resolver     = cluster.Resolver
	HostResolver = cluster.HostResolver
	Node         = node.Node
	NodeID       = node.ID
	Address      = address.Address
	NodeState    = node.State
	ClusterState = cluster.State
)

type KV interface {
	kv.DB
	kvx.Closer
}

const (
	Healthy = node.StateHealthy
	Left    = node.StateLeft
	Dead    = node.StateDead
	Suspect = node.StateSuspect
)

type DB interface {
	Cluster
	KV
}

type db struct {
	Cluster
	kv.DB
	options  *options
	wg       signal.WaitGroup
	shutdown context.CancelFunc
}

func (db *db) Close() error {
	db.shutdown()
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(db.wg.Wait)
	c.Exec(db.options.kv.Engine.Close)
	return lo.Ternary(errors.Is(c.Error(), context.Canceled), nil, c.Error())
}
