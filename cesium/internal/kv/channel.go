package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

// channelEngine is a key-value backed implementation of the core.ChannelWriter interface.
type channelEngine struct {
	db      *gorp.DB
	counter kv.PersistedCounter
}

func openChannelEngine(db kv.DB, counterKey string) (*channelEngine, error) {
	gorpDB := gorp.Wrap(db)
	c, err := kv.OpenCounter(gorpDB, []byte(counterKey))
	return &channelEngine{db: gorpDB, counter: *c}, err
}

var _ core.ChannelWriter = (*channelEngine)(nil)

// GetChannel implements the core.ChannelWriter interface.
func (c *channelEngine) GetChannel(key core.ChannelKey) (ch core.Channel, err error) {
	return ch, gorp.NewRetrieve[core.ChannelKey, core.Channel]().Entry(&ch).WhereKeys(key).Exec(c.db)
}

// GetChannels implements the core.ChannelWriter interface.
func (c *channelEngine) GetChannels(keys ...core.ChannelKey) (res []core.Channel, err error) {
	return res, gorp.NewRetrieve[core.ChannelKey, core.Channel]().Entries(&res).WhereKeys(keys...).Exec(c.db)
}

// SetChannel implements the core.ChannelWriter interface.
func (c *channelEngine) SetChannel(ch core.Channel) error {
	return gorp.NewCreate[core.ChannelKey, core.Channel]().Entry(&ch).Exec(c.db)
}

// SetChannels implements the core.ChannelWriter interface.
func (c *channelEngine) SetChannels(chs ...core.Channel) error {
	return gorp.NewCreate[core.ChannelKey, core.Channel]().Entries(&chs).Exec(c.db)
}

// DeleteChannel implements the core.ChannelWriter interface.
func (c *channelEngine) DeleteChannel(key core.ChannelKey) error {
	return gorp.NewDelete[core.ChannelKey, core.Channel]().WhereKeys(key).Exec(c.db)
}

// DeleteChannels implements the core.ChannelWriter interface.
func (c *channelEngine) DeleteChannels(keys ...core.ChannelKey) error {
	return gorp.NewDelete[core.ChannelKey, core.Channel]().WhereKeys(keys...).Exec(c.db)
}

// ChannelsExist implements the core.ChannelWriter interface.
func (c *channelEngine) ChannelsExist(keys ...core.ChannelKey) (bool, error) {
	return gorp.NewRetrieve[core.ChannelKey, core.Channel]().WhereKeys(keys...).Exists(c.db)
}

// NextChannelKey implements the core.ChannelWriter interface.
func (c *channelEngine) NextChannelKey() (core.ChannelKey, error) {
	n, err := c.counter.Add(1)
	return core.ChannelKey(n), err
}
