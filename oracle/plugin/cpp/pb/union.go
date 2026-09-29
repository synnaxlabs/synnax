// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	"github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// unionTranslatorData drives generation of the free to_proto(const <Name>&) and
// <name>_from_proto functions that translate a discriminated union between its
// std::variant form and its protobuf oneof wrapper message.
type unionTranslatorData struct {
	// CppName is the std::variant alias (e.g. "Entry").
	CppName string
	// SnakeName prefixes the free from_proto function (e.g. entry_from_proto).
	SnakeName string
	// PBType is the qualified wrapper message (e.g. "::service::library::pb::Entry").
	PBType string
	// Bases lists the union's extends bases, which nest as message fields on the
	// wrapper and translate through the bases' own translators.
	Bases []unionBaseTranslatorData
	// Variants lists every variant in declaration order.
	Variants []unionVariantTranslatorData
}

// unionBaseTranslatorData holds data for one extends base of a union translator.
type unionBaseTranslatorData struct {
	// CppType is the base struct every variant inherits (e.g. "BaseEntry").
	CppType string
	// PBAccessor is the wrapper's accessor for the base message (e.g. "base_entry").
	PBAccessor string
}

// unionVariantTranslatorData holds data for one variant of a union translator.
type unionVariantTranslatorData struct {
	// CppType is the variant struct (e.g. "MessageEntry").
	CppType string
	// PBAccessor is the oneof member accessor (e.g. "message", or "enum_").
	PBAccessor string
	// CaseName is the protoc oneof case constant (e.g. "kMessage").
	CaseName string
	// PayloadPBType is the qualified payload message of the oneof member.
	PayloadPBType string
	// IsInline marks an inline variant, whose payload fields the variant struct
	// declares directly and translate through generated helper functions.
	IsInline bool
	// SnakeName prefixes an inline variant's helper functions.
	SnakeName string
	// Fields are an inline variant's payload field conversions, written against a
	// variable named cpp.
	Fields []fieldTranslatorData
	// PayloadCppType is the payload struct a non-inline variant inherits, which
	// translates through its own to_proto and from_proto.
	PayloadCppType string
}

// processUnionForTranslation builds the translator view for a discriminated union.
func (p *Plugin) processUnionForTranslation(
	u resolution.Type,
	data *templateData,
) (*unionTranslatorData, error) {
	form := u.Form.(resolution.UnionForm)
	cppName := domain.GetName(u, "cpp")
	pbNamespace := naming.PBNamespace(output.GetPBPath(u))
	ut := &unionTranslatorData{
		CppName:   cppName,
		SnakeName: casing.FieldSnake(cppName),
		PBType:    fmt.Sprintf("%s::%s", pbNamespace, naming.PBName(u)),
	}
	for _, ext := range form.Extends {
		base, ok := ext.Resolve(data.table)
		if !ok {
			return nil, errors.Newf("union %s: unresolved base %s", u.Name, ext.Name)
		}
		if _, isStruct := base.Form.(resolution.StructForm); !isStruct {
			continue
		}
		ut.Bases = append(ut.Bases, unionBaseTranslatorData{
			CppType:    p.resolveExtendsType(ext, base, data),
			PBAccessor: keywords.Escape(casing.FieldSnake(base.Name)),
		})
	}
	for _, v := range form.Variants {
		payload, ok := v.Type.Resolve(data.table)
		if !ok {
			return nil, errors.Newf(
				"union %s variant %q: unresolved payload type",
				u.Name,
				v.Name,
			)
		}
		payloadPBPath := output.GetPBPath(payload)
		if payloadPBPath == "" {
			payloadPBPath = output.GetPBPath(u)
		}
		variantName := naming.VariantTypeName(cppName, v.Name)
		pbField := casing.FieldSnake(v.Name)
		vt := unionVariantTranslatorData{
			CppType:    variantName,
			PBAccessor: keywords.Escape(pbField),
			CaseName:   "k" + protocCamelCase(pbField),
			PayloadPBType: fmt.Sprintf(
				"%s::%s",
				naming.PBNamespace(payloadPBPath),
				naming.PBName(payload),
			),
			IsInline:  v.Inline,
			SnakeName: casing.FieldSnake(variantName),
		}
		if v.Inline {
			for _, f := range resolution.UnifiedFields(payload, data.table) {
				vt.Fields = append(
					vt.Fields,
					p.processFieldForTranslation(f, "cpp.", data),
				)
			}
		} else {
			vt.PayloadCppType = p.resolveExtendsType(v.Type, payload, data)
		}
		ut.Variants = append(ut.Variants, vt)
	}
	return ut, nil
}

