package main

import (
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/computron/math"
	"github.com/synnaxlabs/x/telem"
)

func main() {
	s1 := telem.NewSeriesV[int32](1, 2, 3, 4, 5)
	arr1, err := math.New(s1)
	if err != nil {
		logrus.Error(err)
	}
	s2 := telem.NewSeriesV[int32](1, 2, 3, 4, 5)
	arr2, err := math.New(s2)
	if err != nil {
		logrus.Error(err)
	}
	res, err := math.Exec("result = arr1 + arr2", map[string]interface{}{
		"arr1": arr1,
		"arr2": arr2,
	}, nil)
	logrus.Info(telem.Unmarshal[int32](res))
}
