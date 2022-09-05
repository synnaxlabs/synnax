package kv

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
)

type ChannelService struct {
	db *gorp.DB
}

func NewChannelService(kve kv.DB) *ChannelService { return &ChannelService{db: gorp.Wrap(kve)} }

func (c *ChannelService) Get(keys ...channel.Key) (res []channel.Channel, err error) {
	return res, gorp.NewRetrieve[channel.Key, channel.Channel]().Entries(&res).WhereKeys(keys...).Exec(c.db)
}

func (c *ChannelService) Set(ch channel.Channel) error {
	return gorp.NewCreate[channel.Key, channel.Channel]().Entry(&ch).Exec(c.db)
}

func (c *ChannelService) Exists(key channel.Key) (bool, error) {
	return gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(key).Exists(c.db)
}
