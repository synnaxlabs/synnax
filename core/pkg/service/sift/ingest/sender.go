// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest

import (
	"context"
	"io"
	"time"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// confirmTimeout bounds how long the sender waits for Sift to confirm a stream.
const confirmTimeout = 30 * time.Second

// sender sends requests on an ingest stream that it opens on the first send after
// each confirm. A sender is not safe for concurrent use.
type sender struct {
	client ingestv1.IngestServiceClient
	stream ingestv1.IngestService_IngestWithConfigDataStreamClient
	cancel context.CancelFunc
}

func (s *sender) send(
	ctx context.Context,
	req *ingestv1.IngestWithConfigDataStreamRequest,
) error {
	if s.stream == nil {
		// The stream outlives ctx so that a stopping task can still confirm what it
		// sent.
		streamCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		stream, err := s.client.IngestWithConfigDataStream(streamCtx)
		if err != nil {
			cancel()
			return err
		}
		s.stream, s.cancel = stream, cancel
	}
	if err := s.stream.Send(req); err != nil {
		if err == io.EOF {
			// Sift ended the stream; its status is the real error.
			_, err = s.stream.CloseAndRecv()
		}
		s.abort()
		return err
	}
	return nil
}

// confirm half-closes the stream and waits for Sift to confirm every request sent on
// it.
func (s *sender) confirm() error {
	if s.stream == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.stream.CloseAndRecv()
		done <- err
	}()
	var err error
	select {
	case err = <-done:
	case <-time.After(confirmTimeout):
		err = status.Error(codes.DeadlineExceeded, "Sift did not confirm the data sent")
	}
	s.abort()
	return err
}

// abort drops the stream without waiting for Sift.
func (s *sender) abort() {
	if s.cancel != nil {
		s.cancel()
	}
	s.stream, s.cancel = nil, nil
}

// permanentCodes are the gRPC codes that retrying cannot fix.
var permanentCodes = map[codes.Code]string{
	codes.Unauthenticated:    "Sift rejected the API key",
	codes.PermissionDenied:   "the API key has no permission for this",
	codes.InvalidArgument:    "Sift rejected the request",
	codes.NotFound:           "Sift could not find the ingestion config",
	codes.FailedPrecondition: "Sift rejected the request",
	codes.Unimplemented:      "the Sift server does not support this API",
}

// classify reports whether retrying cannot fix err, and wraps such an error with a
// message for the user.
func classify(err error) (bool, error) {
	st, ok := status.FromError(err)
	if !ok {
		return false, err
	}
	msg, permanent := permanentCodes[st.Code()]
	if !permanent {
		return false, err
	}
	return true, errors.Wrap(err, msg)
}
