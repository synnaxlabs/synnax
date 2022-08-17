package rand

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func MapKey[K comparable, V any](m map[K]V) (key K) {
	l := len(m)
	if l == 0 {
		return key
	}
	i, ri := 0, rand.Intn(l)
	for k, _ := range m {
		if i == ri {
			key = k
			break
		}
		i++
	}
	return key
}

func MapValue[K comparable, V any](m map[K]V) V {
	return m[MapKey[K, V](m)]
}

func MapElem[K comparable, V any](m map[K]V) (K, V) {
	k := MapKey[K, V](m)
	return k, m[k]
}

func SubMap[K comparable, V any](m map[K]V, n int) map[K]V {
	om := make(map[K]V)
	i := 0
	for {
		k, v := MapElem[K, V](m)
		_, ok := om[k]
		if !ok {
			om[k] = v
			i++
		}
		if i >= n {
			break
		}
	}
	return om
}

func Elem[V any](options ...V) V {
	return Slice[V](options)
}

func Slice[V any](slice []V) V {
	return slice[rand.Intn(len(slice))]
}

func SubSlice[V comparable](slice []V, n int) []V {
	if n >= len(slice) {
		return slice
	}
	os := make([]V, n)
	for i := 0; i < n; i++ {
		if i != 0 {
			for v := Slice[V](slice); v == os[i-1]; v = Slice[V](slice) {
				os[i] = v
			}
		} else {
			os[i] = Slice[V](slice)
		}
	}
	return os
}
