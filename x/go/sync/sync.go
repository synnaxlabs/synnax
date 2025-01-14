package sync

// Package sync provides generic synchronization primitives that wrap the standard sync package.
import "sync"

// Map is a generic type that wraps a sync.Map. It provides a type-safe concurrent map
// implementation where the key type is T and the value type is U. All operations
// are safe for concurrent use by multiple goroutines.
type Map[T any, U any] struct{ m sync.Map }

// Load returns the value stored in the map for a key, or zero value if no
// value is present. The ok result indicates whether value was found in the map.
func (m *Map[T, U]) Load(key T) (value U, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return
	}
	value = v.(U)
	return
}

// Store sets the value for a key.
func (m *Map[T, U]) Store(key T, value U) {
	m.m.Store(key, value)
}

// Delete removes the value for a key.
func (m *Map[T, U]) Delete(key T) {
	m.m.Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *Map[T, U]) Range(f func(key T, value U) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(T), value.(U))
	})
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *Map[T, U]) LoadOrStore(key T, value U) (actual U, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	actual = v.(U)
	return
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map[T, U]) LoadAndDelete(key T) (value U, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	value = v.(U)
	return
}

// Clear removes all key-value pairs from the map.
func (m *Map[T, U]) Clear() {
	m.m.Range(func(key, _ interface{}) bool {
		m.m.Delete(key)
		return true
	})
}

// CompareAndDelete deletes the entry for key if its value equals old.
// The deleted result reports whether the entry was deleted.
func (m *Map[T, U]) CompareAndDelete(key T, old U) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}

// Swap stores a value for a key and returns the previous value if any.
// The loaded result reports whether there was a previous value.
func (m *Map[T, U]) Swap(key T, value U) (previous U, loaded bool) {
	v, loaded := m.m.Swap(key, value)
	if !loaded {
		return
	}
	previous = v.(U)
	return
}
