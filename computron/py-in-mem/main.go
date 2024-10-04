package main

import (
	"fmt"
	"math/rand"
)

func genData() ([]float64, []int) {
	const size = 1000
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Float64()
	}

	indices := []int{7, 113, 835}
	for _, i := range indices {
		data[i] += 97
	}

	return data, indices
}

func main() {
	o, _ := NewOutliers("outliers", "detect")
	defer o.Close()
	data, _ := genData()
	out, _ := o.Detect(data)
	fmt.Println(out)

	return // This is a placeholder
}
