package cache

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"sync"
)

// Channel is a wrapper around ChannelEngine that caches the results of
// channel related operations.
type Channel struct {
	core.ChannelEngine
	cache map[core.ChannelKey]core.Channel
	mu    sync.RWMutex
}

// WrapChannelEngine wraps a ChannelEngine with a cache.
func WrapChannelEngine(engine core.ChannelEngine) *Channel {
	return &Channel{
		ChannelEngine: engine,
		cache:         make(map[core.ChannelKey]core.Channel),
	}
}

var _ core.ChannelEngine = (*Channel)(nil)

// GetChannel implements core.ChannelEngine.
func (cr *Channel) GetChannel(key core.ChannelKey) (core.Channel, error) {
	cr.mu.RLock()
	if ch, ok := cr.cache[key]; ok {
		return ch, nil
	}
	cr.mu.RUnlock()
	ch, err := cr.ChannelEngine.GetChannel(key)
	if err != nil {
		return core.Channel{}, err
	}
	cr.cache[key] = ch
	return ch, nil
}

// GetChannels implements core.ChannelEngine.
func (cr *Channel) GetChannels(keys ...core.ChannelKey) ([]core.Channel, error) {
	cr.mu.RLock()
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
	cr.mu.RUnlock()

	// get the channels from the wrapped reader
	chs, err := cr.ChannelEngine.GetChannels(getKeys...)
	if err != nil {
		return nil, err
	}
	// add the channels to the cache
	cr.mu.Lock()
	for _, ch := range chs {
		cr.cache[ch.Key] = ch
	}
	cr.mu.Unlock()
	// append the channels to the results
	return append(results, chs...), nil
}

// SetChannel implements core.ChannelEngine.
func (cr *Channel) SetChannel(ch core.Channel) error {
	cr.mu.Lock()
	cr.cache[ch.Key] = ch
	cr.mu.Unlock()
	return cr.ChannelEngine.SetChannel(ch)
}

// SetChannels implements core.ChannelEngine.
func (cr *Channel) SetChannels(chs ...core.Channel) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	for _, ch := range chs {
		cr.cache[ch.Key] = ch
	}
	return cr.ChannelEngine.SetChannels(chs...)
}

// DeleteChannel implements core.ChannelEngine.
func (cr *Channel) DeleteChannel(key core.ChannelKey) error {
	cr.mu.Lock()
	delete(cr.cache, key)
	cr.mu.Unlock()
	return cr.ChannelEngine.DeleteChannel(key)
}

// DeleteChannels implements core.ChannelEngine.
func (cr *Channel) DeleteChannels(keys ...core.ChannelKey) error {
	cr.mu.Lock()
	for _, key := range keys {
		delete(cr.cache, key)
	}
	cr.mu.Unlock()
	return cr.ChannelEngine.DeleteChannels(keys...)
}

// ChannelsExist implements core.ChannelEngine.
func (cr *Channel) ChannelsExist(keys ...core.ChannelKey) (bool, error) {
	cr.mu.RLock()
	var checkDB []core.ChannelKey
	for _, key := range keys {
		if _, ok := cr.cache[key]; !ok {
			checkDB = append(checkDB, key)
			break
		}
	}
	cr.mu.RUnlock()
	if len(checkDB) == 0 {
		return true, nil
	}
	return cr.ChannelEngine.ChannelsExist(checkDB...)
}
