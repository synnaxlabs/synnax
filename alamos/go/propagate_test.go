package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
)

type mockCarrier struct {
	data map[string]string
}

var _ alamos.TraceCarrier = mockCarrier{}

func (m mockCarrier) Set(key, value string) {
	m.data[key] = value
}

func (m mockCarrier) Get(key string) string {
	return m.data[key]
}

func (m mockCarrier) Keys() []string {
	return lo.Keys(m.data)
}

var _ = Describe("Propagate", func() {
})
