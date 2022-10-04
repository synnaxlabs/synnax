package freighter_test

import (
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter/ferrors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type request struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type response struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

var myCustomError = ferrors.Typed(roacherrors.New("my custom error"), "myCustomError")

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Freighter Suite")
}

var _ = BeforeSuite(func() {
	ferrors.Register(
		"myCustomError",
		func(err error) string {
			return err.Error()
		},
		func(s string) error {
			if s == "my custom error" {
				return myCustomError
			}
			panic("unknown error")
		},
	)
})
