package sync

import "sync"

// Map is a generic type that wraps a sync.Map.
type Map[T any, U any] struct{ m sync.Map }

func (m *Map[T, U]) Load(key T) (value U, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return
	}
	value = v.(U)
	return
}

func (m *Map[T, U]) Store(key T, value U) {
	m.m.Store(key, value)
}

func (m *Map[T, U]) Delete(key T) {
	m.m.Delete(key)
}

func (m *Map[T, U]) Range(f func(key T, value U) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(T), value.(U))
	})
}

func (m *Map[T, U]) LoadOrStore(key T, value U) (actual U, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	actual = v.(U)
	return
}

func (m *Map[T, U]) LoadAndDelete(key T) (value U, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	value = v.(U)
	return
}
