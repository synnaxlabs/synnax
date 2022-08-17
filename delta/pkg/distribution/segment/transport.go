package segment

import (
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
)

type Transport interface {
	Iterator() iterator.Transport
	Writer() writer.Transport
}
