package kv

import (
	"bytes"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/errutil"
)

func CompositeKey(elems ...interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	c := errutil.NewCatch()
	for _, e := range elems {
		switch e.(type) {
		case string:
			c.Exec(func() error {
				_, err := b.WriteString(e.(string))
				return err
			})
		default:
			c.Exec(func() error { return binary.Write(b, e) })
		}
	}
	return b.Bytes(), nil
}
