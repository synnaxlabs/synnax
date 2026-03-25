// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/gorp"
)

// BuildLayout converts an Oracle resolved type into a gorp.FieldLayout tree that
// describes its binary encoding format. This layout is used by the schema-driven
// byte resolver to transform binary data between schema versions.
func BuildLayout(
	typ resolution.Type,
	table *resolution.Table,
) ([]gorp.FieldLayout, error) {
	fields := resolution.UnifiedFields(typ, table)
	return buildFieldLayouts(fields, table, make(map[string]bool))
}

func buildFieldLayouts(
	fields []resolution.Field,
	table *resolution.Table,
	visiting map[string]bool,
) ([]gorp.FieldLayout, error) {
	var layouts []gorp.FieldLayout
	for _, f := range fields {
		layout, err := buildFieldLayout(f, table, visiting)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		layouts = append(layouts, layout)
	}
	return layouts, nil
}

func buildFieldLayout(
	f resolution.Field,
	table *resolution.Table,
	visiting map[string]bool,
) (gorp.FieldLayout, error) {
	layout := gorp.FieldLayout{
		Name:         f.Name,
		Optional:     f.IsOptional,
		HardOptional: f.IsHardOptional,
	}

	resolved, ok := f.Type.Resolve(table)
	if !ok {
		if f.Type.IsTypeParam() {
			layout.Encoding = gorp.EncodingJSON
			return layout, nil
		}
		return layout, fmt.Errorf("cannot resolve type %s", f.Type.Name)
	}

	enc, nested, err := resolveEncoding(resolved, f.Type, table, visiting)
	if err != nil {
		return layout, err
	}
	layout.Encoding = enc
	if nested != nil {
		layout.Fields = nested.Fields
		layout.Element = nested.Element
		layout.Key = nested.Key
		layout.Value = nested.Value
	}
	return layout, nil
}

type nestedInfo struct {
	Fields  []gorp.FieldLayout
	Element *gorp.FieldLayout
	Key     *gorp.FieldLayout
	Value   *gorp.FieldLayout
}

func resolveEncoding(
	resolved resolution.Type,
	ref resolution.TypeRef,
	table *resolution.Table,
	visiting map[string]bool,
) (gorp.Encoding, *nestedInfo, error) {
	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		return primitiveEncoding(form.Name)

	case resolution.EnumForm:
		if form.IsIntEnum {
			return gorp.EncodingInt64, nil, nil
		}
		return gorp.EncodingString, nil, nil

	case resolution.AliasForm:
		inner, ok := form.Target.Resolve(table)
		if !ok {
			return gorp.EncodingJSON, nil, nil
		}
		return resolveEncoding(inner, form.Target, table, visiting)

	case resolution.DistinctForm:
		inner, ok := form.Base.Resolve(table)
		if !ok {
			return gorp.EncodingJSON, nil, nil
		}
		return resolveEncoding(inner, form.Base, table, visiting)

	case resolution.StructForm:
		qname := resolved.QualifiedName
		if visiting[qname] {
			// Recursive type: treated as length-prefixed struct. The resolver
			// handles recursion via the length prefix.
			return gorp.EncodingStruct, &nestedInfo{}, nil
		}
		visiting[qname] = true
		defer delete(visiting, qname)

		innerFields := resolution.UnifiedFields(resolved, table)
		layouts, err := buildFieldLayouts(innerFields, table, visiting)
		if err != nil {
			return 0, nil, err
		}
		return gorp.EncodingStruct, &nestedInfo{Fields: layouts}, nil

	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(ref.TypeArgs) > 0 {
			elemType, ok := ref.TypeArgs[0].Resolve(table)
			if !ok {
				return gorp.EncodingJSON, nil, nil
			}
			elemEnc, elemNested, err := resolveEncoding(elemType, ref.TypeArgs[0], table, visiting)
			if err != nil {
				return 0, nil, err
			}
			elem := &gorp.FieldLayout{Encoding: elemEnc}
			if elemNested != nil {
				elem.Fields = elemNested.Fields
				elem.Element = elemNested.Element
			}
			return gorp.EncodingArray, &nestedInfo{Element: elem}, nil
		}
		if form.Name == "Map" && len(ref.TypeArgs) >= 2 {
			keyType, ok := ref.TypeArgs[0].Resolve(table)
			if !ok {
				return gorp.EncodingJSON, nil, nil
			}
			valType, ok := ref.TypeArgs[1].Resolve(table)
			if !ok {
				return gorp.EncodingJSON, nil, nil
			}
			keyEnc, _, err := resolveEncoding(keyType, ref.TypeArgs[0], table, visiting)
			if err != nil {
				return 0, nil, err
			}
			valEnc, valNested, err := resolveEncoding(valType, ref.TypeArgs[1], table, visiting)
			if err != nil {
				return 0, nil, err
			}
			key := &gorp.FieldLayout{Encoding: keyEnc}
			val := &gorp.FieldLayout{Encoding: valEnc}
			if valNested != nil {
				val.Fields = valNested.Fields
			}
			return gorp.EncodingMap, &nestedInfo{Key: key, Value: val}, nil
		}
		return gorp.EncodingJSON, nil, nil

	default:
		return gorp.EncodingJSON, nil, nil
	}
}

func primitiveEncoding(name string) (gorp.Encoding, *nestedInfo, error) {
	switch strings.ToLower(name) {
	case "bool":
		return gorp.EncodingBool, nil, nil
	case "int8":
		return gorp.EncodingInt8, nil, nil
	case "int16":
		return gorp.EncodingInt16, nil, nil
	case "int32":
		return gorp.EncodingInt32, nil, nil
	case "int64":
		return gorp.EncodingInt64, nil, nil
	case "uint8":
		return gorp.EncodingUint8, nil, nil
	case "uint12", "uint16":
		return gorp.EncodingUint16, nil, nil
	case "uint20", "uint32":
		return gorp.EncodingUint32, nil, nil
	case "uint64":
		return gorp.EncodingUint64, nil, nil
	case "float32":
		return gorp.EncodingFloat32, nil, nil
	case "float64":
		return gorp.EncodingFloat64, nil, nil
	case "uuid":
		return gorp.EncodingUUID, nil, nil
	case "string":
		return gorp.EncodingString, nil, nil
	case "bytes":
		return gorp.EncodingBytes, nil, nil
	case "record", "any":
		return gorp.EncodingJSON, nil, nil
	default:
		return gorp.EncodingJSON, nil, fmt.Errorf("unknown primitive: %s", name)
	}
}
