package cesium

import (
	"github.com/arya-analytics/x/binary"
)

func MarshalFloat64(data []float64) []byte {
	return marshal(data)
}

func marshal(data interface{}) []byte {
	b, err := binary.Marshal(data)
	if err != nil {
		panic(err)
	}
	return b
}
