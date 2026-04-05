package mapbool

type MyKey string

type TypeDecl = map[string]bool // want `map\[string\]bool can be replaced with set\.Set\[string\]`

var Global map[int]bool // want `map\[int\]bool can be replaced with set\.Set\[int\]`

type MyStruct struct {
	Field map[MyKey]bool // want `map\[MyKey\]bool can be replaced with set\.Set\[MyKey\]`
}

func TakesSet(s map[string]bool) {} // want `map\[string\]bool can be replaced with set\.Set\[string\]`

func ReturnsSet() map[string]bool { // want `map\[string\]bool can be replaced with set\.Set\[string\]`
	return nil
}

func LocalVar() {
	m := make(map[int]bool) // want `map\[int\]bool can be replaced with set\.Set\[int\]`
	_ = m
	var n map[string]bool // want `map\[string\]bool can be replaced with set\.Set\[string\]`
	_ = n
}
