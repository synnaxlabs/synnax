package main

import (
	"encoding/binary"
	"github.com/sirupsen/logrus"
)

func main() {
	b := []byte{123, 202, 242, 8, 0, 0, 0, 0}
	logrus.Info(binary.LittleEndian.Uint64(b))

}
