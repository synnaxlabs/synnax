package kv

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
)

type Channel struct {
	db *gorp.DB
}

func NewChannel(kve kv.DB) *Channel { return &Channel{db: gorp.Wrap(kve)} }

func (c *Channel) Get(keys ...channel.Key) (res []channel.Channel, err error) {
	return res, gorp.NewRetrieve[channel.Key, channel.Channel]().Entries(&res).WhereKeys(keys...).Exec(c.db)
}

func (c *Channel) Set(ch channel.Channel) error {
	return gorp.NewCreate[channel.Key, channel.Channel]().Entry(&ch).Exec(c.db)
}

func (c *Channel) Exists(key channel.Key) (bool, error) {
	var ch channel.Channel
	return ch.Key != 0, gorp.NewRetrieve[channel.Key, channel.Channel]().Entry(&ch).WhereKeys(key).Exec(c.db)
}
