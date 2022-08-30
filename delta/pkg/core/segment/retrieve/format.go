package retrieve

type Format[V Value] interface {
	ValueRange[V] | SampleRange[V] | StampValueRange[V]
}
