package channel

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve information about Channel(s) in delta's distribution
// layer.
type Retrieve struct {
	gorp gorp.Retrieve[Key, Channel]
	db   *gorp.DB
}

func newRetrieve(db *gorp.DB) Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[Key, Channel](), db: db}
}

// Entry binds the Channel that Retrieve will fill results into. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp.Entry(ch); return r }

// Entries binds a slice that Retrieve will fill results into.  This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp.Entries(ch); return r }

// WhereNodeID filters for channels whose NodeID attribute matches the provided
// leaseholder node ID.
func (r Retrieve) WhereNodeID(nodeID core.NodeID) Retrieve {
	r.gorp.Where(func(ch *Channel) bool { return ch.NodeID == nodeID })
	return r
}

// WhereKeys filters for channels with the provided Key. This is an identical interface
// to gorp.Retrieve.
func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp.WhereKeys(keys...)
	return r
}

// WithTxn binds a transaction the query will be executed within. If the option is not set,
// the query will be executed directly against the Service database.
func (r Retrieve) WithTxn(txn gorp.Txn) Retrieve { gorp.SetTxn(r.gorp, txn); return r }

// Exec executes the query, binding
func (r Retrieve) Exec(ctx context.Context) error { return r.gorp.Exec(gorp.GetTxn(r.gorp, r.db)) }

// Exists checks if the query has results matching its parameters. If used in conjunction
// with WhereKeys, Exists will ONLY return true if ALL the keys have a matching Channel.
// Otherwise, Exists returns true if the query has ANY results.
func (r Retrieve) Exists(ctx context.Context) (bool, error) {
	return r.gorp.Exists(gorp.GetTxn(r.gorp, r.db))
}
