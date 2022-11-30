package ranger

import "github.com/synnaxlabs/x/telem"

const pointerByteSize = 26

// pointer is a reference to a telemetry blob occupying a particular time range.
type pointer struct {
	// Bounds is the time interval occupied by the range. This interval is
	// guaranteed to be unique i.e.it won't overlap with any other range within the DB.
	// bounds follows the behavior of telem.TimeRange in that the starting point is inclusive,
	// while the ending point is exclusive. If two ranges share a common start and end point,
	// they are considered continuous.
	telem.TimeRange
	// fileKey
	fileKey uint16
	// offset is the offset of the range within the file.
	offset uint32
	// length is the length of the range within the file.
	length uint32
}
