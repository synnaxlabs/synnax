// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/key"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const (
	// resourceDomain is the domain whose type expression names a resource type.
	resourceDomain = "resource"
	// resourceIDType is the qualified name of the type that identifies any resource.
	resourceIDType = "resource.ID"
)

// refInfo holds extracted data about a @ref-annotated field.
type refInfo struct {
	// FieldName is the Oracle field name, used as the reference's name.
	FieldName string
	// GoName is the PascalCase field name.
	GoName string
	// StructField is the camelCase field on the indexes struct.
	StructField string
	// ElemGoType is the Go type of one target: a resource key type or the ID type.
	ElemGoType string
	// Tables are the names of the tables the reference may point at.
	Tables []string
	// OnDelete is "Restrict", "Cascade", or "Detach".
	OnDelete string
	// Unique is true when at most one entry may hold each target.
	Unique bool
	// Acyclic is true when a chain of references within the table may not return to
	// its start.
	Acyclic bool
	// IsOptional is true when the field is a pointer.
	IsOptional bool
	// IsSlice is true when the field is an array of targets.
	IsSlice bool
	// IsID is true when each target is a resource ID instead of the key of one table.
	IsID bool
	// Zero is the Go zero value of a scalar field that holds no target while it is
	// zero, because its default is zero. It is empty for any other field.
	Zero string
}

// HoldsNone reports whether an entry can hold no target through the field.
func (r refInfo) HoldsNone() bool { return r.IsOptional || r.IsSlice || r.Zero != "" }

// ZeroOperand returns Zero in a form that can follow == in Go.
func (r refInfo) ZeroOperand() string {
	if strings.HasSuffix(r.Zero, "{}") {
		return "(" + r.Zero + ")"
	}
	return r.Zero
}

// TargetType returns the Go type of one target: the key type for a reference to one
// table, or gorp.Target for a reference to resource IDs.
func (r refInfo) TargetType() string {
	if r.IsID {
		return "gorp.Target"
	}
	return r.ElemGoType
}

// TargetOf returns the Go expression that converts expr, one value of the field, to a
// target of TargetType.
func (r refInfo) TargetOf(expr string) string {
	if r.IsID {
		return fmt.Sprintf("resourceTarget(%s)", expr)
	}
	return expr
}

// GorpType returns the Go type of the reference on a table of entries entry with keys
// of type key.
func (r refInfo) GorpType(key, entry string) string {
	if r.IsID {
		return fmt.Sprintf("gorp.TargetReference[%s, %s]", key, entry)
	}
	return fmt.Sprintf("gorp.KeyReference[%s, %s, %s]", key, entry, r.ElemGoType)
}

// Constructor returns the Go expression that opens the config literal of the reference
// on a table of entries entry with keys of type key.
func (r refInfo) Constructor(key, entry string) string {
	if r.IsID {
		return fmt.Sprintf(
			"gorp.NewTargetReference(gorp.TargetReferenceConfig[%s, %s]",
			key,
			entry,
		)
	}
	return fmt.Sprintf(
		"gorp.NewKeyReference(gorp.KeyReferenceConfig[%s, %s, %s]",
		key,
		entry,
		r.ElemGoType,
	)
}

// TablesField returns the config field and Go literal that name the reference's
// target tables.
func (r refInfo) TablesField() string {
	if r.IsID {
		return fmt.Sprintf("Tables:   []string{%s},", r.QuotedTables())
	}
	return fmt.Sprintf("Table:    %q,", r.Tables[0])
}

// QuotedTables returns Tables as a comma-separated list of Go string literals.
func (r refInfo) QuotedTables() string {
	quoted := make([]string, len(r.Tables))
	for i, t := range r.Tables {
		quoted[i] = strconv.Quote(t)
	}
	return strings.Join(quoted, ", ")
}

var onDeleteNames = map[string]string{
	"restrict": "Restrict",
	"cascade":  "Cascade",
	"detach":   "Detach",
}

// resourceTypes maps the qualified name of each resource key type to the resource
// type it identifies, and holds every resource type name.
type resourceTypes struct {
	byKey map[string]string
	// conflicts maps a key type that identifies more than one resource type to the
	// second resource type it identifies.
	conflicts map[string]string
	names     set.Set[string]
}

func newResourceTypes(table *resolution.Table) resourceTypes {
	rt := resourceTypes{
		byKey:     make(map[string]string),
		conflicts: make(map[string]string),
		names:     make(set.Set[string]),
	}
	for _, typ := range slices.Concat(table.StructTypes(), table.UnionTypes()) {
		name := plugindomain.GetStringFromType(typ, resourceDomain, "type")
		if name == "" {
			continue
		}
		rt.names.Add(name)
		fields, _ := resolution.RecordFields(typ, table)
		for _, f := range fields {
			if plugindomain.HasDomainFromField(f, "key") {
				if _, ok := rt.byKey[f.Type.Name]; ok {
					rt.conflicts[f.Type.Name] = name
				} else {
					rt.byKey[f.Type.Name] = name
				}
				break
			}
		}
	}
	return rt
}

