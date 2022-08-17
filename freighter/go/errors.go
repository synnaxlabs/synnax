package freighter

import (
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/cockroachdb/errors"
	"io"
)

const freighterError ferrors.Type = "freighter"

var (
	// EOF is returned when either the receiving or sending end of a Stream
	// exits normally.
	EOF = ferrors.WithType(io.EOF, freighterError)
	// StreamClosed is returned when a caller attempts to send or receive a message
	// from a stream that is already closed.
	StreamClosed = ferrors.WithType(errors.New("[freighter] - stream closed"), freighterError)
)

func encodeErr(err error) string { return err.Error() }

func decodeErr(encoded string) error {
	switch encoded {
	case EOF.Error():
		return EOF
	case StreamClosed.Error():
		return StreamClosed
	}
	panic("unknown error")
}

func init() {
	ferrors.Register(freighterError, encodeErr, decodeErr)
}
