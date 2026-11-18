// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/fs"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// expiryWarning is how long before a certificate expires that its status becomes a
// warning.
const expiryWarning = 30 * telem.Day

const dateLayout = "January 2, 2006"

// expiryStatus returns the status that reports the validity of c at now, and false when
// the expiry of c is unknown.
func expiryStatus(c Certificate, now telem.TimeStamp) (status.Status[any], bool) {
	details, ok := c.Details()
	if !ok {
		return status.Status[any]{}, false
	}
	s := validityStatus(c.Base().Name, details, now)
	s.Owner = new(c.ResourceID())
	return s, true
}

// validityStatus returns the status named name that reports the validity at now of a
// certificate with details.
func validityStatus(
	name string,
	details Details,
	now telem.TimeStamp,
) status.Status[any] {
	date := details.NotAfter.Time().UTC().Format(dateLayout)
	s := status.Status[any]{
		Name:        name,
		Time:        now,
		Variant:     status.VariantSuccess,
		Message:     "Certificate valid",
		Description: "The certificate expires on " + date + ".",
	}
	// A NotAfter past the last time stamp saturates, so its date is not the real one.
	if details.NotAfter == telem.TimeStampMax {
		s.Description = "The certificate does not expire."
	}
	switch {
	case now < details.NotBefore:
		s.Variant = status.VariantWarning
		s.Message = "Certificate not yet valid"
		s.Description = "The certificate becomes valid on " +
			details.NotBefore.Time().UTC().Format(dateLayout) + "."
	case now >= details.NotAfter:
		s.Variant = status.VariantError
		s.Message = "Certificate expired"
		s.Description = "The certificate expired on " + date +
			". Replace it with a new certificate."
	case now >= details.NotAfter.Add(-expiryWarning):
		s.Variant = status.VariantWarning
		s.Message = "Certificate expires soon"
		s.Description = "The certificate expires on " + date +
			". Replace it before then."
	}
	return s
}

// unchanged reports whether next says what prev says, so next needs no write.
func unchanged(next, prev status.Status[any]) bool {
	return next.Variant == prev.Variant && next.Message == prev.Message &&
		next.Description == prev.Description && next.Name == prev.Name
}

// writeExpiry writes the expiry status of each certificate whose status changed.
func (s *Service) writeExpiry(ctx context.Context) error {
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var certs []Certificate
		if err := s.NewRetrieve().Entries(&certs).Exec(ctx, tx); err != nil {
			return err
		}
		now := s.cfg.Now()
		next := make(map[resource.ID]status.Status[any], len(certs))
		for _, c := range certs {
			if st, ok := expiryStatus(c, now); ok {
				next[*st.Owner] = st
			}
		}
		if len(next) == 0 {
			return nil
		}
		var prev []status.Status[any]
		if err := s.cfg.Status.NewRetrieve[any]().
			Where(status.MatchOwners[any](lo.Keys(next)...)).
			Entries(&prev).
			Exec(ctx, tx); err != nil {
			return err
		}
		for _, p := range prev {
			if unchanged(next[*p.Owner], p) {
				delete(next, *p.Owner)
			}
		}
		w := s.cfg.Status.NewWriter(tx)
		for _, c := range certs {
			if st, ok := next[c.ResourceID()]; ok {
				if err := w.Set(ctx, &st); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ownPrefix is the start of the key of the status of each certificate that the node
// presents.
func (s *Service) ownPrefix() string {
	return node.ResourceID(s.cfg.HostKey).String() + ":certificate:"
}

// caPrefix is the start of the key of the status of each CA that the node presents.
func (s *Service) caPrefix() string { return s.ownPrefix() + "ca:" }

// ownCheck is the result of one read of the certificates that the node presents.
type ownCheck struct {
	// next is the expiry status of each certificate that the check read, by key.
	next map[status.Key]status.Status[any]
	// unread holds the status keys of the listener certificates that the check could
	// not read.
	unread set.Set[status.Key]
	// caUnread is true when the check could not read the CA file.
	caUnread bool
}

// readOwn reads each certificate that the node presents and returns its expiry status
// at now: each CA in the CA file, keyed by its fingerprint, and the certificate of each
// listener, keyed by its address. It logs each certificate that it cannot read.
func (s *Service) readOwn(now telem.TimeStamp) ownCheck {
	c := ownCheck{
		next:   make(map[status.Key]status.Status[any]),
		unread: make(set.Set[status.Key]),
	}
	add := func(key, name string, details Details) {
		st := validityStatus(name, details, now)
		st.Key = key
		c.next[key] = st
	}
	if s.cfg.CA != nil {
		cas, err := s.cfg.CA.LoadCAs()
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			c.caUnread = true
			s.cfg.L.Error("failed to read the CA certificates", zap.Error(err))
		}
		for _, ca := range cas {
			details := newDetails(ca)
			add(
				s.caPrefix()+details.Sha256,
				"CA certificate "+details.Sha256[:8],
				details,
			)
		}
	}
	for _, l := range s.cfg.Listeners {
		key := s.ownPrefix() + string(l.Address)
		leaf, err := readLeaf(l.Source)
		if err != nil {
			c.unread.Add(key)
			s.cfg.L.Error(
				"failed to read the certificate of a listener",
				zap.Stringer("address", l.Address),
				zap.Error(err),
			)
			continue
		}
		add(key, "Certificate of "+string(l.Address), newDetails(leaf))
	}
	return c
}

// readLeaf returns the leaf certificate that src presents to a client that sends no
// server name.
func readLeaf(src cert.Source) (*x509.Certificate, error) {
	c, err := src.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(c.Certificate[0])
}

// writeOwnExpiry writes the expiry status of each certificate that the node presents
// whose status changed, and deletes the status of each one that it no longer presents.
// The status of a certificate that the check cannot read stays as it is.
func (s *Service) writeOwnExpiry(ctx context.Context) error {
	c := s.readOwn(s.cfg.Now())
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var prev []status.Status[any]
		if err := s.cfg.Status.NewRetrieve[any]().
			WherePrefix(s.ownPrefix()).
			Entries(&prev).
			Exec(ctx, tx); err != nil {
			return err
		}
		var stale []status.Key
		for _, p := range prev {
			n, read := c.next[p.Key]
			kept := c.unread.Contains(p.Key) ||
				(c.caUnread && strings.HasPrefix(p.Key, s.caPrefix()))
			switch {
			case read && unchanged(n, p):
				delete(c.next, p.Key)
			case !read && !kept:
				stale = append(stale, p.Key)
			}
		}
		w := s.cfg.Status.NewWriter(tx)
		if err := w.Delete(ctx, stale...); err != nil {
			return err
		}
		for _, st := range c.next {
			if err := w.Set(ctx, &st); err != nil {
				return err
			}
		}
		return nil
	})
}
