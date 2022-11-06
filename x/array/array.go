package array

type Array[V any] interface {
	Get(index int) V
	Append(values ...V)
	Len() int
}

type array[V any] struct {
	data []V
}

func (a *array[V]) Get(index int) V {
	return a.data[index]
}

func (a *array[V]) Append(values ...V) {
	a.data = append(a.data, values...)
}

func (a *array[V]) Len() int {
	return len(a.data)
}

func Wrap[V any](data []V) Array[V] {
	return &array[V]{data: data}
}
