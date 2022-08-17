package address

import (
	"strconv"
	"sync"
)

type Factory struct {
	Host      string
	PortStart int
	mu        sync.Mutex
	addresses []Address
}

func (f *Factory) Next() Address {
	f.mu.Lock()
	defer f.mu.Unlock()
	addr := Address(f.Host + ":" + strconv.Itoa(f.PortStart+len(f.addresses)))
	f.addresses = append(f.addresses, addr)
	return addr
}

func (f *Factory) NextN(n int) (addresses []Address) {
	for i := 0; i < n; i++ {
		addresses = append(addresses, f.Next())
	}
	return addresses
}

func NewLocalFactory(portStart int) *Factory {
	return &Factory{Host: "localhost", PortStart: portStart}
}
