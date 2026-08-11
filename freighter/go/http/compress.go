// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"strings"

	"github.com/samber/lo"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// defaultCompressions are the Content-Encodings both ends offer, ordered by preference.
// Zstd leads because it reaches within a few percent of Brotli's ratio at several times
// the speed; Brotli covers peers that predate zstd support; Gzip is the universal
// fallback. Deflate sits last so it is never chosen for an outgoing body, and is
// present only so an incoming one is still understood.
var defaultCompressions = []xhttp.Compression{
	xhttp.Zstd,
	xhttp.Brotli,
	xhttp.Gzip,
	xhttp.Deflate,
}

const (
	// defaultMinCompressSize is the smallest body worth compressing. Below roughly a
	// kilobyte the CPU spent compressing exceeds the transmission time saved on a fast
	// link, and below about 128 bytes gzip and zstd framing makes the body larger.
	defaultMinCompressSize = int(telem.Kilobyte)
	// defaultMaxDecompressedSize bounds how far an incoming body may expand, so a small
	// compressed request cannot force an unbounded allocation.
	defaultMaxDecompressedSize = int(64 * telem.Megabyte)
)

// compressionOptions is the set of compression settings shared by the unary server and
// client. Both ends compress what they send and decompress what they receive, so the
// same knobs apply in both directions.
type compressionOptions struct {
	// compressions are the Compressions this end offers, ordered by preference. Nil
	// disables compression in both directions.
	compressions []xhttp.Compression
	// minCompressSize is the smallest outgoing body that gets compressed. Bodies below
	// it are sent as-is.
	minCompressSize int
	// maxDecompressedSize bounds the size of an incoming body once decompressed.
	maxDecompressedSize int
}

func defaultCompressionOptions() compressionOptions {
	return compressionOptions{
		compressions:        defaultCompressions,
		minCompressSize:     defaultMinCompressSize,
		maxDecompressedSize: defaultMaxDecompressedSize,
	}
}

// acceptEncodingHeader renders the offered Compressions as an Accept-Encoding header
// value. It returns an empty string when compression is disabled, which keeps the
// header off the request entirely so the peer sends an uncompressed body.
func (o compressionOptions) acceptEncodingHeader() string {
	if len(o.compressions) == 0 {
		return ""
	}
	return strings.Join(lo.Map(
		o.compressions,
		func(c xhttp.Compression, _ int) string { return c.ContentEncoding() },
	), ", ")
}

// contentEncodings lists the offered Content-Encoding tokens for reporting.
func (o compressionOptions) contentEncodings() []string {
	return lo.Map(
		o.compressions,
		func(c xhttp.Compression, _ int) string { return c.ContentEncoding() },
	)
}

// compressNegotiated compresses body for a peer advertising acceptEncoding, choosing
// the encoding that peer ranks highest. Servers use it for responses, where the sender
// learns what the receiver accepts from the request.
func (o compressionOptions) compressNegotiated(
	body []byte,
	acceptEncoding string,
) ([]byte, string, error) {
	if !o.worthCompressing(body) {
		return body, "", nil
	}
	compression, ok := xhttp.NegotiateCompression(acceptEncoding, o.compressions)
	if !ok {
		return body, "", nil
	}
	return compressWith(body, compression)
}

// compressPreferred compresses body with the first offered Compression. Clients use it
// for requests, where nothing tells the sender what the receiver accepts, so the
// preferred encoding is sent unilaterally.
func (o compressionOptions) compressPreferred(body []byte) ([]byte, string, error) {
	if !o.worthCompressing(body) {
		return body, "", nil
	}
	return compressWith(body, o.compressions[0])
}

// worthCompressing reports whether body clears both gates on compressing at all: that
// an encoding is offered, and that the body is large enough for the saved bytes to be
// worth the CPU.
func (o compressionOptions) worthCompressing(body []byte) bool {
	return len(o.compressions) > 0 && len(body) >= o.minCompressSize
}

// compressWith returns body compressed alongside the Content-Encoding to declare. The
// returned encoding is empty, and the body untouched, when compressing would not have
// made it smaller.
func compressWith(body []byte, compression xhttp.Compression) ([]byte, string, error) {
	compressed, err := compression.Compress(body)
	if err != nil {
		return nil, "", err
	}
	if len(compressed) >= len(body) {
		return body, "", nil
	}
	return compressed, compression.ContentEncoding(), nil
}

// decompressIncoming returns body decoded according to contentEncoding, passing it
// through untouched when the peer sent it uncompressed. It returns
// xhttp.ErrUnsupportedContentEncoding if contentEncoding names an encoding this end
// does not offer, and xhttp.ErrBodyTooLarge if the body expands past
// maxDecompressedSize.
func (o compressionOptions) decompressIncoming(
	body []byte,
	contentEncoding string,
) ([]byte, error) {
	compression, err := xhttp.ResolveCompression(contentEncoding, o.compressions)
	if err != nil || compression == nil {
		return body, err
	}
	return compression.Decompress(body, o.maxDecompressedSize)
}
