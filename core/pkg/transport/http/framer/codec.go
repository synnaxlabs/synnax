// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

type (
	Frame          = framer.Frame
	WriterConfig   = framer.WriterConfig
	WriterRequest  = framer.WriterRequest
	WriterResponse = framer.WriterResponse

	IteratorCommand  = framer.IteratorCommand
	IteratorRequest  = framer.IteratorRequest
	IteratorResponse = framer.IteratorResponse
	StreamerRequest  = framer.StreamerRequest
	StreamerResponse = framer.StreamerResponse
	DeleteRequest    = framer.DeleteRequest
)

const (
	WriterCommandOpen           = framer.WriterCommandOpen
	WriterCommandWrite          = framer.WriterCommandWrite
	WriterCommandCommit         = framer.WriterCommandCommit
	WriterCommandSetAuthority   = framer.WriterCommandSetAuthority
	IteratorCommandNext         = framer.IteratorCommandNext
	IteratorCommandPrev         = framer.IteratorCommandPrev
	IteratorCommandSeekFirst    = framer.IteratorCommandSeekFirst
	IteratorCommandSeekLast     = framer.IteratorCommandSeekLast
	IteratorCommandSeekLE       = framer.IteratorCommandSeekLE
	IteratorCommandSeekGE       = framer.IteratorCommandSeekGE
	IteratorCommandValid        = framer.IteratorCommandValid
	IteratorCommandError        = framer.IteratorCommandError
	IteratorCommandSetBounds    = framer.IteratorCommandSetBounds
	IteratorResponseVariantAck  = framer.IteratorResponseVariantAck
	IteratorResponseVariantData = framer.IteratorResponseVariantData
)

type Codec struct {
	*codec.Codec
	LowerPerfCodec encoding.Codec
	// Ctx scopes channel resolution to the lifetime of the connection the codec serves.
	// Resolution runs while a message decodes, and the encoding.Codec interface carries
	// no context.
	Ctx context.Context
}

var _ encoding.Codec = (*Codec)(nil)

func (c *Codec) Decode(data []byte, value any) error {
	return c.DecodeStream(bytes.NewReader(data), value)
}

var (
	highPerfSpecialChar byte = 255
	lowPerfSpecialChar  byte = 254
)

func (c *Codec) DecodeStream(r io.Reader, value any) error {
	switch v := value.(type) {
	case *http.WSMessage[WriterRequest]:
		return c.decodeWriteRequest(r, v)
	case *http.WSMessage[WriterResponse]:
		return c.decodeWriteResponse(r, v)
	case *http.WSMessage[StreamerRequest]:
		return c.decodeStreamRequest(r, v)
	case *http.WSMessage[StreamerResponse]:
		return c.decodeStreamResponse(r, v)
	case *http.WSMessage[IteratorRequest]:
		return c.decodeIteratorRequest(r, v)
	case *http.WSMessage[IteratorResponse]:
		return c.decodeIteratorResponse(r, v)
	default:
		err := errors.Newf(
			"[api.Codec] incompatible type %T provided to framer codec",
			value,
		)
		zap.S().DPanic(err.Error())
		return err
	}
}

func (c *Codec) Encode(value any) ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := c.EncodeStream(wr, value); err != nil {
		return nil, err
	}
	return wr.Bytes(), nil
}

func (c *Codec) EncodeStream(w io.Writer, value any) error {
	switch v := value.(type) {
	case http.WSMessage[WriterRequest]:
		return c.encodeWriteRequest(w, v)
	case http.WSMessage[WriterResponse]:
		return c.lowPerfEncode(true, w, v)
	case http.WSMessage[StreamerRequest]:
		return c.lowPerfEncode(false, w, v)
	case http.WSMessage[StreamerResponse]:
		return c.encodeStreamResponse(w, v)
	case http.WSMessage[IteratorRequest]:
		return c.lowPerfEncode(false, w, v)
	case http.WSMessage[IteratorResponse]:
		return c.encodeIteratorResponse(w, v)
	default:
		err := errors.Newf(
			"[api.Codec] incompatible type %T provided to framer codec",
			value,
		)
		zap.S().DPanic(err.Error())
		return err
	}
}

