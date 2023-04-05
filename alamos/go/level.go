package alamos

type Level uint8

const (
	DebugLevel Level = iota + 1
	InfoLevel
)

type Filter func(level Level, key string) bool

func CompoundFilter(filters ...Filter) Filter {
	return func(level Level, key string) bool {
		for _, f := range filters {
			if f(level, key) {
				return true
			}
		}
		return false
	}
}

func ThresholdFilter(level Level) Filter {
	return func(l Level, _ string) bool {
		return l < level
	}
}
