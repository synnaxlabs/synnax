package core

import "github.com/synnaxlabs/cesium/internal/channel"

type ChannelService interface {
	Get(key channel.Key) (channel.Channel, error)
	GetMany(keys ...channel.Key) ([]channel.Channel, error)
	Iter(func(c *channel.Channel)) error
	Set(c channel.Channel) error
	Delete(key channel.Key) error
	Exists(keys ...channel.Key) (bool, error)
}