// protocCamelCase mirrors protoc's UnderscoresToCamelCase, which names oneof case
// constants: a letter capitalizes at the start, after an underscore, or after a digit.
func protocCamelCase(name string) string {
	var b strings.Builder
	capNext := true
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			if capNext {
				r -= 'a' - 'A'
			}
			b.WriteRune(r)
			capNext = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			capNext = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			capNext = true
		default:
			capNext = true
		}
	}
	return b.String()
}

// unionTranslators returns the qualified names of a union's free to_proto and
// <name>_from_proto functions, adding the includes a cross-namespace reference needs.
// Calls are always qualified: unqualified lookup inside a member to_proto finds the
// member and never reaches the free overload.
func (p *Plugin) unionTranslators(
	resolved resolution.Type,
	data *templateData,
) (toProto, fromProto string) {
	qualifier := "::" + data.Namespace
	if resolved.Namespace != data.rawNs {
		if targetOutputPath := output.GetPath(resolved, "cpp"); targetOutputPath != "" {
			data.addConversionIncludes(targetOutputPath)
			qualifier = "::" + naming.Namespace(targetOutputPath)
		}
	}
	return qualifier + "::to_proto", fmt.Sprintf(
		"%s::%s_from_proto",
		qualifier,
		casing.FieldSnake(domain.GetName(resolved, "cpp")),
	)
}

// generateUnionConversion renders the conversion of a union-typed field.
func (p *Plugin) generateUnionConversion(
	resolved resolution.Type,
	isOptional bool,
	data *templateData,
	recv, cppFieldName, pbAccessorName string,
) (forward, backward string) {
	toProto, fromProto := p.unionTranslators(resolved, data)
	if isOptional {
		forward = fmt.Sprintf(`if (%s%s.has_value()) {
        auto [v, err] = %s(*%s%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, recv, cppFieldName, toProto, recv, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = %s(pb.%s());
        if (err) return {{}, err};
        cpp.%s = std::move(v);
    }`, pbAccessorName, fromProto, pbAccessorName, cppFieldName)
		return forward, backward
	}
	forward = fmt.Sprintf(`{
        auto [v, err] = %s(%s%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, toProto, recv, cppFieldName, pbAccessorName)
	backward = fmt.Sprintf(`{
        auto [v, err] = %s(pb.%s());
        if (err) return {{}, err};
        cpp.%s = std::move(v);
    }`, fromProto, pbAccessorName, cppFieldName)
	return forward, backward
}

// generateUnionArrayConversion renders the conversion of an array of unions.
func (p *Plugin) generateUnionArrayConversion(
	resolved resolution.Type,
	isOptional bool,
	data *templateData,
	recv, cppFieldName, pbAccessorName string,
) (forward, backward string) {
	toProto, fromProto := p.unionTranslators(resolved, data)
	if isOptional {
		forward = fmt.Sprintf(`if (%s%s.has_value()) {
        auto* wrapper = pb.mutable_%s();
        for (const auto& item : *%s%s) {
            auto [v, err] = %s(item);
            if (err) return {{}, err};
            *wrapper->add_values() = v;
        }
    }`, recv, cppFieldName, pbAccessorName, recv, cppFieldName, toProto)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s.emplace();
        for (const auto& item : pb.%s().values()) {
            auto [v, err] = %s(item);
            if (err) return {{}, err};
            cpp.%s->push_back(std::move(v));
        }
    }`, pbAccessorName, cppFieldName, pbAccessorName, fromProto, cppFieldName)
		return forward, backward
	}
	forward = fmt.Sprintf(`for (const auto& item : %s%s) {
        auto [v, err] = %s(item);
        if (err) return {{}, err};
        *pb.add_%s() = v;
    }`, recv, cppFieldName, toProto, pbAccessorName)
	backward = fmt.Sprintf(`for (const auto& item : pb.%s()) {
        auto [v, err] = %s(item);
        if (err) return {{}, err};
        cpp.%s.push_back(std::move(v));
    }`, pbAccessorName, fromProto, cppFieldName)
	return forward, backward
}
