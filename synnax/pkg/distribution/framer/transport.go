package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

type Transport interface {
	Iterator() iterator.Transport
	Writer() writer.Transport
}
