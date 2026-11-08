// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// Pipeline runs one request at a time on a flowing StreamIterator. It is not safe for
// concurrent use.
type Pipeline struct {
	// Requests is the input stream of the StreamIterator.
	Requests confluence.Inlet[Request]
	// Responses is the output stream of the StreamIterator, which must close it on
	// exit.
	Responses confluence.Outlet[Response]
	// Shutdown cancels the context that the StreamIterator flows under.
	Shutdown context.CancelFunc
	// WaitGroup tracks the routines of the StreamIterator.
	WaitGroup signal.WaitGroup
	stopErr   error
	stopped   bool
	closed    bool
}

// Exec sends req, appends its data responses to value, and returns its ack. If the
// StreamIterator exits first, Exec and every later call return false and the error that
// stopped it. An exit without an error returns an error that wraps ErrClosed. After
// Close, Exec returns ErrClosed.
func (p *Pipeline) Exec(req Request, value *[]Response) (bool, error) {
	if p.closed {
		return false, ErrClosed
	}
	if p.stopped {
		return false, p.stopErr
	}
	requests := p.Requests.Inlet()
	for {
		select {
		case requests <- req:
			requests = nil
		case res, ok := <-p.Responses.Outlet():
			if !ok {
				if p.stop() == nil {
					p.stopErr = errors.Wrap(
						ErrClosed,
						"iterator exited without a response",
					)
				}
				return false, p.stopErr
			}
			if res.Variant == ResponseVariantAck {
				return res.Ack, res.Error
			}
			*value = append(*value, res)
		}
	}
}

// Close stops the StreamIterator and returns the error that stopped it. Closing a
// closed Pipeline returns nil.
func (p *Pipeline) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	defer p.Shutdown()
	return p.stop()
}

func (p *Pipeline) stop() error {
	if !p.stopped {
		p.stopped = true
		p.Requests.Close()
		p.stopErr = p.WaitGroup.Wait()
	}
	return p.stopErr
}
