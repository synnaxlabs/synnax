package address

import (
	"fmt"
	"strings"
)

type Address string

func Newf(format string, args ...any) Address {
	return Address(fmt.Sprintf(format, args...))
}

func (a Address) String() string { return string(a) }

func (a Address) PortString() string {
	str := strings.Split(string(a), ":")
	return ":" + str[1]
}

type Addressable interface {
	Address() Address
}
