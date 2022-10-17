package array

type Array[V any] interface {
	Get(index int) V
	Append(values ...V)
	Size() int
}
