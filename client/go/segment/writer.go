package segment

import (
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/cockroachdb/errors"
	"sync"
)

type Writer interface {
	Write(segments []api.Segment) bool
	Error() error
	Close() error
}

type WriterStream = freighter.ClientStream[api.WriterRequest, api.WriterResponse]

type writer struct {
	stream WriterStream
	mu     struct {
		sync.Mutex
		err         error
		receivedRes chan struct{}
	}
}

func NewWriter(stream WriterStream, keys ...string) (Writer, error) {
	w := &writer{stream: stream}
	if err := w.stream.Send(api.WriterRequest{OpenKeys: keys}); err != nil {
		return nil, err
	}
	w.mu.receivedRes = make(chan struct{}, 1)
	go w.receiveErrors()
	return w, nil
}

func (w *writer) Write(segments []api.Segment) bool {
	select {
	case <-w.mu.receivedRes:
		return false
	default:
		err := w.stream.Send(api.WriterRequest{Segments: segments})
		w.maybeSetError(err)
		return err == nil
	}
}

func (w *writer) Error() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mu.err != nil {
		w.mu.receivedRes = make(chan struct{}, 1)
		err := w.mu.err
		w.mu.err = nil
		return err
	}
	return nil
}

func (w *writer) Close() error {
	if err := w.stream.CloseSend(); err != nil {
		return err
	}
	<-w.mu.receivedRes
	err := w.Error()
	if errors.Is(err, freighter.EOF) {
		return nil
	}
	return err
}

func (w *writer) receiveErrors() {
	for {
		msg, err := w.stream.Receive()
		if err != nil {
			w.maybeSetError(err)
			return
		}
		w.maybeSetError(ferrors.Decode(msg.Err))
	}
}

func (w *writer) maybeSetError(err error) {
	if err == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mu.err != nil {
		return
	}
	w.mu.err = err
	close(w.mu.receivedRes)
}
