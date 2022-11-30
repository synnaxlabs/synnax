package mock

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
)

type ChannelService struct {
	DB *gorp.DB
}

func (c *ChannelService) Create(channel *channel.Channel) error {
	return gorp.NewCreate[channel.Key, channel.Channel]().Entry(channel).Exec(c.DB)
}

func (c *ChannelService) CreateMany(channels *[]channel.Channel) error {
	return gorp.NewCreate[channel.Key, channel.Channel]().Entries(channels).Exec(c.DB)
}

func (c *ChannelService) NewRetrieve() channel.Retrieve {
	return channel.NewRetrieve(c.DB)
}
