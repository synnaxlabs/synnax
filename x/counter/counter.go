package counter

type Uint16 interface {
	Add(delta ...uint16) uint16
	Value() uint16
}

type Uint16Error interface {
	Uint16
	Error
}

type Int32 interface {
	Add(delta ...int32) int32
	Value() int32
}

type Int64 interface {
	Add(delta ...int64) int64
	Value() int64
}

type Error interface {
	Error() error
}
