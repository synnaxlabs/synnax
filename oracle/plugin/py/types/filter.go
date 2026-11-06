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
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// readers maps each operator type to the function of the hand-written
// x.filter.shorthand module that reads it from a plain value or a list.
var readers = map[string]string{
	resolution.OperatorSet:      "read_set",
	resolution.OperatorOrdered:  "read_set",
	resolution.OperatorString:   "read_set",
	resolution.OperatorBool:     "read_bool",
	resolution.OperatorContains: "read_contains",
}

// addFilterInput lets an operator or filter model read a filter written as a partial
// record, and declares the TypedDict and input types that type one.
func addFilterInput(
	typ resolution.Type,
	sd *structData,
	table *resolution.Table,
	data *templateData,
) error {
	if sd.Skip || sd.IsAlias {
		return nil
	}
	if typ.Namespace == resolution.FilterNamespace {
		if _, ok := resolution.OperatorMembers(typ.QualifiedName); ok {
			return addOperatorInput(typ, sd, table, data)
		}
	}
	if typ.Filter != nil {
		return addWhereInput(typ, sd, table, data)
	}
	return nil
}

// addOperatorInput reads the operator from a plain value or a list, and declares its
// TypedDict and the input type of a line that holds it.
func addOperatorInput(
	typ resolution.Type,
	sd *structData,
	table *resolution.Table,
	data *templateData,
) error {
	reader, ok := readers[typ.QualifiedName]
	if !ok {
		return errors.Newf("py/types has no shorthand reader for %s", typ.QualifiedName)
	}
	data.imports.addTyping("Any")
	data.imports.addTyping("TypedDict")
	data.imports.addPydantic("model_validator")
	sd.Validators = fmt.Sprintf(`    @model_validator(mode="before")
    @classmethod
    def _read_shorthand(cls, value: Any) -> Any:
        return %s.%s(value, cls)`, importShorthand(typ, data), reader)

	params := lo.Map(sd.TypeParams, func(p typeParamData, _ int) string {
		return p.Name
	})
	args := ""
	value := strings.Join(params, ", ")
	if len(params) > 0 {
		args = "[" + value + "]"
	} else {
		eq, ok := typ.Form.(resolution.StructForm).Field("eq")
		if !ok {
			return errors.Newf("operator %s has no eq member", typ.QualifiedName)
		}
		value = typeToPython(eq.Type, table, data)
	}
	dict := sd.PyName + "Dict"
	variants := []string{value}
	plain := "a plain value"
	if typ.QualifiedName != resolution.OperatorBool {
		variants = append(variants, "list["+value+"]")
		plain = "a plain value or list"
	}
	variants = append(variants, dict+args, sd.PyName+args)
	sd.After = fmt.Sprintf(`# %s is the dict form of %s.
%s

# %sInput is %s, %s, or %s in its place.
type %sInput%s = %s`,
		dict, sd.PyName, typedDict(dict, sd.Fields, params),
		sd.PyName, sd.PyName, dict, plain,
		sd.PyName, args, strings.Join(variants, " | "),
	)
	return nil
}

