// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json/jsontext"
	"fmt"
	"net"
	"slices"
	"unicode/utf8"

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// StageTLS is the stage of a connection test that runs the TLS handshake.
const StageTLS = "tls"

// failure is an error that answers a command with a status message and description.
type failure struct{ message, description string }

// Error implements error.
func (f failure) Error() string { return f.message + ": " + f.description }

// AnswerTest returns the data that answers a connection test that ran steps. When a
// step failed, the error gives the answer the message and the description of the first
// step that failed.
func AnswerTest(steps []task.TestStep) (msgpack.EncodedJSON, error) {
	data, err := msgpack.NewEncodedJSON(task.TestResult{Steps: steps})
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(steps, func(s task.TestStep) bool {
		return s.Variant == status.VariantError
	})
	if i < 0 {
		return data, nil
	}
	return data, failure{message: steps[i].Message, description: steps[i].Description}
}

// Handshake runs a TLS handshake with cfg on conn, a connection to addr, and returns
// the outcome as the StageTLS stage of a connection test. A cfg with no server name
// takes the host of addr. Handshake does not close conn.
func Handshake(
	ctx context.Context,
	conn net.Conn,
	addr string,
	cfg *tls.Config,
) task.TestStep {
	if cfg = cfg.Clone(); cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if host, _, err := net.SplitHostPort(addr); err == nil && cfg.ServerName == "" {
		cfg.ServerName = host
	}
	step := task.TestStep{Stage: StageTLS, Address: addr, Variant: status.VariantError}
	err := tls.Client(conn, cfg).HandshakeContext(ctx)
	var (
		authority x509.UnknownAuthorityError
		hostname  x509.HostnameError
		invalid   x509.CertificateInvalidError
		header    tls.RecordHeaderError
	)
	switch {
	case err == nil:
		step.Variant, step.Message = status.VariantSuccess, "Secured connection"
	case errors.As(err, &authority):
		step.Message = "Certificate not trusted"
		step.Description = fmt.Sprintf(
			"%s. Set the CA certificate that signed the certificate of the server.",
			err,
		)
	case errors.As(err, &hostname):
		step.Message = "Certificate does not match the address"
		step.Description = fmt.Sprintf(
			"%s. Connect with a host name that the certificate names.", err,
		)
	case errors.As(err, &invalid):
		step.Message = "Certificate not valid"
		step.Description = fmt.Sprintf(
			"%s. Check the dates and the usage of the certificate of the server.", err,
		)
	case errors.As(err, &header):
		step.Message = "Server does not use TLS"
		step.Description = "The server answered without TLS. Turn off TLS, or connect " +
			"to the TLS port of the server."
	default:
		step.Message = "TLS handshake failed"
		step.Description = fmt.Sprintf(
			"%s. Check that this port of the server uses TLS.", err,
		)
	}
	return step
}

// Format is the encoding of a message payload.
type Format string

const (
	// FormatJSON is one JSON value.
	FormatJSON Format = "json"
	// FormatSchemaRegistry is a Confluent Schema Registry payload: a zero byte, a
	// 4-byte schema ID, and the encoded value.
	FormatSchemaRegistry Format = "schema_registry"
	// FormatText is UTF-8 text that is not JSON.
	FormatText Format = "text"
	// FormatBinary is any other payload.
	FormatBinary Format = "binary"
)

// FormatOf returns the format of payload.
func FormatOf(payload []byte) Format {
	switch {
	case jsontext.Value(payload).IsValid():
		return FormatJSON
	case len(payload) >= 5 && payload[0] == 0:
		return FormatSchemaRegistry
	case utf8.Valid(payload):
		return FormatText
	default:
		return FormatBinary
	}
}

// Display returns the format of payload and the start of payload as an answer shows
// it, from at most limit bytes. A JSON or text payload shows as text, cut at a rune
// boundary. Any other payload shows as hex digits.
func Display(payload []byte, limit int) (Format, string) {
	format := FormatOf(payload)
	if format != FormatJSON && format != FormatText {
		return format, hex.EncodeToString(payload[:min(len(payload), limit)])
	}
	return format, CutText(payload, limit)
}

// CutText returns at most limit bytes from the start of text, cut at a rune boundary.
func CutText(text []byte, limit int) string {
	if len(text) <= limit {
		return string(text)
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return string(text[:limit])
}
