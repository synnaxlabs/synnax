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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ReferenceConfig configures a reference from an entry to the certificates that it
// names.
type ReferenceConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name names the reference in errors. Use the name of the field that holds it.
	//
	// [REQUIRED]
	Name string
	// Targets returns the keys of the certificates that e names, or nil when it names
	// none.
	//
	// [REQUIRED]
	Targets func(e *E) []Key
	// Rack returns the rack that reads the certificates of e.
	//
	// [REQUIRED]
	Rack func(e *E) rack.Key
}

// NewReference returns a reference from an entry to the certificates that it names. A
// certificate cannot be deleted while an entry names it. A write of the entry fails
// with a validation error when it names a file certificate of a rack other than the
// entry's rack.
func NewReference[K gorp.Key, E gorp.Entry[K]](
	certs *Service,
	cfg ReferenceConfig[K, E],
) *gorp.KeyReference[K, E, Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[K, E, Key]{
		Name:     cfg.Name,
		Table:    tableName,
		OnDelete: gorp.Restrict,
		Kind:     gorp.Use,
		Targets:  cfg.Targets,
		Check: func(ctx context.Context, tx gorp.Tx, e *E, target Key) error {
			var c Certificate
			if err := certs.NewRetrieve().
				Where(MatchKeys(target)).
				Entry(&c).
				Exec(ctx, tx); err != nil {
				return err
			}
			file, ok := c.Variant.(FileCertificate)
			if r := cfg.Rack(e); ok && file.Rack != r {
				return errors.Wrapf(
					validate.ErrValidation,
					"certificate %q is a file of rack %d, not of rack %d",
					c.Base().Name,
					file.Rack,
					r,
				)
			}
			return nil
		},
	})
}
