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
	w := refWalker{entry: entry, table: table}
	if err := w.walkFields(entry, ""); err != nil {
		return err
	}
	if uform, ok := entry.Form.(resolution.UnionForm); ok {
		for _, ext := range uform.Extends {
			if err := w.walk(ext, "extends"); err != nil {
				return err
			}
		}
		for _, v := range uform.Variants {
			if err := w.walk(v.Type, v.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// refWalker walks the type references of an entry's persisted wire format.
type refWalker struct {
	entry resolution.Type
	table *resolution.Table
}

// walkFields walks each persisted field of t. prefix, when not empty, qualifies the
// field names in errors.
func (w refWalker) walkFields(t resolution.Type, prefix string) error {
	fields := schemadiff.PersistedFields(resolution.UnifiedFields(t, w.table))
	for _, f := range fields {
		if f.Type.Name == "nil" {
			continue
		}
		fieldName := f.Name
		if prefix != "" {
			fieldName = prefix + "." + f.Name
		}
		if err := w.walk(f.Type, fieldName); err != nil {
			return err
		}
	}
	return nil
}

func (w refWalker) walk(ref resolution.TypeRef, fieldName string) error {
	if ref.TypeParam != nil {
		return nil
	}
	// Container elements dispatch to their own codecs; type arguments of a named
	// generic do not — the generic's codec encodes them as JSON.
	if ref.Name == "Array" || ref.Name == "Map" {
		for _, arg := range ref.TypeArgs {
			if err := w.walk(arg, fieldName); err != nil {
				return err
			}
		}
		return nil
	}
	if resolution.IsPrimitive(ref.Name) {
		return nil
	}
	resolved, ok := ref.Resolve(w.table)
	if !ok {
		return nil
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm, resolution.UnionForm:
		return w.walkNamed(resolved, fieldName)
	case resolution.DistinctForm:
		if domain.GetStringFromType(resolved, "go", "marshal") == "flex" {
			return nil
		}
		return w.walk(form.Base, fieldName)
	case resolution.AliasForm:
		return w.walk(form.Target, fieldName)
	}
	return nil
}

// walkNamed checks a referenced struct or union.
func (w refWalker) walkNamed(resolved resolution.Type, fieldName string) error {
	// Synthetic payloads of inline union variants carry no codec of their own; the
	// union codec inlines their fields.
	if resolved.Synthetic {
		return w.walkFields(resolved, fieldName)
	}
	if !domain.HasExprFromType(resolved, "go", "marshal") {
		return errors.Newf(
			"cannot generate codec for %s: field %q references %s, "+
				"which has no @go marshal; tag its declaration in the "+
				"defining version file, or use \"@go marshal hand\" "+
				"for hand-written codec methods",
			w.entry.Name, fieldName, resolved.Name,
		)
	}
	return nil
}

// handCodec reports whether the type declares @go marshal hand: its codec
// methods are hand-written, so no codec is generated but references to it
// from other codecs remain valid.
func handCodec(t resolution.Type) bool {
	return domain.GetStringFromType(t, "go", "marshal") == "hand"
}
