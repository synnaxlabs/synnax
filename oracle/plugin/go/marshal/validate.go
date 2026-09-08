// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal

import (
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// validateEntryRefs checks that every struct or union the entry's persisted
// wire format dispatches to carries its own @go marshal declaration. The
// generated codec calls EncodeOrc/DecodeOrc on such references, so an untagged
// target only compiles while a stale codec file lingers on disk.
func validateEntryRefs(entry resolution.Type, table *resolution.Table) error {
	var walkRef func(ref resolution.TypeRef, fieldName string) error
	walkRef = func(ref resolution.TypeRef, fieldName string) error {
		if ref.TypeParam != nil {
			return nil
		}
		// Container elements dispatch to their own codecs; type arguments of a
		// named generic do not — the generic's codec encodes them as JSON.
		if ref.Name == "Array" || ref.Name == "Map" {
			for _, arg := range ref.TypeArgs {
				if err := walkRef(arg, fieldName); err != nil {
					return err
				}
			}
			return nil
		}
		if resolution.IsPrimitive(ref.Name) {
			return nil
		}
		resolved, ok := ref.Resolve(table)
		if !ok {
			return nil
		}
		switch form := resolved.Form.(type) {
		case resolution.StructForm, resolution.UnionForm:
			// Synthetic payloads of inline union variants carry no codec of
			// their own; the union codec inlines their fields.
			if resolved.Synthetic {
				fields := schemadiff.PersistedFields(
					resolution.UnifiedFields(resolved, table),
				)
				for _, f := range fields {
					if f.Type.Name == "nil" {
						continue
					}
					if err := walkRef(f.Type, fieldName+"."+f.Name); err != nil {
						return err
					}
				}
				return nil
			}
			if !domain.HasExprFromType(resolved, "go", "marshal") {
				return errors.Newf(
					"cannot generate codec for %s: field %q references %s, "+
						"which has no @go marshal; tag its declaration in the "+
						"defining version file, or use \"@go marshal hand\" "+
						"for hand-written codec methods",
					entry.Name, fieldName, resolved.Name,
				)
			}
		case resolution.DistinctForm:
			if domain.GetStringFromType(resolved, "go", "marshal") == "flex" {
				return nil
			}
			return walkRef(form.Base, fieldName)
		case resolution.AliasForm:
			return walkRef(form.Target, fieldName)
		}
		return nil
	}
	fields := schemadiff.PersistedFields(resolution.UnifiedFields(entry, table))
	for _, f := range fields {
		if f.Type.Name == "nil" {
			continue
		}
		if err := walkRef(f.Type, f.Name); err != nil {
			return err
		}
	}
	if uform, ok := entry.Form.(resolution.UnionForm); ok {
		for _, ext := range uform.Extends {
			if err := walkRef(ext, "extends"); err != nil {
				return err
			}
		}
		for _, v := range uform.Variants {
			if err := walkRef(v.Type, v.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// handCodec reports whether the type declares @go marshal hand: its codec
// methods are hand-written, so no codec is generated but references to it
// from other codecs remain valid.
func handCodec(t resolution.Type) bool {
	return domain.GetStringFromType(t, "go", "marshal") == "hand"
}
