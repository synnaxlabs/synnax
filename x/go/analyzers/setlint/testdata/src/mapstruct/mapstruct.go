package mapstruct

type MyKey string

type TypeDecl = map[string]struct{} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`

var Global map[int]struct{} // want `map\[int\]struct\{\} can be replaced with set\.Set\[int\]`

type MyStruct struct {
	Field map[MyKey]struct{} // want `map\[MyKey\]struct\{\} can be replaced with set\.Set\[MyKey\]`
}

func TakesSet(s map[string]struct{}) {} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`

func ReturnsSet() map[string]struct{} { // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`
	return nil
}

func LocalVar() {
	m := make(map[int]struct{}) // want `map\[int\]struct\{\} can be replaced with set\.Set\[int\]`
	_ = m
	var n map[string]struct{} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`
	_ = n
}
