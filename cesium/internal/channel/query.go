package channel

import "github.com/arya-analytics/x/query"

const keysOptKey query.OptionKey = "cks"

// SetKeys sets the channel keys on the provided query. Calling SetKeys will
// override and previous keys set on the query. Use GetKeys to retrieve the keys.
func SetKeys(q query.Query, keys ...Key) { q.Set(keysOptKey, keys) }

// GetKeys retrieves the channel keys on the provided query. GetKeys will panic
// if SetKeys was not called on teh query.
func GetKeys(q query.Query) []Key { return q.GetRequired(keysOptKey).([]Key) }

// ExtractKeys extracts the keys from the provided channel slice.
func ExtractKeys(channels []Channel) []Key {
	keys := make([]Key, len(channels))
	for i, ch := range channels {
		keys[i] = ch.Key
	}
	return keys
}
