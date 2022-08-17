package alamos_test

import (
	"fmt"
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
)

type paramVars struct {
	counter int
}

type paramConfig struct {
	count int
	next  int
}

func (pc *paramConfig) Next() (paramVars, error) {
	pc.next++
	if pc.next > pc.count {
		return paramVars{}, io.EOF
	}
	return paramVars{counter: pc.next}, nil
}

var _ = Describe("Parametrize", func() {
	Describe("Basic", func() {
		cfg := &paramConfig{next: -1, count: 7}
		p := alamos.NewParametrize[paramVars](cfg)
		p.Template(func(i int, values paramVars) {
			It(fmt.Sprintf("Should increment the counter correctly - %v", values.counter), func() {
				Expect(values.counter).To(Equal(i))
			})
		})
		p.Construct()
	})
})
