// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package protocol

import "go.lsp.dev/jsonrpc2"

const (
	// LSPReservedErrorRangeStart is the start range of LSP reserved error codes.
	//
	// It doesn't denote a real error code.
	//
	// @since 3.16.0.
	LSPReservedErrorRangeStart jsonrpc2.Code = -32899

	// ContentModified is the state change that invalidates the result of a request in execution.
	//
	// Defined by the protocol.
	CodeContentModified jsonrpc2.Code = -32801

	// RequestCancelled is the cancellation error.
	//
	// Defined by the protocol.
	CodeRequestCancelled jsonrpc2.Code = -32800

	// LSPReservedErrorRangeEnd is the end range of LSP reserved error codes.
	//
	// It doesn't denote a real error code.
	//
	// @since 3.16.0.
	LSPReservedErrorRangeEnd jsonrpc2.Code = -32800
)

var (
	// ErrContentModified should be used when a request is canceled early.
	ErrContentModified = jsonrpc2.NewError(CodeContentModified, "cancelled JSON-RPC")

	// ErrRequestCancelled should be used when a request is canceled early.
	ErrRequestCancelled = jsonrpc2.NewError(CodeRequestCancelled, "cancelled JSON-RPC")
)
