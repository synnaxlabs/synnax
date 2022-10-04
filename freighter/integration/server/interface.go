package server

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"strconv"
	"strings"
)

type Message struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type (
	Server       = freighter.StreamServer[Message, Message]
	Client       = freighter.StreamClient[Message, Message]
	ServerStream = freighter.ServerStream[Message, Message]
	ClientStream = freighter.ClientStream[Message, Message]
)

type TestError struct {
	Code    int
	Message string
}

func (t TestError) Error() string {
	return t.Message
}

func (t TestError) FreighterType() ferrors.Type {
	return "integration.error"
}

func encodeTestError(err error) string {
	te, ok := err.(TestError)
	if !ok {
		panic("unexpected error type")
	}
	return strconv.Itoa(te.Code) + "," + te.Message
}

func decodeTestError(s string) error {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return errors.New("unexpected error format")
	}
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	return TestError{
		Code:    code,
		Message: parts[1],
	}
}

func init() {
	ferrors.Register("integration.error", encodeTestError, decodeTestError)
}
