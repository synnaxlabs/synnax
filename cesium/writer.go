package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type WriterConfig struct {
	Start    telem.TimeStamp
	Channels []string
}

type Writer interface {
	Write(frame Frame) bool
	Commit() bool
	Error() error
	Close() error
}

type writer struct {
	requests          confluence.Inlet[WriteRequest]
	responses         confluence.Outlet[WriteResponse]
	wg                signal.WaitGroup
	logger            *zap.Logger
	hasAccumulatedErr bool
}

const unexpectedSteamClosure = "[cesium] - unexpected early closure of response stream"

func wrapStreamWriter(internal StreamWriter) *writer {
	sCtx, _ := signal.Background()
	req := confluence.NewStream[WriteRequest](1)
	res := confluence.NewStream[WriteResponse](1)
	internal.InFrom(req)
	internal.OutTo(res)
	internal.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
	)
	return &writer{requests: req, responses: res, wg: sCtx}
}

// Write implements the Writer interface.
func (w *writer) Write(rec Frame) bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriteRequest{Frame: rec, Command: WriterWrite}:
		return true
	}
}

// Commit implements the Writer interface.
func (w *writer) Commit() bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriteRequest{Command: WriterCommit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterCommit {
			return res.Ack
		}
	}
	w.logger.DPanic(unexpectedSteamClosure)
	return false
}

func (w *writer) Error() error {
	w.requests.Inlet() <- WriteRequest{Command: WriterError}
	for res := range w.responses.Outlet() {
		if res.Command == WriterError {
			w.hasAccumulatedErr = false
			return res.Err
		}
	}
	w.logger.DPanic(unexpectedSteamClosure)
	return errors.New(unexpectedSteamClosure)
}

// Close implements the Writer interface.
func (w *writer) Close() (err error) {
	w.requests.Close()
	for range w.responses.Outlet() {
	}
	return w.wg.Wait()
}
