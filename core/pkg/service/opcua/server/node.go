// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// coreNamespace is the URI of namespace 0, whose node IDs carry no namespace.
const coreNamespace = "http://opcfoundation.org/UA/"

// maxStringIdentifier is the longest string identifier, in bytes, that OPC UA allows.
const maxStringIdentifier = 4096

// normalizeNode returns id in the form the Driver writes: nsu=<uri>;<type>=<identifier>
// with ';' and '%' in the URI percent-encoded, or <type>=<identifier> alone in
// namespace 0. It returns a validation error for a malformed ID and for an ID that
// names a namespace by an index other than 0, which a server can renumber.
func normalizeNode(id string) (string, error) {
	namespace, body, err := splitNamespace(id)
	if err == nil {
		body, err = normalizeIdentifier(body)
	}
	if err != nil {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"invalid node ID %q: %s",
			id,
			err,
		)
	}
	if namespace == "" || namespace == coreNamespace {
		return body, nil
	}
	encoded := strings.NewReplacer("%", "%25", ";", "%3B").Replace(namespace)
	return "nsu=" + encoded + ";" + body, nil
}

// splitNamespace returns the decoded namespace URI of id, empty for namespace 0, and
// the identifier that follows it.
func splitNamespace(id string) (namespace, body string, err error) {
	switch lower := strings.ToLower(id); {
	case strings.HasPrefix(lower, "nsu="):
		uri, rest, ok := strings.Cut(id[4:], ";")
		if !ok {
			return "", "", errors.New("no identifier after the namespace URI")
		}
		decoded, err := url.PathUnescape(uri)
		if err != nil || decoded == "" {
			return "", "", errors.New("malformed namespace URI")
		}
		return decoded, rest, nil
	case strings.HasPrefix(lower, "ns="):
		index, rest, ok := strings.Cut(id[3:], ";")
		if !ok {
			return "", "", errors.New("no identifier after the namespace index")
		}
		if n, err := strconv.ParseUint(index, 10, 16); err != nil || n != 0 {
			return "", "", errors.New("name the namespace by its URI, as nsu=<uri>")
		}
		return "", rest, nil
	default:
		return "", id, nil
	}
}

// normalizeIdentifier returns body, a type letter, '=', and an identifier, with a
// lowercase type letter and the identifier in its canonical form.
func normalizeIdentifier(body string) (string, error) {
	if len(body) < 3 || body[1] != '=' {
		return "", errors.New("expected i=, s=, g=, or b= and an identifier")
	}
	identifier := body[2:]
	switch body[0] {
	case 'i', 'I':
		n, err := strconv.ParseUint(identifier, 10, 32)
		if err != nil {
			return "", errors.New(
				"the numeric identifier is not a 32-bit unsigned integer",
			)
		}
		return "i=" + strconv.FormatUint(n, 10), nil
	case 's', 'S':
		if len(identifier) > maxStringIdentifier {
			return "", errors.New("the string identifier is longer than 4096 bytes")
		}
		return "s=" + identifier, nil
	case 'g', 'G':
		g, err := uuid.Parse(identifier)
		if err != nil || len(identifier) != 36 {
			return "", errors.New("the GUID identifier is malformed")
		}
		return "g=" + g.String(), nil
	case 'b', 'B':
		b, err := base64.StdEncoding.DecodeString(identifier)
		if err != nil || base64.StdEncoding.EncodeToString(b) != identifier {
			return "", errors.New("the byte string identifier is not canonical base64")
		}
		return "b=" + identifier, nil
	default:
		return "", errors.New("expected i=, s=, g=, or b= and an identifier")
	}
}
