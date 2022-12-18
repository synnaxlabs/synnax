package ranger

import (
	"github.com/synnaxlabs/x/telem"
	"io"
)

// Reader is a readable range of telemetry within the DB implementing the io.ReaderAt
// and io.Closer interfaces.
type Reader struct {
	ptr pointer
	io.ReaderAt
	io.Closer
}

// Len returns the number of bytes in the entire range.
func (r *Reader) Len() int64 { return int64(r.ptr.length) }

// Range returns the time interval occupied by the range.
func (r *Reader) Range() telem.TimeRange { return r.ptr.TimeRange }
