// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ReferenceConfig configures a reference from an entry to the secrets that it names.
type ReferenceConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name names the reference in errors. Use the name of the field that holds it.
	//
	// [REQUIRED]
	Name string
	// Targets returns the keys of the secrets that e names, or nil when it names none.
	//
	// [REQUIRED]
	Targets func(e *E) []Key
	// Rack returns the rack that reads the secrets of e.
	//
	// [REQUIRED]
	Rack func(e *E) rack.Key
	// Detach removes the secret with key target from e.
	//
	// [OPTIONAL]
	Detach func(e *E, target Key)
}

// NewReference returns a reference from an entry to the secrets that it names. A
// secret cannot be deleted while an entry names it. A write of the entry fails with a
// validation error when it names a file secret of a rack other than the entry's rack.
func NewReference[K gorp.Key, E gorp.Entry[K]](
	secrets *Service,
	cfg ReferenceConfig[K, E],
) *gorp.KeyReference[K, E, Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[K, E, Key]{
		Name:     cfg.Name,
		Table:    tableName,
		OnDelete: gorp.Restrict,
		Kind:     gorp.Use,
		Targets:  cfg.Targets,
		Detach:   cfg.Detach,
		Check: func(ctx context.Context, tx gorp.Tx, e *E, target Key) error {
			var s Secret
			if err := secrets.NewRetrieve().
				Where(MatchKeys(target)).
				Entry(&s).
				Exec(ctx, tx); err != nil {
				return err
			}
			file, ok := s.Variant.(FileSecret)
			if r := cfg.Rack(e); ok && file.Rack != r {
				return errors.Wrapf(
					validate.ErrValidation,
					"secret %q is a file of rack %d, not of rack %d",
					s.Base().Name,
					file.Rack,
					r,
				)
			}
			return nil
		},
	})
}