// refExpressions are the expressions a @ref domain may hold.
var refExpressions = set.New("on_delete", "types", "unique", "acyclic")

// extractRef returns the reference that field of typ holds. inArray is true when a
// detach can drop the array element that holds field.
func extractRef(
	typ resolution.Type,
	field resolution.Field,
	ownTable string,
	inArray bool,
	rt resourceTypes,
	table *resolution.Table,
	r *resolver.Resolver,
	ctx *resolver.Context,
) (refInfo, error) {
	fail := func(format string, args ...any) (refInfo, error) {
		return refInfo{}, errors.Newf(
			"%s: @ref field %s %s",
			typ.QualifiedName,
			field.Name,
			fmt.Sprintf(format, args...),
		)
	}
	elem := field.Type
	isSlice := elem.Name == "Array" && len(elem.TypeArgs) == 1
	if isSlice {
		elem = elem.TypeArgs[0]
	}
	for _, expr := range field.Domains["ref"].Expressions {
		if !refExpressions.Contains(expr.Name) {
			return fail("has unknown expression %s", expr.Name)
		}
	}
	onDelete, ok := onDeleteNames[plugindomain.GetStringFromField(
		field,
		"ref",
		"on_delete",
	)]
	if !ok {
		return fail("needs on_delete cascade, restrict, or detach")
	}
	isOptional := field.Optional && !isSlice
	var zero string
	if !isOptional && !isSlice && isZeroDefault(field.Default) {
		zero = zeroValue(elem, table, func() string {
			return r.ResolveTypeRef(elem, ctx)
		})
	}
	if onDelete == "Detach" && !isOptional && !isSlice && zero == "" && !inArray {
		return fail("needs to be optional, an array, or zero by default to detach")
	}
	types := refTypes(field)
	info := refInfo{
		FieldName:   field.Name,
		GoName:      naming.GetFieldName(field),
		StructField: naming.LowerFirst(naming.GetFieldName(field)),
		OnDelete:    onDelete,
		Unique:      plugindomain.HasExprFromField(field, "ref", "unique"),
		Acyclic:     plugindomain.HasExprFromField(field, "ref", "acyclic"),
		IsOptional:  isOptional,
		IsSlice:     isSlice,
		IsID:        elem.Name == resourceIDType,
		Zero:        zero,
	}
	info.ElemGoType = r.ResolveTypeRef(elem, ctx)
	if info.IsID {
		if len(types) == 0 {
			return fail("needs types to name the resource types it may reference")
		}
		for _, t := range types {
			if !rt.names.Contains(t) {
				return fail("names unknown resource type %s", t)
			}
		}
		info.Tables = types
	} else {
		if len(types) > 0 {
			return fail("can name types only when it holds resource IDs")
		}
		target, ok := rt.byKey[elem.Name]
		if !ok {
			return fail("needs the key type of a resource, not %s", elem.Name)
		}
		if other, ok := rt.conflicts[elem.Name]; ok {
			return fail(
				"has key type %s, which identifies both %s and %s",
				elem.Name,
				target,
				other,
			)
		}
		info.Tables = []string{target}
	}
	if info.Acyclic && !slices.Contains(info.Tables, ownTable) {
		return fail("can be acyclic only when it references its own type")
	}
	return info, nil
}

// refTypes returns the resource types named by the types expression of field's @ref.
func refTypes(field resolution.Field) []string {
	domain, ok := field.Domains["ref"]
	if !ok {
		return nil
	}
	expr, ok := domain.Expressions.Find("types")
	if !ok {
		return nil
	}
	types := make([]string, 0, len(expr.Values))
	for _, v := range expr.Values {
		if v.IdentValue != "" {
			types = append(types, v.IdentValue)
		} else if v.StringValue != "" {
			types = append(types, v.StringValue)
		}
	}
	return types
}

// isZeroDefault reports whether v is the zero of a string or integer field.
func isZeroDefault(v *resolution.ExpressionValue) bool {
	if v == nil {
		return false
	}
	switch v.Kind {
	case resolution.ValueKindString:
		return v.StringValue == ""
	case resolution.ValueKindInt:
		return v.IntValue == 0
	}
	return false
}

// zeroValue returns the Go zero value of the type that ref names. goType returns the
// Go type of ref.
func zeroValue(
	ref resolution.TypeRef,
	table *resolution.Table,
	goType func() string,
) string {
	switch p := key.ResolvePrimitive(ref, table); {
	case p == "string":
		return `""`
	case strings.HasPrefix(p, "int"), strings.HasPrefix(p, "uint"):
		return "0"
	default:
		return goType() + "{}"
	}
}
