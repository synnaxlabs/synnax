package core

import "github.com/synnaxlabs/cesium/internal/position"

// MDBatch writes segment metadata to disk.
type MDBatch interface {
	// Write writes the segment metadata to the writer. The writes are not durable until
	// Commit is called.
	Write([]SegmentMD) error
	// Commit commits the writes to disk.
	Commit() error
	// Close closes the writer, releasing an uncommitted writes.
	Close() error
	// NewIterator
	NewIterator(Channel) PositionIterator
	Retrieve(ChannelKey, position.Position) (SegmentMD, error)
}
