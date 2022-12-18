package ranger

import (
	"github.com/synnaxlabs/x/telem"
)

// IteratorConfig is the configuration for opening a new iterator.
type IteratorConfig struct {
	// Bounds represent the interval of time that the iterator will be able to access.
	// Any ranges whose Bounds overlap with the iterator's Bounds will be accessible.
	// A zero span range is valid, but is relatively useless.
	// [REQUIRED]
	Bounds telem.TimeRange
}

// IterRange generates an IteratorConfig that iterates over the provided time range.
func IterRange(tr telem.TimeRange) IteratorConfig { return IteratorConfig{Bounds: tr} }

// Iterator iterates over the telemetry ranges of a DB in time order. Iterator does
// not read any of the underlying data of a range, but instead provides a means to access
// it via calls to Iterator.NewReader.
//
// Iterator is not safe for concurrent use, but it is safe to have multiple iterators over
// the same DB.
//
// It's important to not that an Iterator does NOT iterator over a snapshot of the DB,
// and is not isolated from any writes that may be committed during the iterators lifetime.
// This means that the position of an iterator may shift unexpectedly. There are plans
// to implement MVCC in the future, but until then you have been warned.
type Iterator struct {
	// position stores the current position of the iterator in the idx. NOTE: At the
	// moment, this position may not hold a consistent reference to the same value
	// if the idx is modified during iteration.
	position int
	// idx is the index that the iterator is iterating over.
	idx *index
	// bounds stores the bounds of the iterator.
	bounds telem.TimeRange
	// value stores the current value of the iterator. This value is only valid if
	// the iterator is valid.
	value pointer
	// valid stores whether the iterator is currently valid.
	valid         bool
	readerFactory func(ptr pointer) (*Reader, error)
}

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (it *Iterator) SetBounds(bounds telem.TimeRange) {
	it.bounds = bounds
	it.valid = false
}

// SeekFirst seeks to the first range in the iterator's bounds. If no such range exists,
// SeekFirst returns false.
func (it *Iterator) SeekFirst() bool { return it.SeekGE(it.bounds.Start) }

// SeekLast seeks to the last range in the iterator's bounds. If no such range exists,
// SeekLast returns false.
func (it *Iterator) SeekLast() bool { return it.SeekLE(it.bounds.End - 1) }

// SeekLE seeks to the range whose Range contain the provided timestamp. If no such range
// exists, SeekLE seeks to the closes range whose ending timestamp is less than the provided
// timestamp. If no such range exists, SeekLE returns false.
func (it *Iterator) SeekLE(stamp telem.TimeStamp) bool {
	it.valid = true
	it.position = it.idx.searchLE(stamp)
	return it.reload()
}

// SeekGE seeks to the range whose Range contain the provided timestamp. If no such range
// exists, SeekGE seeks to the closes range whose starting timestamp is greater than the
// provided timestamp. If no such range exists, SeekGE returns false.
func (it *Iterator) SeekGE(stamp telem.TimeStamp) bool {
	it.valid = true
	it.position = it.idx.searchGE(stamp)
	return it.reload()
}

// Next advances the iterator to the next range. If the iterator has been exhausted, Next
// returns false.
func (it *Iterator) Next() bool {
	if !it.valid {
		return false
	}
	it.position++
	return it.reload()
}

// Prev advances the iterator to the previous range. If the iterator has been exhausted,
// Prev returns false.
func (it *Iterator) Prev() bool {
	if !it.valid {
		return false
	}
	it.position--
	return it.reload()
}

// Valid returns true if the iterator is currently pointing to a valid range and has
// not accumulated an error. Returns false otherwise.
func (it *Iterator) Valid() bool { return it.valid }

// Range returns the time interval occupied by current range.
func (it *Iterator) Range() telem.TimeRange { return it.value.TimeRange }

// NewReader returns a new Reader that can be used to read telemetry from the current
// range. The returned Reader is not safe for concurrent use, but it is safe to have
// multiple Readers open over the same range.
func (it *Iterator) NewReader() (*Reader, error) { return it.readerFactory(it.value) }

// Len returns the number of bytes occupied by the telemetry in the current range.
func (it *Iterator) Len() int64 { return int64(it.value.length) }

// Close closes the iterator.
func (it *Iterator) Close() error { return nil }

func (it *Iterator) reload() bool {
	if it.position == -1 {
		it.valid = false
		return it.valid
	}
	ptr, ok := it.idx.get(it.position)
	if !ok || !ptr.OverlapsWith(it.bounds) {
		it.valid = false
		// it's important that we return here, so we don't clear the current value
		// of the iterator.
		return it.valid
	}
	it.value = ptr
	return it.valid
}
