package channel

import (
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/x/telem"
	"strconv"
)

type Key uint16

// String implements the fmt.Stringer interface.
func (k Key) String() string { return strconv.Itoa(int(k)) }

type Channel struct {
	Key     Key
	Rate    telem.Rate
	Density telem.Density
}

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key }

// SetOptions implements the gorp.Entry interface.
func (c Channel) SetOptions() []interface{} { return []interface{}{pebble.NoSync} }
