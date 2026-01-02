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
	"fmt"
	"io"
	"reflect"

	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	xbinary "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
)

type Codec struct {
	*codec.Codec
	LowerPerfCodec xbinary.Codec
}

func NewWSFramerCodec(channelSvc *channel.Service) httputil.Codec {
	return &Codec{
		LowerPerfCodec: httputil.JSONCodec,
		Codec:          codec.NewDynamic(channelSvc),
	}
}

var _ xbinary.Codec = (*Codec)(nil)

func (c *Codec) Decode(
	ctx context.Context,
	data []byte,
	value any,
) error {
	r := bytes.NewReader(data)
	return c.DecodeStream(ctx, r, value)
}

var (
	highPerfSpecialChar byte = 255
	lowPerfSpecialChar  byte = 254
)

func (c *Codec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value any,
) error {
	switch v := value.(type) {
	case *fhttp.WSMessage[framer.WriterRequest]:
		return c.decodeWriteRequest(ctx, r, v)
	case *fhttp.WSMessage[framer.WriterResponse]:
		return c.decodeWriteResponse(ctx, r, v)
	case *fhttp.WSMessage[framer.StreamerRequest]:
		return c.decodeStreamRequest(ctx, r, v)
	case *fhttp.WSMessage[framer.StreamerResponse]:
		return c.decodeStreamResponse(ctx, r, v)
	default:
		panic(fmt.Sprintf("incompatible type %s provided to framer codec", reflect.TypeOf(value)))
	}
}

func (c *Codec) Encode(ctx context.Context, value any) ([]byte, error) {
	wr := &bytes.Buffer{}
	err := c.EncodeStream(ctx, wr, value)
	return wr.Bytes(), err
}

func (c *Codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	switch v := value.(type) {
	case fhttp.WSMessage[framer.WriterRequest]:
		return c.encodeWriteRequest(ctx, w, v)
	case fhttp.WSMessage[framer.WriterResponse]:
		return c.lowPerfEncode(ctx, true, w, v)
	case fhttp.WSMessage[framer.StreamerRequest]:
		return c.lowPerfEncode(ctx, false, w, v)
	case fhttp.WSMessage[framer.StreamerResponse]:
		return c.encodeStreamResponse(ctx, w, v)
	default:
		panic("incompatible type")
	}
}

func (c *Codec) lowPerfEncode(
	ctx context.Context,
	addSpecialChar bool,
	w io.Writer,
	value any,
) error {
	if addSpecialChar {
		if _, err := w.Write([]byte{lowPerfSpecialChar}); err != nil {
			return err
		}
	}
	b, err := c.LowerPerfCodec.Encode(ctx, value)
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
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[framer.WriterResponse],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if !isLowPerf {
		return errors.Newf("[api.Codec] unexpected high performance codec special character")
	}
	return c.lowPerfDecode(ctx, r, v)
}

func (c *Codec) lowPerfDecode(ctx context.Context, r io.Reader, value any) error {
	return c.LowerPerfCodec.DecodeStream(ctx, r, value)
}

func (c *Codec) decodeWriteRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[framer.WriterRequest],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if isLowPerf {
		if err := c.lowPerfDecode(ctx, r, v); err != nil {
			return err
		}
		if v.Type != fhttp.WSMessageTypeData {
			return nil
		}
		if v.Payload.Command == writer.Open {
			return c.Update(ctx, v.Payload.Config.Keys)
		}
		return nil
	}
	v.Type = fhttp.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Command = writer.Write
	v.Payload.Frame = fr
	return nil
}

func (c *Codec) encodeWriteRequest(
	ctx context.Context,
	w io.Writer,
	v fhttp.WSMessage[framer.WriterRequest],
) error {
	if v.Type != fhttp.WSMessageTypeData || v.Payload.Command != writer.Write {
		return c.lowPerfEncode(ctx, true, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(ctx, w, v.Payload.Frame)
}

func (c *Codec) decodeStreamResponse(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[framer.StreamerResponse],
) error {
	isLowPerf, err := c.decodeIsLowPerf(r)
	if err != nil {
		return err
	}
	if isLowPerf {
		return c.lowPerfDecode(ctx, r, v)
	}
	v.Type = fhttp.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Frame = fr
	return nil
}

func (c *Codec) encodeStreamResponse(
	ctx context.Context,
	w io.Writer,
	v fhttp.WSMessage[framer.StreamerResponse],
) error {
	if v.Type != fhttp.WSMessageTypeData || v.Payload.Frame.Empty() {
		return c.lowPerfEncode(ctx, true, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(ctx, w, v.Payload.Frame)
}

func (c *Codec) decodeStreamRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[framer.StreamerRequest],
) error {
	if err := c.lowPerfDecode(ctx, r, v); err != nil {
		return err
	}
	if v.Type != fhttp.WSMessageTypeData {
		return nil
	}
	return c.Update(ctx, v.Payload.Keys)
}

func (c *Codec) ContentType() string {
	return framerContentType
}

const framerContentType = "application/sy-framer"

func NewCodecResolver(channelSvc *channel.Service) httputil.CodecResolver {
	return func(ct string) (httputil.Codec, error) {
		if ct == framerContentType {
			return NewWSFramerCodec(channelSvc), nil
		}
		return httputil.ResolveCodec(ct)
	}
}
