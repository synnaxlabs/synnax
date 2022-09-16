package segment

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
)

type Transport interface {
	Iterator() iterator.Transport
	Writer() writer.Transport
}
