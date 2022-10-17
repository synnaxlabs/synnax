package kv

import (
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

type channelService struct {
	db *gorp.DB
}

func newChannelService(kve kv.DB) *channelService { return &channelService{db: gorp.Wrap(kve)} }

func (c *channelService) Get(keys ...channel.Key) (res []channel.Channel, err error) {
	return res, gorp.NewRetrieve[channel.Key, channel.Channel]().Entries(&res).WhereKeys(keys...).Exec(c.db)
}

func (c *channelService) Set(ch channel.Channel) error {
	return gorp.NewCreate[channel.Key, channel.Channel]().Entry(&ch).Exec(c.db)
}

func (c *channelService) Exists(key channel.Key) (bool, error) {
	return gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(key).Exists(c.db)
}
