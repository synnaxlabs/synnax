package mock

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/errutil"
	"os"
	"path/filepath"
	"strconv"
)

type Builder struct {
	PortRangeStart  int
	DataDir         string
	DefaultOptions  []aspen.Option
	peerAddresses   []address.Address
	TmpDirs         map[aspen.NodeID]string
	tmpDir          string
	_addressFactory *address.Factory
	Nodes           map[aspen.NodeID]NodeInfo
	memBacked       bool
}

type NodeInfo struct {
	Addr address.Address
	Dir  string
	DB   aspen.DB
}

func (b *Builder) Dir() string {
	if b.tmpDir == "" {
		var err error
		b.tmpDir, err = os.MkdirTemp(b.DataDir, "aspen")
		if err != nil {
			panic(err)
		}
	}
	return b.tmpDir
}

func (b *Builder) addressFactory() *address.Factory {
	if b._addressFactory == nil {
		b._addressFactory = address.NewLocalFactory(b.PortRangeStart)
	}
	return b._addressFactory
}

func (b *Builder) New(opts ...aspen.Option) (aspen.DB, error) {
	dir := filepath.Join(b.Dir(), strconv.Itoa(len(b.peerAddresses)))
	if len(b.Nodes) == 0 {
		opts = append(opts, aspen.Bootstrap())
	}
	addr := b.addressFactory().Next()
	db, err := aspen.Open(context.TODO(), dir, addr, b.peerAddresses, append(b.DefaultOptions, opts...)...)
	if err != nil {
		return nil, err
	}
	b.Nodes[db.HostID()] = NodeInfo{
		Addr: addr,
		Dir:  dir,
		DB:   db,
	}
	b.peerAddresses = append(b.peerAddresses, addr)
	return db, nil
}

func (b *Builder) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, ni := range b.Nodes {
		c.Exec(ni.DB.Close)
	}
	c.Exec(b.Cleanup)
	return c.Error()
}

func (b *Builder) Cleanup() error {
	if !b.memBacked {
		return os.RemoveAll(b.Dir())
	}
	return nil
}
