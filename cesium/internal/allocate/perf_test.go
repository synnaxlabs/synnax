package allocate_test

import (
	"fmt"
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	"sync"
)

type perfVars struct {
	nRoutines     int
	nBatch        int
	allocPerBatch int
	key           func(i int) int
	size          func(k int) int
	experiment    alamos.Experiment
}

type allocateItem struct {
	key  int
	size int
}

func (a allocateItem) Key() int {
	return a.key
}

func (a allocateItem) Size() int {
	return a.size
}

var _ = Describe("Perf", func() {
	var (
		a allocate.Allocator[int, int, allocate.Item[int]]
	)
	Describe("Incrementing Allocation Strains", Serial, func() {
		p := alamos.NewParametrize(alamos.IterVars([]perfVars{
			{
				nRoutines:     1,
				nBatch:        1,
				allocPerBatch: 1,
				key: func(i int) int {
					return i
				},
				size: func(k int) int {
					return k
				},
				experiment: alamos.New("allocate-1"),
			},
			{
				nRoutines:     3,
				nBatch:        5,
				allocPerBatch: 9,
				key: func(i int) int {
					return 1
				},
				size: func(k int) int {
					return 1
				},
				experiment: alamos.New("allocate-2"),
			},
			{
				nRoutines:     3,
				nBatch:        5,
				allocPerBatch: 22,
				key: func(i int) int {
					return i
				},
				size: func(k int) int {
					return k
				},
				experiment: alamos.New("allocate-2"),
			},
			{
				nRoutines:     10,
				nBatch:        20,
				allocPerBatch: 100,
				// Returns a random integer between 1 and 10
				key: func(i int) int {
					return 1 + (i % 10)
				},
				// Returns a random integer between 1 and 100
				size: func(k int) int {
					return 1 + (k % 100)
				},
				experiment: alamos.New("allocate-3"),
			},
		},
		))
		p.Template(func(i int, values perfVars) {
			It(fmt.Sprintf("allocation - nr %d - nb %d  - abp %d",
				values.nRoutines, values.nBatch, values.allocPerBatch), func() {
				a = allocate.New[int, int, allocate.Item[int]](&allocate.NextDescriptorInt{}, allocate.Config{
					Experiment: values.experiment,
				})
				wg := sync.WaitGroup{}
				wg.Add(values.nRoutines)
				for i := 0; i < values.nRoutines; i++ {
					go func() {
						defer wg.Done()
						for j := 0; j < values.nBatch; j++ {
							var items []allocate.Item[int]
							for k := 0; k < values.allocPerBatch; k++ {
								items = append(items, allocateItem{
									key:  values.key(k),
									size: values.size(values.key(k)),
								})
							}
							a.Allocate(items...)
						}
					}()
				}
				wg.Wait()
			})
		})
		p.Construct()
	})
})
