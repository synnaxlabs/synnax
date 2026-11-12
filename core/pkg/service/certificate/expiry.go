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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
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
	base := c.Base()
	date := details.NotAfter.Time().UTC().Format(dateLayout)
	s := status.Status[any]{
		Owner:       new(c.ResourceID()),
		Name:        base.Name,
		Time:        now,
		Variant:     status.VariantSuccess,
		Message:     "Certificate valid",
		Description: "The certificate expires on " + date + ".",
	}
	// A NotAfter past the last time stamp saturates, so its date is not the real one.
	if details.NotAfter == telem.TimeStampMax {
		s.Description = "The certificate does not expire."
	}
	if now < details.NotBefore {
		s.Variant = status.VariantWarning
		s.Message = "Certificate not yet valid"
		s.Description = "The certificate becomes valid on " +
			details.NotBefore.Time().UTC().Format(dateLayout) + "."
	} else if now >= details.NotAfter {
		s.Variant = status.VariantError
		s.Message = "Certificate expired"
		s.Description = "The certificate expired on " + date +
			". Replace it with a new certificate."
	} else if now >= details.NotAfter.Add(-expiryWarning) {
		s.Variant = status.VariantWarning
		s.Message = "Certificate expires soon"
		s.Description = "The certificate expires on " + date +
			". Replace it before then."
	}
	return s, true
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
			n := next[*p.Owner]
			if n.Variant == p.Variant && n.Message == p.Message &&
				n.Description == p.Description && n.Name == p.Name {
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
