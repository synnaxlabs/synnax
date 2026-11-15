// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package batch runs an ordered list of requests to other endpoints in one Gorp
// transaction.
package batch

import (
	"context"
	"slices"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/json"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// Payload is the body of one request, held undecoded until the batch knows which
// endpoint takes it. It decodes with the codec that decoded the batch.
type Payload struct {
	data   []byte
	decode func(data []byte, v any) error
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Payload) UnmarshalJSON(data []byte) error {
	p.data, p.decode = slices.Clone(data), json.Codec.Decode
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder.
func (p *Payload) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	p.data, p.decode = raw, xmsgpack.Codec.Decode
	return err
}

// Handler runs one request of a batch in tx.
type Handler func(ctx context.Context, tx gorp.Tx, p Payload) (any, error)

// Route returns a Handler that decodes a payload into the request of handle and runs
// it.
func Route[RQ, RS any](handle func(context.Context, gorp.Tx, RQ) (RS, error)) Handler {
	return func(ctx context.Context, tx gorp.Tx, p Payload) (any, error) {
		var req RQ
		if p.decode == nil {
			return nil, validate.PathedError(validate.ErrRequired, "payload")
		}
		if err := p.decode(p.data, &req); err != nil {
			return nil, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"payload",
			)
		}
		return handle(ctx, tx, req)
	}
}

// Routes maps each endpoint a batch accepts to its Handler.
type Routes map[string]Handler

// Service runs batches.
type Service struct{ routes Routes }

// NewService returns a Service that accepts the endpoints in routes. Every route must
// do all of its work inside the transaction it receives.
func NewService(routes Routes) *Service { return &Service{routes: routes} }

type (
	// Request names an endpoint and the payload to send to it.
	Request struct {
		Endpoint string  `json:"endpoint" msgpack:"endpoint"`
		Payload  Payload `json:"payload"  msgpack:"payload"`
	}
	// ExecRequest is an ordered list of requests.
	ExecRequest struct {
		Requests []Request `json:"requests" msgpack:"requests"`
	}
	// ExecResponse holds the response to each request, in order.
	ExecResponse struct {
		Responses []any `json:"responses" msgpack:"responses"`
	}
)

// Exec runs the requests in order in tx and stops at the first error, which names the
// index of the failed request. It returns a validation error before running anything
// when an endpoint is not one the batch accepts.
func (s *Service) Exec(
	ctx context.Context,
	tx gorp.Tx,
	req ExecRequest,
) (ExecResponse, error) {
	handlers := make([]Handler, len(req.Requests))
	for i, r := range req.Requests {
		h, ok := s.routes[r.Endpoint]
		if !ok {
			return ExecResponse{}, validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"%q cannot run in a batch",
				r.Endpoint,
			), "requests", strconv.Itoa(i), "endpoint")
		}
		handlers[i] = h
	}
	res := ExecResponse{Responses: make([]any, len(req.Requests))}
	for i, r := range req.Requests {
		var err error
		if res.Responses[i], err = handlers[i](ctx, tx, r.Payload); err != nil {
			return ExecResponse{}, requestError(err, i)
		}
	}
	return res, nil
}

// requestError names request i in err. A validation error gets the path requests.i.
// Any other error keeps its type, and its message starts with requests.i.
func requestError(err error, i int) error {
	var pathErr validate.PathError
	if errors.Is(err, validate.ErrValidation) || errors.As(err, &pathErr) {
		return validate.PathedError(err, "requests", strconv.Itoa(i))
	}
	return errors.Wrapf(err, "requests.%d", i)
}
