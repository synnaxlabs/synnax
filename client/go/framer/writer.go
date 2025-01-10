package framer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/errors"
)

type (
	WriterRequest  = api.FrameWriterRequest
	WriterResponse = api.FrameWriterResponse
	WriterStream   = freighter.ClientStream[WriterRequest, WriterResponse]
	WriterClient   = freighter.StreamClient[WriterRequest, WriterResponse]
	WriterConfig   = api.FrameWriterConfig
)

// Writer is used to write channel data to a Synnax cluster. It should not be constructed
// directly, but instead created using the OpenWriter method on the Synnax client.
//
// The Writer is a streaming protocol that is heavily optimized for performance. For a
// detailed guide on writing data to Synnax, see https://docs.synnaxlabs.com/reference/concepts/writes.
// A rough summary of the writer process is as follows:
//
//  1. The writer is opened with a starting timestamp and a list of channel keys (or
//     names). The writer will fail to open if the starting timestamp overlaps with any
//     existing telemetry for any of the channels specified. If the writer is opened
//     successfully, the caller is then free to write frames to the writer.
//
//  2. To write a frame, the caller can use the Write method and follow the validation
//     rules described in the method's documentation. This process is asynchronous,
//     meaning that write will return before the frame has been written to the cluster. This
//     also means that the writer can accumulate an error after write is called. If the
//     writer accumulates an error, all subsequent write and commit calls will return False.
//     The caller can check for errors by calling the error method, which returns the
//     accumulated error and resets the writer for future use. The caller can also check
//     for errors by closing the writer, which will raise any accumulated error.
//
//  3. To commit the written frames to the database, the caller can call the Commit
//     method. Unlike write, commit is synchronous, meaning that it will not return until
//     all frames have been committed to the database. If the writer has accumulated an
//     error, commit will return False. After the caller acknowledges the error, they can
//     attempt to commit again. Commit can be called several times throughout a writer's
//     lifetime, and will commit all frames written since the last commit.
//
//  4. A writer MUST be closed after use in order to prevent resource leaks. Close
//     should typically be called in a 'finally' block. If the writer has accumulated an
//     error, close will raise the accumulated error.
type Writer struct{ stream WriterStream }

func openWriter(ctx context.Context, client WriterClient, cfg WriterConfig) (*Writer, error) {
	s, err := client.Stream(ctx, "")
	if err != nil {
		return nil, err
	}
	if err := s.Send(WriterRequest{Config: cfg}); err != nil {
		return nil, err
	}
	return &Writer{stream: s}, nil
}

// Write writes the given data to the database.
func (w *Writer) Write(ctx context.Context, frame core.Frame) bool {
	if err := w.stream.Send(WriterRequest{Command: writer.Data, Frame: frame}); err != nil {
		return false
	}
	return true
}

// Commit commits written data to the database, making it available for reads.
func (w *Writer) Commit(ctx context.Context) bool {
	if err := w.stream.Send(WriterRequest{Command: writer.Commit}); err != nil {
		return false
	}
	for {
		resp, err := w.stream.Receive()
		if err != nil {
			return false
		}
		if resp.Command == writer.Commit {
			return true
		}
	}
}

func (w *Writer) Close(ctx context.Context) error {
	if err := w.stream.CloseSend(); err != nil {
		return err
	}
	for {
		if _, err := w.stream.Receive(); err != nil {
			return errors.Skip(err, freighter.EOF)
		}
	}

}
