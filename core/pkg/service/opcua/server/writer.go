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

// prepare validates s, normalizes the node ID of each binding, and refuses a read on
// the same node, field, and index as another read, or a write to the same node as
// another write.
func prepare(s *Server) error {
	if err := s.Validate(); err != nil {
		return err
	}
	type readKey struct {
		node, field string
		index       channel.Key
	}
	v := validate.New("Server")
	reads := make(set.Set[readKey], len(s.Reads))
	for i := range s.Reads {
		r := &s.Reads[i]
		path := fmt.Sprintf("reads.%d", i)
		node, err := normalizeNode(r.Node)
		if err != nil {
			v.Exec(func() error { return validate.PathedError(err, path+".node") })
			continue
		}
		r.Node = node
		k := readKey{node: node, field: r.Field, index: r.Index}
		v.Ternaryf(path, reads.Contains(k), "duplicates the read of %s", node)
		reads.Add(k)
	}
	writes := make(set.Set[string], len(s.Writes))
	for i := range s.Writes {
		w := &s.Writes[i]
		path := fmt.Sprintf("writes.%d", i)
		node, err := normalizeNode(w.Node)
		if err != nil {
			v.Exec(func() error { return validate.PathedError(err, path+".node") })
			continue
		}
		w.Node = node
		v.Ternaryf(path, writes.Contains(node), "duplicates the write to %s", node)
		writes.Add(node)
	}
	return v.Error()
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
