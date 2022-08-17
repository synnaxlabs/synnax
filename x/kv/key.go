package kv

import (
	"bytes"
	"github.com/arya-analytics/x/errutil"
)

func CompositeKey(elems ...interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	cw := errutil.NewCatchWrite(b)
	for _, e := range elems {
		switch e.(type) {
		case string:
			cw.Write([]byte(e.(string)))
		default:
			cw.Write(e)
		}
	}
	return b.Bytes(), cw.Error()
}

func StaticCompositeKey(elems ...interface{}) []byte {
	b, err := CompositeKey(elems...)
	if err != nil {
		panic(err)
	}
	return b
}
