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
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
)

// Failure is an error that answers a command with a status message and description.
type Failure struct {
	// Message says what happened in one short line.
	Message string
	// Description says why, and what to do now, in full sentences.
	Description string
}

// Error implements error.
func (f Failure) Error() string { return f.Message + ": " + f.Description }

// Handshake runs a TLS handshake with cfg on conn, a connection to addr. It returns a
// Failure that says why the handshake failed. A cfg with no server name takes the host
// of addr. Handshake does not close conn.
func Handshake(ctx context.Context, conn net.Conn, addr string, cfg *tls.Config) error {
	if cfg = cfg.Clone(); cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if host, _, err := net.SplitHostPort(addr); err == nil && cfg.ServerName == "" {
		cfg.ServerName = host
	}
	err := tls.Client(conn, cfg).HandshakeContext(ctx)
	if err == nil {
		return nil
	}
	if f, ok := certificateFailure(err); ok {
		return f
	}
	var header tls.RecordHeaderError
	if errors.As(err, &header) {
		return Failure{
			Message: "Server does not use TLS",
			Description: "The server answered without TLS. Turn off TLS, or connect " +
				"to the TLS port of the server.",
		}
	}
	return Failure{
		Message: "TLS handshake failed",
		Description: fmt.Sprintf(
			"%s. Check that this port of the server uses TLS.", err,
		),
	}
}

// certificateFailure returns the Failure for a handshake error that rejects the
// certificate of the server, and reports whether err is such an error.
func certificateFailure(err error) (Failure, bool) {
	var (
		authority    x509.UnknownAuthorityError
		hostname     x509.HostnameError
		invalid      x509.CertificateInvalidError
		verification *tls.CertificateVerificationError
	)
	switch {
	case errors.As(err, &authority):
		return Failure{
			Message: "Certificate not trusted",
			Description: fmt.Sprintf(
				"%s. Set the CA certificate that signed the certificate of the server.",
				err,
			),
		}, true
	case errors.As(err, &hostname):
		return Failure{
			Message: "Certificate does not match the address",
			Description: fmt.Sprintf(
				"%s. Connect with a host name that the certificate names.", err,
			),
		}, true
	case errors.As(err, &invalid):
		return Failure{
			Message: "Certificate not valid",
			Description: fmt.Sprintf(
				"%s. Check the dates and the usage of the certificate of the server.",
				err,
			),
		}, true
	// macOS returns most rejections as untyped errors, an unknown CA among them.
	case errors.As(err, &verification):
		return Failure{
			Message: "Certificate not trusted",
			Description: fmt.Sprintf(
				"%s. Check the certificate of the server, or set the CA certificate "+
					"that signed it.",
				err,
			),
		}, true
	default:
		return Failure{}, false
	}
}

// FormatOf returns the format of data: json for one JSON value, text for other UTF-8
// text, and binary for anything else.
func FormatOf(data []byte) payload.Format {
	switch {
	case jsontext.Value(data).IsValid():
		return payload.FormatJSON
	case utf8.Valid(data):
		return payload.FormatText
	default:
		return payload.FormatBinary
	}
}

// Display returns the start of data as an answer shows it, from at most limit bytes.
// Data whose format is JSON or text shows as text, cut at a rune boundary. Data of any
// other format shows as hex digits.
func Display(data []byte, format payload.Format, limit int) string {
	if format != payload.FormatJSON && format != payload.FormatText {
		return hex.EncodeToString(data[:min(len(data), limit)])
	}
	return CutText(data, limit)
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
