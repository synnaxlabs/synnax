package aspen

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/errutil"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
)

type (
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
	c := errutil.NewCatchSimple(errutil.WithAggregation())
	c.Exec(db.wg.Wait)
	c.Exec(db.options.kv.Engine.Close)
	if !errors.Is(c.Error(), context.Canceled) {
		return c.Error()
	}
	return nil
}
