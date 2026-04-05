package clean

type Custom struct {
	Name string
}

var StringMap map[string]int

var StructMap map[string]Custom

var SliceMap map[string][]byte

func TakesMap(m map[int]string) {}

func ReturnsMap() map[string]float64 {
	return nil
}

func LocalVars() {
	m := make(map[string]int)
	_ = m
	var n map[int]Custom
	_ = n
}
