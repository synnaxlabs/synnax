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
	var (
		authority x509.UnknownAuthorityError
		hostname  x509.HostnameError
		invalid   x509.CertificateInvalidError
		header    tls.RecordHeaderError
	)
	switch {
	case errors.As(err, &authority):
		return Failure{
			Message: "Certificate not trusted",
			Description: fmt.Sprintf(
				"%s. Set the CA certificate that signed the certificate of the server.",
				err,
			),
		}
	case errors.As(err, &hostname):
		return Failure{
			Message: "Certificate does not match the address",
			Description: fmt.Sprintf(
				"%s. Connect with a host name that the certificate names.", err,
			),
		}
	case errors.As(err, &invalid):
		return Failure{
			Message: "Certificate not valid",
			Description: fmt.Sprintf(
				"%s. Check the dates and the usage of the certificate of the server.",
				err,
			),
		}
	case errors.As(err, &header):
		return Failure{
			Message: "Server does not use TLS",
			Description: "The server answered without TLS. Turn off TLS, or connect " +
				"to the TLS port of the server.",
		}
	default:
		return Failure{
			Message: "TLS handshake failed",
			Description: fmt.Sprintf(
				"%s. Check that this port of the server uses TLS.", err,
			),
		}
	}
}

// Format is the encoding of a message payload.
type Format string

const (
	// FormatJSON is one JSON value.
	FormatJSON Format = "json"
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
	case utf8.Valid(payload):
		return FormatText
	default:
		return FormatBinary
	}
}

// Display returns the start of payload as an answer shows it, from at most limit
// bytes. A payload whose format is JSON or text shows as text, cut at a rune boundary.
// A payload of any other format shows as hex digits.
func Display(payload []byte, format Format, limit int) string {
	if format != FormatJSON && format != FormatText {
		return hex.EncodeToString(payload[:min(len(payload), limit)])
	}
	return CutText(payload, limit)
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
