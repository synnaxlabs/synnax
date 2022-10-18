package testutil

import (
	"bytes"
	"encoding/binary"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

func Marshal[T types.Numeric](values []T) []byte {
	buf := new(bytes.Buffer)
	for _, v := range values {
		Expect(binary.Write(buf, binary.BigEndian, v)).To(Succeed())
	}
	return buf.Bytes()
}

func MarshalTimeStamps(values []telem.TimeStamp) []byte {
	buf := new(bytes.Buffer)
	for _, v := range values {
		Expect(binary.Write(buf, binary.BigEndian, int64(v))).To(Succeed())
	}
	return buf.Bytes()
}
