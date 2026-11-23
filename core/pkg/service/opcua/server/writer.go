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
	"context"
	"fmt"
	"net"
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes OPC UA servers in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Server]
	certs    *certificate.Service
}

// prepare validates s and normalizes the node ID of each binding. Beyond the schema
// rules, it refuses an IPv6 endpoint host that does not parse, a malformed field path,
// a read whose index is its own channel, two reads on one node, field, and index, two
// reads into one channel, and two writes to one node or from one channel.
func prepare(s *Server) error {
	if err := s.Validate(); err != nil {
		return err
	}
	v := validate.New("Server")
	v.Ternary(
		"connection.endpoint",
		!validIPv6Host(s.Connection.Endpoint),
		"has an IPv6 host that is not an IPv6 address",
	)
	prepareReads(v, s.Reads)
	prepareWrites(v, s.Writes)
	return v.Error()
}

// validIPv6Host reports false when endpoint has a bracketed host that is not an IPv6
// address.
func validIPv6Host(endpoint string) bool {
	rest, ok := strings.CutPrefix(endpoint, "opc.tcp://[")
	if !ok {
		return true
	}
	host, _, ok := strings.Cut(rest, "]")
	ip := net.ParseIP(host)
	return ok && ip != nil && ip.To4() == nil
}

func prepareReads(v *validate.Validator, reads []Read) {
	type readKey struct {
		node, field string
		index       channel.Key
	}
	keys := make(set.Set[readKey], len(reads))
	channels := make(set.Set[channel.Key], len(reads))
	for i := range reads {
		r := &reads[i]
		path := fmt.Sprintf("reads.%d", i)
		v.Ternary(path+".field", !validFieldPath(r.Field), "is not a valid field path")
		v.Ternary(path+".index", r.Index == r.Channel, "must differ from the channel")
		v.Ternary(
			path+".channel",
			channels.Contains(r.Channel),
			"duplicates the channel of another read",
		)
		channels.Add(r.Channel)
		node, err := normalizeNode(r.Node)
		if err != nil {
			v.Exec(func() error { return validate.PathedError(err, path+".node") })
			continue
		}
		r.Node = node
		k := readKey{node: node, field: r.Field, index: r.Index}
		v.Ternaryf(path, keys.Contains(k), "duplicates the read of %s", node)
		keys.Add(k)
	}
}

func prepareWrites(v *validate.Validator, writes []Write) {
	nodes := make(set.Set[string], len(writes))
	channels := make(set.Set[channel.Key], len(writes))
	for i := range writes {
		w := &writes[i]
		path := fmt.Sprintf("writes.%d", i)
		v.Ternary(
			path+".channel",
			channels.Contains(w.Channel),
			"duplicates the channel of another write",
		)
		channels.Add(w.Channel)
		node, err := normalizeNode(w.Node)
		if err != nil {
			v.Exec(func() error { return validate.PathedError(err, path+".node") })
			continue
		}
		w.Node = node
		v.Ternaryf(path, nodes.Contains(node), "duplicates the write to %s", node)
		nodes.Add(node)
	}
}

// validFieldPath reports whether field is empty or is names joined by '.', where each
// name is not empty and '&' escapes the next character.
func validFieldPath(field string) bool {
	if field == "" {
		return true
	}
	empty := true
	for i := 0; i < len(field); i++ {
		switch field[i] {
		case '&':
			if i == len(field)-1 {
				return false
			}
			i++
			empty = false
		case '.':
			if empty {
				return false
			}
			empty = true
		default:
			empty = false
		}
	}
	return !empty
}

// checkPrivateKeys refuses a login certificate or a client certificate that has no
// private key, because the Driver signs with both.
func (w Writer) checkPrivateKeys(ctx context.Context, s Server) error {
	keys := make(map[string]certificate.Key, 2)
	if auth, ok := s.Connection.Auth.Variant.(CertificateAuth); ok {
		keys["connection.auth.certificate"] = auth.Certificate
	}
	if c, ok := securedChannel(s.Connection); ok && c.ClientCertificate != nil {
		keys["connection.security.client_certificate"] = *c.ClientCertificate
	}
	v := validate.New("Server")
	for path, key := range keys {
		var c certificate.Certificate
		if err := w.certs.NewRetrieve().
			Where(certificate.MatchKeys(key)).
			Entry(&c).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		v.Ternaryf(
			path,
			!hasPrivateKey(c),
			"certificate %q has no private key",
			c.Base().Name,
		)
	}
	return v.Error()
}

func hasPrivateKey(c certificate.Certificate) bool {
	switch v := c.Variant.(type) {
	case certificate.StoredCertificate:
		return v.PrivateKey != nil
	case certificate.FileCertificate:
		return v.PrivateKeyFile != nil
	default:
		return false
	}
}

func (w Writer) setStatus(ctx context.Context, s Server) error {
	var st Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](s.ResourceID())).
		Entry(&st).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		st = Status{
			Owner:   new(s.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Server: s.Key},
		}
	} else if err != nil {
		return err
	}
	st.Name = s.Name
	st.Details.Rack = s.Rack
	return w.status.Set(ctx, &st)
}

// Create stores s and writes an unknown status when it has none. An existing status
// takes the name and rack of s. A server with a nil key gets a new one. It returns a
// validation error when s is invalid, or when its rack, password, or certificates do
// not exist.
func (w Writer) Create(ctx context.Context, s *Server) error {
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if err := prepare(s); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.checkPrivateKeys(ctx, *s); err != nil {
		return err
	}
	return w.setStatus(ctx, *s)
}

// update applies change to the server stored under key. A rename or rack move updates
// the status, or writes it as unknown when it is missing.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Server) (Server, error),
) error {
	var prev, next Server
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Server](key)).
		ChangeErr(func(_ gorp.Context, s Server) (Server, error) {
			prev = s
			updated, err := change(s)
			if err != nil {
				return Server{}, err
			}
			updated.Key = key
			if err = prepare(&updated); err != nil {
				return Server{}, err
			}
			next = updated
			return updated, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.checkPrivateKeys(ctx, next); err != nil {
		return err
	}
	if next.Name != prev.Name || next.Rack != prev.Rack {
		return w.setStatus(ctx, next)
	}
	return nil
}

// Delete removes the servers stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Server](keys...)).
		Exec(ctx, w.tx)
}
