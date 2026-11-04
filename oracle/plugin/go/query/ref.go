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
	FieldName   string // oracle field name, used as the reference's name
	GoName      string // PascalCase field name
	StructField string // camelCase field on the indexes struct
	ElemGoType  string // Go type of one target: a resource key type or the ID type
	// Tables are the names of the tables the reference may point at.
	Tables   []string
	OnDelete string // "Restrict", "Cascade", or "Detach"
	Unique   bool
	Acyclic  bool
	// IsOptional is true when the field is a pointer.
	IsOptional bool
	IsSlice    bool
	// IsID is true when each target is a resource ID instead of the key of one table.
	IsID bool
}

// TargetOf returns the Go expression that converts expr, one value of the field, to a
// gorp.Target.
func (r refInfo) TargetOf(expr string) string {
	if r.IsID {
		return fmt.Sprintf("resourceTarget(%s)", expr)
	}
	return fmt.Sprintf("gorp.To(%q, %s)", r.Tables[0], expr)
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
	names set.Set[string]
}

func newResourceTypes(table *resolution.Table) resourceTypes {
	rt := resourceTypes{byKey: make(map[string]string), names: make(set.Set[string])}
	for _, typ := range slices.Concat(table.StructTypes(), table.UnionTypes()) {
		name := plugindomain.GetStringFromType(typ, resourceDomain, "type")
		if name == "" {
			continue
		}
		rt.names.Add(name)
		for _, f := range retrieveFields(typ, table) {
			if plugindomain.HasDomainFromField(f, "key") {
				if _, ok := rt.byKey[f.Type.Name]; !ok {
					rt.byKey[f.Type.Name] = name
				}
				break
			}
		}
	}
	return rt
}

func extractRef(
	typ resolution.Type,
	field resolution.Field,
	ownTable string,
	rt resourceTypes,
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
	onDelete, ok := onDeleteNames[plugindomain.GetStringFromField(field, "ref", "on_delete")]
	if !ok {
		return fail("needs on_delete cascade, restrict, or detach")
	}
	isOptional := field.Optional && !isSlice
	if onDelete == "Detach" && !isOptional && !isSlice {
		return fail("needs to be optional or an array to detach")
	}
	types := refTypes(field)
	info := refInfo{
		FieldName:   field.Name,
		GoName:      naming.GetFieldName(field),
		StructField: naming.LowerFirst(naming.GetFieldName(field)),
		ElemGoType:  r.ResolveTypeRef(elem, ctx),
		OnDelete:    onDelete,
		Unique:      plugindomain.HasExprFromField(field, "ref", "unique"),
		Acyclic:     plugindomain.HasExprFromField(field, "ref", "acyclic"),
		IsOptional:  isOptional,
		IsSlice:     isSlice,
		IsID:        elem.Name == resourceIDType,
	}
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
