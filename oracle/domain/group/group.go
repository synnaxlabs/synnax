// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package group extracts @group domain data: the record types that can sit in a group.
// A record marks the field that holds its group with @group. scope names the scope of
// the groups that can hold the record, and pinned says that the top parent of the group
// must be the record's holder, or none when the record has no holder.
package group

import (
	"cmp"
	"slices"

	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Data is the @group data of a schema set.
type Data struct {
	// Record is the group record that the marked fields hold the key of.
	Record resolution.Type
	// Enum is the enum of resource types, the type of the group record's scope field.
	Enum resolution.Type
	// Members holds the record types that can sit in a group, ordered by value.
	Members []Member
}

// Member is a record type that can sit in a group.
type Member struct {
	// Type is the resource type of the record.
	Type resolution.EnumValue
	// Scope is the scope of the groups that can hold the record.
	Scope resolution.EnumValue
	// Pinned is true when the top parent of the group must be the record's holder.
	Pinned bool
}

// Extract returns the @group data of table, or nil when no field carries @group. A type
// in a version file is skipped. It returns an error when a marked field does not hold
// the key of one group record, when a record or scope is not a member of the group
// record's scope enum, or when two fields mark the same resource type.
func Extract(table *resolution.Table) (*Data, error) {
	var (
		data   *Data
		marked = make(set.Set[string])
	)
	for _, typ := range table.StructTypes() {
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		// An extending type inherits the field, so only the declaring type counts.
		for _, f := range typ.Form.(resolution.StructForm).Fields {
			domain, ok := f.Domains["group"]
			if !ok {
				continue
			}
			if data == nil {
				d, err := newData(f, table)
				if err != nil {
					return nil, errors.Wrapf(err, "%s.%s", typ.QualifiedName, f.Name)
				}
				data = &d
			}
			m, err := newMember(typ, f, domain, *data, table)
			if err == nil && marked.Contains(m.Type.StringValue()) {
				err = errors.Newf(
					"@group marks resource type %s a second time",
					m.Type.StringValue(),
				)
			}
			if err != nil {
				return nil, errors.Wrapf(err, "%s.%s", typ.QualifiedName, f.Name)
			}
			marked.Add(m.Type.StringValue())
			data.Members = append(data.Members, m)
		}
	}
	if data != nil {
		slices.SortFunc(data.Members, func(a, b Member) int {
			return cmp.Compare(a.Type.StringValue(), b.Type.StringValue())
		})
	}
	return data, nil
}

// newData finds the group record whose key f holds, and the enum of its scope field.
func newData(f resolution.Field, table *resolution.Table) (Data, error) {
	for _, typ := range table.StructTypes() {
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		if keyName(typ, table) != f.Type.Name {
			continue
		}
		fields := resolution.UnifiedFields(typ, table)
		i := slices.IndexFunc(fields, func(f resolution.Field) bool {
			return f.Name == "scope"
		})
		if i < 0 {
			return Data{}, errors.Newf(
				"@group needs %s to have a scope field", typ.QualifiedName,
			)
		}
		enum, _, ok := resolution.Dealias(fields[i].Type, table)
		if _, isEnum := enum.Form.(resolution.EnumForm); !ok || !isEnum {
			return Data{}, errors.Newf(
				"@group needs the scope of %s to be an enum", typ.QualifiedName,
			)
		}
		return Data{Record: typ, Enum: enum}, nil
	}
	return Data{}, errors.Newf("@group holds %s, which is no record's key", f.Type.Name)
}

// newMember returns the member that f, a field of typ marked with domain, declares.
func newMember(
	typ resolution.Type,
	f resolution.Field,
	domain resolution.Domain,
	data Data,
	table *resolution.Table,
) (Member, error) {
	if f.Type.Name != keyName(data.Record, table) {
		return Member{}, errors.Newf(
			"@group holds %s, not the key of %s",
			f.Type.Name,
			data.Record.QualifiedName,
		)
	}
	m := Member{}
	for _, expr := range domain.Expressions {
		switch expr.Name {
		case "scope":
			if len(expr.Values) != 1 {
				return Member{}, errors.Newf(
					"@group needs one value for scope, not %d", len(expr.Values),
				)
			}
			scope, ok := enumMember(data.Enum, valueOf(expr.Values[0]))
			if !ok {
				return Member{}, errors.Newf(
					"@group has scope %s, which %s does not hold",
					valueOf(expr.Values[0]),
					data.Enum.QualifiedName,
				)
			}
			m.Scope = scope
		case "pinned":
			if len(expr.Values) > 0 {
				return Member{}, errors.New("@group needs no value for pinned")
			}
			m.Pinned = true
		default:
			return Member{}, errors.Newf("@group has unknown expression %s", expr.Name)
		}
	}
	if m.Scope.Name == "" {
		return Member{}, errors.New("@group needs a scope")
	}
	var name string
	if expr, ok := typ.Domains["resource"].Expressions.Find("type"); ok &&
		len(expr.Values) > 0 {
		name = valueOf(expr.Values[0])
	}
	if name == "" {
		return Member{}, errors.New("@group needs a type with a @resource type")
	}
	var ok bool
	if m.Type, ok = enumMember(data.Enum, name); !ok {
		return Member{}, errors.Newf(
			"@group has resource type %s, which %s does not hold",
			name,
			data.Enum.QualifiedName,
		)
	}
	return m, nil
}

// keyName returns the name of the type of typ's key field, or "" when it has none.
func keyName(typ resolution.Type, table *resolution.Table) string {
	for _, f := range resolution.UnifiedFields(typ, table) {
		if key.HasKey(f) {
			return f.Type.Name
		}
	}
	return ""
}

// enumMember returns the member of enum whose value is value.
func enumMember(enum resolution.Type, value string) (resolution.EnumValue, bool) {
	form := enum.Form.(resolution.EnumForm)
	i := slices.IndexFunc(form.Values, func(v resolution.EnumValue) bool {
		return v.StringValue() == value
	})
	if i < 0 {
		return resolution.EnumValue{}, false
	}
	return form.Values[i], true
}

// valueOf returns the text of v, which may be a string or an identifier.
func valueOf(v resolution.ExpressionValue) string {
	if v.StringValue != "" {
		return v.StringValue
	}
	return v.IdentValue
}