// addWhereInput reads a typed list line from one value, and declares the TypedDict of
// a filter written as a partial record.
func addWhereInput(
	typ resolution.Type,
	sd *structData,
	table *resolution.Table,
	data *templateData,
) error {
	form := typ.Form.(resolution.StructForm)
	input := sd.PyName + "Input"
	keys := []string{
		fmt.Sprintf(`"and": list["%s"]`, input),
		fmt.Sprintf(`"or": list["%s"]`, input),
		fmt.Sprintf(`"not": "%s"`, input),
	}
	var lists []string
	for _, line := range typ.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		lineType, err := lineInput(line, field, table, data)
		if err != nil {
			return errors.Wrapf(err, "filter %s", typ.QualifiedName)
		}
		if line.HasType() && isList(field.Type) {
			lists = append(lists, fmt.Sprintf("%q", fieldName(line.Name)))
		}
		key := casing.FieldSnake(line.Name)
		keys = append(keys, fmt.Sprintf("%q: %s", key, lineType))
	}
	data.imports.addTyping("TypedDict")
	if len(lists) > 0 {
		set, ok := table.Get(resolution.OperatorSet)
		if !ok {
			return errors.Newf("filter %s: no Set operator type", typ.QualifiedName)
		}
		data.imports.addTyping("Any")
		data.imports.addPydantic("field_validator")
		sd.Validators = fmt.Sprintf(`    @field_validator(%s, mode="before")
    @classmethod
    def _read_lists(cls, value: Any) -> Any:
        return %s.read_list(value)`,
			strings.Join(lists, ", "),
			importShorthand(set, data),
		)
	}
	sd.After = fmt.Sprintf(`# %s is a %s written as a partial record.
# A plain value or a list stands for an operator.
%s = TypedDict(
    "%s",
    {%s},
    total=False,
)`, input, sd.PyName, input, input, strings.Join(keys, ", "))
	return nil
}

// lineInput returns the Python type that a line of a partial record takes: the input
// of a bare line's operator, or a typed line's type, which a list line also takes as
// one element.
func lineInput(
	line resolution.Field,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) (string, error) {
	if line.HasType() {
		if isList(field.Type) {
			elem := typeToPython(field.Type.TypeArgs[0], table, data)
			return elem + " | list[" + elem + "]", nil
		}
		return typeToPython(field.Type, table, data), nil
	}
	op, ok := field.Type.Resolve(table)
	if !ok {
		return "", errors.Newf(
			"line %s has the unknown operator %s",
			line.Name,
			field.Type.Name,
		)
	}
	name := getPyName(op) + "Input"
	if op.Namespace != data.Namespace {
		modulePath := toPythonModulePath(output.GetPath(op, "py"))
		name = addCrossNamespaceImport(modulePath, name, data)
	}
	return name + typeArgsToPython(op, field.Type.TypeArgs, table, data), nil
}

func isList(ref resolution.TypeRef) bool {
	return ref.Name == "Array" && ref.ArraySize == nil
}

// importShorthand imports the hand-written shorthand module beside the operator op.
func importShorthand(op resolution.Type, data *templateData) string {
	return data.imports.addModuleImport(
		toPythonModulePath(output.GetPath(op, "py")),
		"shorthand",
	)
}

// typedDict declares a TypedDict whose keys are the wire names of fields. It uses class
// syntax unless a key is a Python keyword, such as "in", which only call syntax takes.
func typedDict(name string, fields []fieldData, params []string) string {
	valueType := func(f fieldData) string {
		return strings.TrimSuffix(f.PyType, " | None")
	}
	if !lo.SomeBy(fields, func(f fieldData) bool { return f.Alias != "" }) {
		generic := ""
		if len(params) > 0 {
			generic = ", Generic[" + strings.Join(params, ", ") + "]"
		}
		lines := lo.Map(fields, func(f fieldData, _ int) string {
			return "    " + f.Name + ": " + valueType(f)
		})
		return fmt.Sprintf(
			"class %s(TypedDict%s, total=False):\n%s",
			name,
			generic,
			strings.Join(lines, "\n"),
		)
	}
	keys := lo.Map(fields, func(f fieldData, _ int) string {
		key := lo.Ternary(f.Alias != "", f.Alias, f.Name)
		return fmt.Sprintf("%q: %s", key, valueType(f))
	})
	// mypy checks the uses of a generic TypedDict in call syntax, but rejects its type
	// variables at the declaration.
	ignore := lo.Ternary(len(params) > 0, "  # type: ignore[valid-type]", "")
	return fmt.Sprintf(
		"%s = TypedDict(\n    %q,\n    {%s},%s\n    total=False,\n)",
		name,
		name,
		strings.Join(keys, ", "),
		ignore,
	)
}
