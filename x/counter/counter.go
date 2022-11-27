package counter

type Int32 interface {
	Add(delta ...int32) int32
	Value() int32
}

type Int64 interface {
	Add(delta ...int64) int64
	Value() int64
}
