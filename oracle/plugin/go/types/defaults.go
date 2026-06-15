// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/resolution"
)

const validateImportPath = "github.com/synnaxlabs/x/validate"

// defaultFillData describes a single field assignment emitted by a generated
// ApplyDefaults method: when the field equals ZeroLit it is set to Expr.
type defaultFillData struct {
	GoName  string
	ZeroLit string
	Expr    string
}

// enumCheckData describes a single enum-membership assertion emitted by a generated
// Validate method.
type enumCheckData struct {
	GoName    string
	FieldName string
}

// goDefaultFill returns the fill for a required field whose static default differs
// from its type's zero value (RFC 0043 section 5). It returns ok=false for fields with
// no default, nullable/optional fields, defaults that equal the zero value (nothing to
// fill), and non-static defaults such as `create`/`now`.
func goDefaultFill(field resolution.Field, data *templateData) (defaultFillData, bool) {
	if field.Default == nil || field.Optional {
		return defaultFillData{}, false
	}
	name := naming.GetFieldName(field)
	d := field.Default
	switch d.Kind {
	case resolution.ValueKindString:
		if d.StringValue == "" {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: `""`, Expr: strconv.Quote(d.StringValue)}, true
	case resolution.ValueKindInt:
		if d.IntValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: "0", Expr: fmt.Sprintf("%d", d.IntValue)}, true
	case resolution.ValueKindFloat:
		if d.FloatValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: "0", Expr: strconv.FormatFloat(d.FloatValue, 'g', -1, 64)}, true
	case resolution.ValueKindIdent:
		ev, ok := validation.ResolveEnumVariant(d.IdentValue, field.Type, data.table)
		if !ok {
			return defaultFillData{}, false
		}
		form, _ := ev.Type.Form.(resolution.EnumForm)
		if form.IsIntEnum {
			// A valid integer-enum default is the zeroth member (the zero value), so
			// there is nothing to fill; a non-zeroth default is an invariant violation.
			return defaultFillData{}, false
		}
		enumType := stripPointer(data.resolver.ResolveTypeRef(field.Type, data.ctx))
		return defaultFillData{
			GoName:  name,
			ZeroLit: `""`,
			Expr:    enumType + naming.ToPascalCase(ev.Variant.Name),
		}, true
	}
	return defaultFillData{}, false
}

// goEnumCheck returns an enum-membership validation for a required field whose type is
// an enum (RFC 0043 section 5.2). Nullable and optional fields are skipped: their
// pointer may be nil, and absence is legitimate.
func goEnumCheck(field resolution.Field, data *templateData) (enumCheckData, bool) {
	if field.Optional {
		return enumCheckData{}, false
	}
	resolved, ok := field.Type.Resolve(data.table)
	if !ok {
		return enumCheckData{}, false
	}
	if _, ok := resolved.Form.(resolution.EnumForm); !ok {
		return enumCheckData{}, false
	}
	return enumCheckData{GoName: naming.GetFieldName(field), FieldName: naming.GetFieldName(field)}, true
}

// stripPointer removes a leading pointer marker from a resolved Go type.
func stripPointer(goType string) string { return strings.TrimPrefix(goType, "*") }
