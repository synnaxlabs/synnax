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
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// hasDictForm reports whether a struct declares a dict form with @py dict.
func hasDictForm(typ resolution.Type) bool {
	return domain.HasExprFromType(typ, "py", "dict")
}

// addDictForm declares the dict form of a struct with @py dict: a TypedDict whose
// optional fields are keys that can be absent, and whose fields of a struct with a dict
// form take that dict form.
func addDictForm(
	typ resolution.Type,
	sd *structData,
	table *resolution.Table,
	data *templateData,
) error {
	if sd.Skip || sd.IsAlias || !hasDictForm(typ) {
		return nil
	}
	form := typ.Form.(resolution.StructForm)
	if sd.After != "" || len(form.Extends) > 0 {
		return errors.Newf(
			"%s: @py dict needs a struct with no parent and no filter input",
			typ.QualifiedName,
		)
	}
	lines := make([]string, 0, len(form.Fields))
	for _, f := range form.Fields {
		key := casing.FieldSnake(f.Name)
		if fieldName(f.Name) != key {
			return errors.Newf(
				"%s: @py dict cannot declare the keyword field %s",
				typ.QualifiedName,
				key,
			)
		}
		value := dictType(f.Type, table, data)
		if f.Optional {
			data.imports.addTyping("NotRequired")
			value = "NotRequired[" + value + "]"
		}
		lines = append(lines, "    "+key+": "+value)
	}
	data.imports.addTyping("TypedDict")
	generic := ""
	if len(sd.TypeParams) > 0 {
		params := lo.Map(sd.TypeParams, func(p typeParamData, _ int) string {
			return p.Name
		})
		generic = ", Generic[" + strings.Join(params, ", ") + "]"
	}
	name := sd.PyName + "Dict"
	sd.After = fmt.Sprintf(
		"# %s is the dict form of %s.\nclass %s(TypedDict%s):\n%s",
		name,
		sd.PyName,
		name,
		generic,
		strings.Join(lines, "\n"),
	)
	return nil
}

// dictType returns the Python type of a field of a dict form, where a struct of the
// same
// namespace with a dict form takes it.
func dictType(
	ref resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if isList(ref) {
		return "list[" + dictType(ref.TypeArgs[0], table, data) + "]"
	}
	resolved, ok := ref.Resolve(table)
	if ref.IsTypeParam() || !ok || resolved.Namespace != data.Namespace ||
		!hasDictForm(resolved) {
		return typeToPython(ref, table, data)
	}
	name := getPyName(resolved) + "Dict"
	if len(ref.TypeArgs) == 0 {
		return name
	}
	args := lo.Map(ref.TypeArgs, func(a resolution.TypeRef, _ int) string {
		return dictType(a, table, data)
	})
	return name + "[" + strings.Join(args, ", ") + "]"
}
