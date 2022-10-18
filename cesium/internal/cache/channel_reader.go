package cache

import "github.com/synnaxlabs/cesium/internal/core"

type Channel struct {
	core.ChannelReader
	core.ChannelWriter
	cache map[core.ChannelKey]core.Channel
}

func NewChannel(reader core.ChannelReader, writer core.ChannelWriter) *Channel {
	return &Channel{
		ChannelReader: reader,
		ChannelWriter: writer,
		cache:         make(map[core.ChannelKey]core.Channel),
	}
}

var (
	_ core.ChannelReader = (*Channel)(nil)
	_ core.ChannelWriter = (*Channel)(nil)
)

func (cr *Channel) GetChannel(key core.ChannelKey) (core.Channel, error) {
	if ch, ok := cr.cache[key]; ok {
		return ch, nil
	}
	ch, err := cr.ChannelReader.GetChannel(key)
	if err != nil {
		return core.Channel{}, err
	}
	cr.cache[key] = ch
	return ch, nil
}

func (cr *Channel) GetChannels(keys ...core.ChannelKey) ([]core.Channel, error) {
	var getKeys []core.ChannelKey
	results := make([]core.Channel, 0, len(keys))
	// iterate over keys and check if they are in the cache
	for _, key := range keys {
		ch, ok := cr.cache[key]
		if !ok {
			getKeys = append(getKeys, key)
		} else {
			results = append(results, ch)
		}
	}
	// get the channels from the wrapped reader
	chs, err := cr.ChannelReader.GetChannels(getKeys...)
	if err != nil {
		return nil, err
	}
	// add the channels to the cache
	for _, ch := range chs {
		cr.cache[ch.Key] = ch
	}
	// append the channels to the results
	return append(results, chs...), nil
}

func (cr *Channel) SetChannel(ch core.Channel) error {
	cr.cache[ch.Key] = ch
	return cr.ChannelWriter.SetChannel(ch)
}

func (cr *Channel) SetChannels(chs ...core.Channel) error {
	for _, ch := range chs {
		cr.cache[ch.Key] = ch
	}
	return cr.ChannelWriter.SetChannels(chs...)
}

func (cr *Channel) DeleteChannel(key core.ChannelKey) error {
	delete(cr.cache, key)
	return cr.ChannelWriter.DeleteChannel(key)
}

func (cr *Channel) DeleteChannels(keys ...core.ChannelKey) error {
	for _, key := range keys {
		delete(cr.cache, key)
	}
	return cr.ChannelWriter.DeleteChannels(keys...)
}

func (cr *Channel) ChannelsExist(keys ...core.ChannelKey) (bool, error) {
	var checkDB []core.ChannelKey
	for _, key := range keys {
		if _, ok := cr.cache[key]; !ok {
			checkDB = append(checkDB, key)
			break
		}
	}
	if len(checkDB) == 0 {
		return true, nil
	}
	return cr.ChannelReader.ChannelsExist(checkDB...)
}
