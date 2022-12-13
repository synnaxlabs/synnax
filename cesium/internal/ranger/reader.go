package ranger

import (
	"github.com/synnaxlabs/x/telem"
	"io"
)

// Reader is a readable range of telemetry within the DB.
type Reader struct {
	ptr pointer
	io.ReaderAt
	io.Closer
}

func (r *Reader) Len() int64 { return int64(r.ptr.length) }

// Range returns the time interval occupied by the range.
func (r *Reader) Range() telem.TimeRange { return r.ptr.TimeRange }