func (c *Codec) lowPerfEncode(addSpecialChar bool, w io.Writer, value any) error {
	if addSpecialChar {
		if _, err := w.Write([]byte{lowPerfSpecialChar}); err != nil {
			return err
		}
	}
	b, err := c.LowerPerfCodec.Encode(value)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func (c *Codec) decodeIsLowPerf(r io.Reader) (bool, error) {
	var sc uint8
	if err := binary.Read(r, binary.LittleEndian, &sc); err != nil {
		return false, err
	}
	return sc == lowPerfSpecialChar, nil
}

func (c *Codec) decodeWriteResponse(
	r io.Reader,
	v *http.WSMessage[WriterResponse],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if !isLowPerf {
		return errors.Newf(
			"[api.Codec] unexpected high performance codec special character",
		)
	}
	return c.lowPerfDecode(r, v)
}

func (c *Codec) lowPerfDecode(r io.Reader, value any) error {
	return c.LowerPerfCodec.DecodeStream(r, value)
}

func (c *Codec) decodeWriteRequest(
	r io.Reader,
	v *http.WSMessage[WriterRequest],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if isLowPerf {
		if err := c.lowPerfDecode(r, v); err != nil {
			return err
		}
		if v.Type != http.WSMessageTypeData {
			return nil
		}
		if v.Payload.Command == framer.WriterCommandOpen {
			return c.Update(c.Ctx, v.Payload.Config.Keys)
		}
		return nil
	}
	v.Type = http.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Command = framer.WriterCommandWrite
	v.Payload.Frame = fr
	return nil
}

func (c *Codec) encodeWriteRequest(
	w io.Writer,
	v http.WSMessage[WriterRequest],
) error {
	if v.Type != http.WSMessageTypeData ||
		v.Payload.Command != framer.WriterCommandWrite {
		return c.lowPerfEncode(true, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(c.Ctx, w, v.Payload.Frame)
}

func (c *Codec) decodeStreamResponse(
	r io.Reader,
	v *http.WSMessage[StreamerResponse],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if isLowPerf {
		return c.lowPerfDecode(r, v)
	}
	v.Type = http.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Frame = fr
	return nil
}

func (c *Codec) encodeStreamResponse(
	w io.Writer,
	v http.WSMessage[StreamerResponse],
) error {
	if v.Type != http.WSMessageTypeData || v.Payload.Frame.Empty() {
		return c.lowPerfEncode(true, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(c.Ctx, w, v.Payload.Frame)
}

func (c *Codec) decodeStreamRequest(
	r io.Reader,
	v *http.WSMessage[StreamerRequest],
) error {
	if err := c.lowPerfDecode(r, v); err != nil {
		return err
	}
	if v.Type != http.WSMessageTypeData {
		return nil
	}
	if len(v.Payload.Keys) == 0 {
		return nil
	}
	return c.Update(c.Ctx, v.Payload.Keys)
}

func (c *Codec) decodeIteratorRequest(
	r io.Reader,
	v *http.WSMessage[IteratorRequest],
) error {
	if err := c.lowPerfDecode(r, v); err != nil {
		return err
	}
	if v.Type != http.WSMessageTypeData {
		return nil
	}
	if len(v.Payload.Keys) == 0 {
		return nil
	}
	return c.Update(c.Ctx, v.Payload.Keys)
}

func (c *Codec) decodeIteratorResponse(
	r io.Reader,
	v *http.WSMessage[IteratorResponse],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if isLowPerf {
		return c.lowPerfDecode(r, v)
	}
	v.Type = http.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Frame = fr
	v.Payload.Variant = framer.IteratorResponseVariantData
	return nil
}

func (c *Codec) encodeIteratorResponse(
	w io.Writer,
	v http.WSMessage[IteratorResponse],
) error {
	if v.Type != http.WSMessageTypeData ||
		v.Payload.Variant != framer.IteratorResponseVariantData ||
		v.Payload.Frame.Empty() {
		return c.lowPerfEncode(true, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(c.Ctx, w, v.Payload.Frame)
}

// WithCodec returns a StreamServerOption that registers the WS framer codec on a
// streaming server. A fresh codec instance is constructed per connection because the
// framer codec is stateful (it tracks the channel keys for the active stream).
func WithCodec(channelResolver codec.ChannelResolver) http.StreamServerOption {
	return http.WithAdditionalCodec(
		"application/vnd.synnax.frame",
		func(ctx context.Context) encoding.Codec {
			return &Codec{
				Ctx:            ctx,
				LowerPerfCodec: json.Codec,
				Codec:          codec.NewDynamic(channelResolver),
			}
		})
}
