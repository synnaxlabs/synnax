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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
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

// ownExpiry returns the expiry status at now of each certificate that the node
// presents, by key: each CA in the CA file, by its fingerprint, and the certificate of
// each listener, by its address.
func (s *Service) ownExpiry(
	now telem.TimeStamp,
) (map[status.Key]status.Status[any], error) {
	next := make(map[status.Key]status.Status[any])
	add := func(key, name string, details Details) {
		st := validityStatus(name, details, now)
		st.Key = s.ownPrefix() + key
		next[st.Key] = st
	}
	if s.cfg.CA != nil {
		cas, err := s.cfg.CA.LoadCAs()
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, errors.Wrap(err, "CA certificate")
		}
		for _, ca := range cas {
			details := newDetails(ca)
			add("ca:"+details.Sha256, "CA certificate", details)
		}
	}
	for _, l := range s.cfg.Listeners {
		c, err := l.Source.GetCertificate(&tls.ClientHelloInfo{})
		if err != nil {
			return nil, errors.Wrapf(err, "certificate of %s", l.Address)
		}
		leaf, err := x509.ParseCertificate(c.Certificate[0])
		if err != nil {
			return nil, errors.Wrapf(err, "certificate of %s", l.Address)
		}
		add(string(l.Address), "Certificate of "+string(l.Address), newDetails(leaf))
	}
	return next, nil
}

// writeOwnExpiry writes the expiry status of each certificate that the node presents
// whose status changed, and deletes the status of each one that it no longer presents.
func (s *Service) writeOwnExpiry(ctx context.Context) error {
	next, err := s.ownExpiry(s.cfg.Now())
	if err != nil {
		return err
	}
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
			n, presented := next[p.Key]
			switch {
			case !presented:
				stale = append(stale, p.Key)
			case unchanged(n, p):
				delete(next, p.Key)
			}
		}
		w := s.cfg.Status.NewWriter(tx)
		if err := w.Delete(ctx, stale...); err != nil {
			return err
		}
		for _, st := range next {
			if err := w.Set(ctx, &st); err != nil {
				return err
			}
		}
		return nil
	})
}
