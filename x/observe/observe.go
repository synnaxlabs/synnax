package observe

// Observable is an interface that represents an entity whose state can be observed.
type Observable[T any] interface {
	// OnChange is called when the state of the observable changes.
	OnChange(handler func(T))
}

// Observer is an interface that can notify subscribers of changes to an observable.
type Observer[T any] interface {
	Observable[T]
	// Notify notifies all subscribers of the value.
	Notify(T)
}

type base[T any] struct {
	options  *options
	handlers []func(T)
}

// New creates a new observer with the given options.
func New[T any](opts ...Option) Observer[T] {
	return &base[T]{options: newOptions(opts...)}
}

// OnChange implements the Observable interface.
func (b *base[T]) OnChange(handler func(T)) {
	b.handlers = append(b.handlers, handler)
}

// Notify implements the Observer interface.
func (b *base[T]) Notify(v T) {
	for _, handler := range b.handlers {
		handler(v)
	}
}
