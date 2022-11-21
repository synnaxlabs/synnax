package ranger

import (
	"github.com/synnaxlabs/x/telem"
)

type IteratorConfig struct {
	// Bounds represent the interval of time that the iterator will be able to access.
	// Any ranges whose Bounds overlap with the iterator's Bounds will be accessible.
	Bounds telem.TimeRange
}

// Iterator iterates over a DBs telemetry ranges in time order. Iterator does not directly
// read any telemetry ranges, but instead provides a means to access the ranges themselves.
// The actual telemetry ranges can be read by calling NewReader. Iterator is not safe for
// concurrent use, but it is safe to have multiple iterators open over the same DB.
type Iterator struct {
	position      int
	idx           *index
	bounds        telem.TimeRange
	_value        *pointer
	valid         bool
	readerFactory func(ptr *pointer) (*Reader, error)
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
func (it *Iterator) SeekLast() bool { return it.SeekLE(it.bounds.End) }

// SeekLE seeks to the range whose Range contain the provided timestamp. If no such range
// exists, SeekLE seeks to the closes range whose ending timestamp is less than the provided
// timestamp. If no such range exists, SeekLE returns false.
func (it *Iterator) SeekLE(stamp telem.TimeStamp) bool {
	return it.seek(func() (int, *pointer, bool) {
		i, ptr := it.idx.searchLE(stamp)
		return i, &ptr, i != -1 && ptr.bounds.OverlapsWith(it.bounds)
	})
}

// SeekGE seeks to the range whose Range contain the provided timestamp. If no such range
// exists, SeekGE seeks to the closes range whose starting timestamp is greater than the
// provided timestamp. If no such range exists, SeekGE returns false.
func (it *Iterator) SeekGE(stamp telem.TimeStamp) bool {
	return it.seek(func() (int, *pointer, bool) {
		i, ptr := it.idx.searchGE(stamp)
		return i, &ptr, i != -1 && ptr.bounds.OverlapsWith(it.bounds)
	})
}

func (it *Iterator) seek(f func() (int, *pointer, bool)) bool {
	i, ptr, valid := f()
	it.valid = valid
	it.position = i
	it._value = ptr
	return it.valid
}

// Next advances the iterator to the next range. If the iterator has been exhausted, Next
// returns false.
func (it *Iterator) Next() bool {
	if !it.valid {
		return false
	}
	it.position++
	ptr, ok := it.idx.get(it.position)
	if !ok || !ptr.bounds.OverlapsWith(it.bounds) {
		it.valid = false
	}
	it._value = &ptr
	return it.valid
}

// Prev advances the iterator to the previous range. If the iterator has been exhausted,
// Prev returns false.
func (it *Iterator) Prev() bool {
	if !it.valid {
		return false
	}
	it.position--
	ptr, ok := it.idx.get(it.position)
	if !ok || !ptr.bounds.OverlapsWith(it.bounds) {
		it.valid = false
	}
	it._value = &ptr
	return it.valid
}

// Valid returns true if the iterator is currently pointing to a valid range and has
// not accumulated an error. Returns false otherwise.
func (it *Iterator) Valid() bool { return it.valid }

// Range returns the time interval occupied by current range.
func (it *Iterator) Range() telem.TimeRange { return it._value.bounds }

// NewReader returns a new Reader that can be used to read telemetry ranges from the current
// range. The returned Reader is not safe for concurrent use, but it is safe to have
// multiple Readers open over the same range.
func (it *Iterator) NewReader() (*Reader, error) {
	if !it.Valid() {
		panic("cannot open a reader on an invalidated iterator")
	}
	return it.readerFactory(it._value)
}
