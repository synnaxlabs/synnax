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
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/py/keywords"
	pyprimitives "github.com/synnaxlabs/oracle/plugin/py/primitives"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// primitiveMapper is the Python-specific primitive type mapper.
var primitiveMapper = pyprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	OutputPath      string
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types_gen.py",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "py/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	gen := &framework.Generator{
		Domain:          "py",
		FilePattern:     p.Options.FileNamePattern,
		FileGenerator:   &pyFileGenerator{},
		MergeByName:     true,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	return gen.Generate(req)
}

// pyFileGenerator implements framework.FileGenerator for Python code generation.
type pyFileGenerator struct{}

func (g *pyFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	content, err := generatePyFile(
		ctx.Namespace,
		ctx.OutputPath,
		ctx.Structs,
		ctx.Enums,
		ctx.TypeDefs,
		ctx.Unions,
		ctx.Table,
	)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func generatePyFile(
	namespace string,
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	table *resolution.Table,
) ([]byte, error) {
	data := &templateData{
		Namespace:   namespace,
		OutputPath:  outputPath,
		Structs:     make([]structData, 0, len(structs)),
		Enums:       make([]enumData, 0, len(enums)),
		TypeDefs:    make([]typeDefData, 0, len(typeDefs)),
		SortedDecls: make([]sortedDeclData, 0),
		imports:     newImportManager(),
	}
	if len(structs) > 0 {
		data.imports.addPydantic("BaseModel")
	}

	skip := func(typ resolution.Type) bool { return omit.IsSkipped(typ, "py") }
	rawKeyFields := key.Collect(structs, table, skip)
	keyFields := convertKeyFields(rawKeyFields, data)
	allStructs := lo.Filter(table.StructTypes(), func(typ resolution.Type, _ int) bool {
		return output.GetPath(typ, "py") == outputPath
	})
	allRawKeyFields := key.Collect(allStructs, table, nil)
	allKeyFields := convertKeyFields(allRawKeyFields, data)
	data.Ontology = extractOntology(allStructs, allRawKeyFields, allKeyFields, nil)
	if data.Ontology != nil {
		data.imports.addOntology("ID")
	}

	// Pre-collect all field names to detect conflicts with module import names.
	// This allows us to alias module imports when a field name would shadow them.
	for _, s := range structs {
		if form, ok := s.Form.(resolution.StructForm); ok {
			for _, field := range form.Fields {
				data.imports.registerFieldName(field.Name)
			}
		}
		// Also collect unified fields (which include inherited fields)
		for _, field := range resolution.UnifiedFields(s, table) {
			data.imports.registerFieldName(field.Name)
		}
	}

	// A TypeAlias assignment evaluates its right-hand side eagerly, so an alias
	// whose target names a type defined in the SAME file (e.g. Nodes = list[Node])
	// must be emitted after that type or list[Node] raises NameError at import.
	// Those are collected into trailingTypeDefs and emitted after every struct and
	// union. Aliases that only reference primitives or imported (cross-namespace)
	// types have no same-file forward-reference hazard and keep their existing
	// position: distinct primitive aliases (e.g. Key = uuid) up-front, other
	// aliases (e.g. Status = status.Status[...]) topologically sorted with structs.
	var aliasTypeDefs []resolution.Type
	var trailingTypeDefs []resolution.Type
	for _, td := range typeDefs {
		_, isAlias := td.Form.(resolution.AliasForm)
		switch {
		case typeDefRefsSameNamespaceType(td, table):
			trailingTypeDefs = append(trailingTypeDefs, td)
		case isAlias:
			aliasTypeDefs = append(aliasTypeDefs, td)
		default:
			data.TypeDefs = append(data.TypeDefs, processTypeDef(td, table, data))
		}
	}

	for _, e := range enums {
		data.Enums = append(data.Enums, processEnum(e, data))
	}

	// Hazard-free aliases sort up-front with structs and unions, preserving their
	// existing positions. Same-file-referencing aliases are appended last so that
	// when a reference cycle forces the topological sort to fall back to input
	// order, those eagerly-evaluated list[...] / dict[...] assignments land after
	// the classes they name. For acyclic aliases the sort still positions them by
	// dependency (e.g. Functions before IR), so the trailing input order only
	// matters inside a cycle. Any forward reference that remains is resolved by the
	// model_rebuild() calls emitted at the end of the file.
	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, aliasTypeDefs...)
	combinedTypes = append(combinedTypes, structs...)
	combinedTypes = append(combinedTypes, unions...)
	combinedTypes = append(combinedTypes, trailingTypeDefs...)

	sortedTypes := table.TopologicalSort(combinedTypes)

	typePos := make(map[string]int, len(sortedTypes))
	for i, typ := range sortedTypes {
		typePos[typ.QualifiedName] = i
	}

	for i, typ := range sortedTypes {
		switch typ.Form.(type) {
		case resolution.AliasForm, resolution.DistinctForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsTypeDef: true,
				TypeDef:   processTypeDef(typ, table, data),
			})
		case resolution.StructForm:
			sd := processStruct(typ, table, data, keyFields)
			for _, f := range sd.Fields {
				if f.Alias != "" {
					sd.HasKeywordAlias = true
					data.imports.addPydantic("ConfigDict")
					break
				}
			}
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsStruct: true,
				Struct:   sd,
			})
			// A struct that names a same-file type defined at or after its own
			// position (a self-reference or a cycle) leaves a forward reference
			// pydantic cannot resolve until the file finishes loading, so emit a
			// trailing model_rebuild() for it.
			if structHasForwardRef(typ, i, typePos, table) {
				data.RebuildModels = append(data.RebuildModels, sd.PyName)
			}
		case resolution.UnionForm:
			u := processUnion(typ, table, data, keyFields)
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsUnion: true,
				Union:   u,
			})
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func convertKeyFields(fields []key.Field, data *templateData) []keyFieldData {
	result := make([]keyFieldData, 0, len(fields))
	for _, f := range fields {
		result = append(result, keyFieldData{
			Name:   f.Name,
			PyType: primitiveToPython(f.Primitive, data),
		})
	}
	return result
}

func extractOntology(
	types []resolution.Type,
	rawFields []key.Field,
	keyFields []keyFieldData,
	skip ontology.SkipFunc,
) *ontologyData {
	data := ontology.Extract(types, rawFields, skip)
	if data == nil || len(keyFields) == 0 {
		return nil
	}
	return &ontologyData{
		TypeName:   data.TypeName,
		KeyType:    "Key",
		StructName: data.StructName,
	}
}

func processEnum(typ resolution.Type, data *templateData) enumData {
	form, ok := typ.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: typ.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	var literalValues []string
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue(),
			IntValue:  v.IntValue(),
			IsIntEnum: form.IsIntEnum,
		})
		if !form.IsIntEnum {
			literalValues = append(literalValues, fmt.Sprintf("%q", v.StringValue()))
		}
	}
	if form.IsIntEnum {
		data.imports.addEnum("IntEnum")
	} else {
		data.imports.addTyping("Literal")
	}
	return enumData{
		Name:          typ.Name,
		Values:        values,
		IsIntEnum:     form.IsIntEnum,
		LiteralValues: strings.Join(literalValues, ", "),
	}
}

// typeDefRefsSameNamespaceType reports whether a type alias's target references a
// type declared in the same namespace (and therefore the same generated file).
// Array and map wrappers are followed to their element/value types. Only these
// aliases carry a same-file forward-reference hazard: an eagerly-evaluated
// list[...] / dict[...] assignment names a class that may be defined later in the
// file. References to primitives or to imported (cross-namespace) types are safe.
func typeDefRefsSameNamespaceType(td resolution.Type, table *resolution.Table) bool {
	var refs func(ref resolution.TypeRef) bool
	refs = func(ref resolution.TypeRef) bool {
		if ref.Name == "Array" || ref.Name == "Map" {
			return slices.ContainsFunc(ref.TypeArgs, refs)
		}
		if ref.Name == "" || ref.IsTypeParam() || resolution.IsPrimitive(ref.Name) {
			return false
		}
		resolved, ok := table.Get(ref.Name)
		if !ok {
			resolved, ok = table.Lookup(td.Namespace, ref.Name)
		}
		return ok && resolved.Namespace == td.Namespace
	}
	switch form := td.Form.(type) {
	case resolution.AliasForm:
		return refs(form.Target)
	case resolution.DistinctForm:
		return refs(form.Base)
	default:
		return false
	}
}

// structHasForwardRef reports whether a struct at position selfIdx references a
// type in the same namespace that is emitted at or after selfIdx (a self-reference
// or a member of a reference cycle). With deferred annotations such a reference is
// unresolved at class-definition time; pydantic resolves it once the file finishes
// loading, which requires a model_rebuild() call. References to types defined
// earlier in the file or in imported (cross-namespace) modules are already
// resolvable and need no rebuild.
func structHasForwardRef(
	typ resolution.Type,
	selfIdx int,
	typePos map[string]int,
	table *resolution.Table,
) bool {
	var refs func(ref resolution.TypeRef) bool
	refs = func(ref resolution.TypeRef) bool {
		if ref.Name == "Array" || ref.Name == "Map" {
			return slices.ContainsFunc(ref.TypeArgs, refs)
		}
		if ref.Name == "" || ref.IsTypeParam() || resolution.IsPrimitive(ref.Name) {
			return false
		}
		resolved, ok := table.Get(ref.Name)
		if !ok {
			resolved, ok = table.Lookup(typ.Namespace, ref.Name)
		}
		if !ok || resolved.Namespace != typ.Namespace {
			return false
		}
		if pos, ok := typePos[resolved.QualifiedName]; ok && pos >= selfIdx {
			return true
		}
		// Follow a same-namespace alias target (e.g. inputs Params -> list[Param])
		// to the type it ultimately names.
		switch f := resolved.Form.(type) {
		case resolution.AliasForm:
			return refs(f.Target)
		case resolution.DistinctForm:
			return refs(f.Base)
		}
		return false
	}
	for _, field := range resolution.UnifiedFields(typ, table) {
		if refs(field.Type) {
			return true
		}
	}
	return false
}

func processTypeDef(
	typ resolution.Type,
	table *resolution.Table,
	data *templateData,
) typeDefData {
	switch form := typ.Form.(type) {
	case resolution.DistinctForm:
		data.imports.addTyping("TypeAlias")
		return typeDefData{
			Name:       typ.Name,
			BaseType:   typeDefBaseToPython(form.Base, typ.Namespace, table, data),
			IsDistinct: false,
		}
	case resolution.AliasForm:
		// For alias types, use typeRefToPythonAlias to properly handle struct references
		// with type arguments (e.g., status.Status<StatusDetails> -> status.Status[StatusDetails])
		data.imports.addTyping("TypeAlias")
		return typeDefData{
			Name:       typ.Name,
			BaseType:   typeRefToPythonAlias(form.Target, table, data),
			IsDistinct: false,
		}
	default:
		data.imports.addTyping("TypeAlias")
		return typeDefData{Name: typ.Name, BaseType: "Any", IsDistinct: false}
	}
}

func processTypeParam(
	tp resolution.TypeParam,
	table *resolution.Table,
	data *templateData,
) typeParamData {
	tpd := typeParamData{
		Name:       tp.Name,
		IsOptional: tp.Optional,
	}
	if tp.Constraint != nil {
		tpd.Constraint = constraintToPython(*tp.Constraint, table, data)
	}
	return tpd
}

func constraintToPython(
	constraint resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if resolution.IsConstraint(constraint.Name) {
		return ""
	}
	if resolution.IsPrimitive(constraint.Name) {
		return primitiveToPython(constraint.Name, data)
	}
	resolved, ok := constraint.Resolve(table)
	if !ok {
		data.imports.addTyping("Any")
		return "Any"
	}
	// Handle enums
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		if resolved.Namespace != data.Namespace {
			outputPath := enum.FindOutputPath(resolved, table, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name
	}
	// Handle structs
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name
	}
	data.imports.addTyping("Any")
	return "Any"
}

func containsTypeVar(typeVars []typeVarData, name string) bool {
	for _, tv := range typeVars {
		if tv.Name == name {
			return true
		}
	}
	return false
}

func typeDefBaseToPython(
	typeRef resolution.TypeRef,
	currentNamespace string,
	table *resolution.Table,
	data *templateData,
) string {
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToPython(typeRef.Name, data)
	}
	// Handle Array type. Collection elements are resolved via typeToPython, which
	// handles every form (struct, enum, alias, union); typeDefBaseToPython only
	// special-cases distinct bases and would yield Any for a struct element.
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemType := typeToPython(typeRef.TypeArgs[0], table, data)
		// Fixed-size array -> tuple type
		if typeRef.ArraySize != nil {
			data.imports.addTyping("Tuple")
			elements := make([]string, *typeRef.ArraySize)
			for i := range elements {
				elements[i] = elemType
			}
			return fmt.Sprintf("Tuple[%s]", strings.Join(elements, ", "))
		}
		return fmt.Sprintf("list[%s]", elemType)
	}
	// Handle Map type
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := typeToPython(typeRef.TypeArgs[0], table, data)
		valueType := typeToPython(typeRef.TypeArgs[1], table, data)
		return fmt.Sprintf("dict[%s, %s]", keyType, valueType)
	}
	// Try to resolve another typedef
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		data.imports.addTyping("Any")
		return "Any"
	}
	if _, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		if resolved.Namespace != currentNamespace {
			ns := resolved.Namespace
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = ns
			}
			modulePath := toPythonModulePath(outputPath)
			data.imports.addNamespace(ns, modulePath)
			return fmt.Sprintf("%s.%s", ns, resolved.Name)
		}
		return resolved.Name
	}
	data.imports.addTyping("Any")
	return "Any"
}

// hashableKey reports whether field is a @key field that can serve as the basis
// for __hash__. An optional key (e.g. on a creation payload, where the key is
// assigned by the server) is normally absent, so hashing by it is meaningless;
// only a required key yields a __hash__.
func hashableKey(field resolution.Field) bool {
	return key.HasKey(field) && !field.Optional
}

func processStruct(
	entry resolution.Type,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) structData {
	sd := structData{
		Name:   entry.Name,
		Doc:    doc.Get(entry.Domains),
		PyName: entry.Name, // Default to original name
	}

	form, ok := entry.Form.(resolution.StructForm)
	if !ok {
		sd.Skip = true
		return sd
	}

	// Check for py domain omit or hand-written implementation
	if domain.HasExprFromType(entry, "py", "omit") ||
		domain.HasExprFromType(entry, "py", "hand") {
		sd.Skip = true
		return sd
	}
	// Check for py domain name override
	if name := domain.GetStringFromType(entry, "py", "name"); name != "" {
		sd.PyName = name
	}

	// Process type parameters for generic structs
	if len(form.TypeParams) > 0 {
		sd.IsGeneric = true
		data.imports.addTyping("Generic")
		data.imports.addTyping("TypeVar")
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
			tpd := processTypeParam(tp, table, data)
			sd.TypeParams = append(sd.TypeParams, tpd)

			// Collect TypeVars for module-level declaration (with or without constraints)
			if !containsTypeVar(data.TypeVars, tp.Name) {
				data.TypeVars = append(data.TypeVars, typeVarData{
					Name:       tp.Name,
					Constraint: tpd.Constraint,
				})
			}
		}
	}

	// Handle struct aliases (AliasForm)
	if aliasForm, isAlias := entry.Form.(resolution.AliasForm); isAlias {
		sd.IsAlias = true
		sd.AliasOf = typeRefToPythonAlias(aliasForm.Target, table, data)
		return sd
	}

	// Handle struct extension with Pydantic inheritance (supports multiple parents)
	if len(form.Extends) > 0 {
		allParentsValid := true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(table)
			if !ok {
				allParentsValid = false
				break
			}
			extendsExpr := buildExtendsExpr(extendsRef, parent, table, data)
			sd.ExtendsNames = append(sd.ExtendsNames, extendsExpr)
		}

		if allParentsValid {
			// A domain removal (-@domain) cannot be expressed through class
			// inheritance — the base class still carries the domain — so emit a
			// flattened standalone class. Typeless overrides are already resolved
			// by the analyzer.
			if resolver.HasDomainOmissions(form) {
				sd.HasExtends = false
				sd.ExtendsNames = nil
				flat := resolution.UnifiedFields(entry, table)
				sd.Fields = make([]fieldData, 0, len(flat))
				for _, field := range flat {
					sd.Fields = append(
						sd.Fields,
						processField(field, table, data, keyFields, form.OmittedFields),
					)
					if hashableKey(field) {
						sd.KeyField = field.Name
					}
				}
				return sd
			}

			// Get all parent fields for comparison (first parent wins on conflict)
			parentFields := make([]resolution.Field, 0)
			seenFields := make(set.Set[string])
			for _, extendsRef := range form.Extends {
				parent, _ := extendsRef.Resolve(table)
				for _, pf := range resolution.UnifiedFields(parent, table) {
					if !seenFields.Contains(pf.Name) {
						seenFields.Add(pf.Name)
						parentFields = append(parentFields, pf)
					}
				}
			}

			// Check if any child field would create a type incompatibility
			// (overriding a required parent field as optional). If so, we
			// inline all fields instead of using inheritance to avoid
			// type: ignore comments.
			hasTypeConflict := false
			for _, field := range form.Fields {
				if field.Optional {
					for _, pf := range parentFields {
						if pf.Name == field.Name && !pf.Optional {
							hasTypeConflict = true
							break
						}
					}
				}
				if hasTypeConflict {
					break
				}
			}

			if hasTypeConflict {
				// Inline all fields from parents and child into a standalone class
				sd.HasExtends = false
				sd.ExtendsNames = nil
				childFieldsByName := make(map[string]resolution.Field)
				for _, f := range form.Fields {
					childFieldsByName[f.Name] = f
				}
				omittedSet := make(set.Set[string])
				for _, name := range form.OmittedFields {
					omittedSet.Add(name)
				}

				sd.Fields = make([]fieldData, 0, len(parentFields)+len(form.Fields))
				addedFields := make(set.Set[string])

				// Add parent fields, using child overrides where they exist
				for _, pf := range parentFields {
					if omittedSet.Contains(pf.Name) {
						continue
					}
					if childField, ok := childFieldsByName[pf.Name]; ok {
						fd := processField(
							childField,
							table,
							data,
							keyFields,
							form.OmittedFields,
						)
						sd.Fields = append(sd.Fields, fd)
						if hashableKey(childField) {
							sd.KeyField = childField.Name
						}
					} else {
						fd := processField(
							pf,
							table,
							data,
							keyFields,
							form.OmittedFields,
						)
						sd.Fields = append(sd.Fields, fd)
						if hashableKey(pf) {
							sd.KeyField = pf.Name
						}
					}
					addedFields.Add(pf.Name)
				}
				// Add child-only fields that aren't in the parent
				for _, field := range form.Fields {
					if addedFields.Contains(field.Name) {
						continue
					}
					fd := processField(
						field,
						table,
						data,
						keyFields,
						form.OmittedFields,
					)
					sd.Fields = append(sd.Fields, fd)
					if hashableKey(field) {
						sd.KeyField = field.Name
					}
				}
				return sd
			}

			sd.HasExtends = true
			// For extends, only include child's own fields (not inherited)
			// Pass OmittedFields so excluded fields get Field(exclude=True)
			sd.Fields = make([]fieldData, 0, len(form.Fields)+len(form.OmittedFields))
			redefinedFields := make(set.Set[string])
			for _, field := range form.Fields {
				redefinedFields.Add(field.Name)
				fd := processField(field, table, data, keyFields, form.OmittedFields)
				sd.Fields = append(sd.Fields, fd)
				// Only a required @key field yields a __hash__.
				if hashableKey(field) {
					sd.KeyField = field.Name
				}
			}
			// Also check parent fields for @key if not redefined
			for _, pf := range parentFields {
				if !redefinedFields.Contains(pf.Name) && hashableKey(pf) {
					sd.KeyField = pf.Name
				}
			}

			// For omitted fields that aren't redefined, we need to explicitly
			// add them with Field(exclude=True) to exclude from serialization
			for _, omittedName := range form.OmittedFields {
				if redefinedFields.Contains(omittedName) {
					continue // Already handled above
				}
				// Find the field type from parent
				for _, pf := range parentFields {
					if pf.Name == omittedName {
						data.imports.addPydantic("Field")
						// Run the inherited field through the normal field
						// pipeline so it keeps its real type, default, and
						// validation; processField adds exclude=True because the
						// name is in OmittedFields. A hand-built default here
						// would drop the parent's default and make an excluded
						// defaulted field wrongly required.
						fd := processField(
							pf,
							table,
							data,
							keyFields,
							form.OmittedFields,
						)
						sd.Fields = append(sd.Fields, fd)
						break
					}
				}
			}
			return sd
		}
	}

	// Non-extending struct: use all fields (flattened for compatibility)
	allFields := resolution.UnifiedFields(entry, table)
	sd.Fields = make([]fieldData, 0, len(allFields))
	for _, field := range allFields {
		sd.Fields = append(sd.Fields, processField(field, table, data, keyFields, nil))
		// Only a required @key field yields a __hash__.
		if hashableKey(field) {
			sd.KeyField = field.Name
		}
	}
	return sd
}

func getPyName(typ resolution.Type) string {
	return domain.GetName(typ, "py")
}

// buildExtendsExpr builds the Python extends expression for a parent class.
// If the parent has type params and type args are provided,
// it returns something like "Status[D, V]" instead of just "Status".
func buildExtendsExpr(
	extendsRef resolution.TypeRef,
	parent resolution.Type,
	table *resolution.Table,
	data *templateData,
) string {
	baseName := getPyName(parent)

	// A base class defined in another schema module must be imported and
	// referenced through its module alias (e.g. task_.BaseReadConfig).
	if parent.Namespace != data.Namespace {
		outputPath := output.GetPath(parent, "py")
		if outputPath == "" {
			outputPath = parent.Namespace
		}
		baseName = addCrossNamespaceImport(
			toPythonModulePath(outputPath),
			baseName,
			data,
		)
	}

	// Check if parent is generic
	parentForm, ok := parent.Form.(resolution.StructForm)
	if !ok || !parentForm.IsGeneric() {
		return baseName
	}

	// Find which parent type params don't have defaults (those are the ones in Generic[])
	var activeParamIndices []int
	for i, tp := range parentForm.TypeParams {
		if !tp.HasDefault() {
			activeParamIndices = append(activeParamIndices, i)
		}
	}

	if len(activeParamIndices) == 0 {
		return baseName
	}

	// Build type arguments for the active params
	var typeArgs []string
	for _, idx := range activeParamIndices {
		if idx < len(extendsRef.TypeArgs) {
			arg := extendsRef.TypeArgs[idx]
			typeArgs = append(typeArgs, typeToPython(arg, table, data))
		}
	}

	if len(typeArgs) == 0 {
		return baseName
	}

	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
}

func processField(
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
	excludedFields []string,
) fieldData {
	escapedName := keywords.Escape(field.Name)
	fd := fieldData{
		Name:       escapedName,
		Doc:        doc.Get(field.Domains),
		IsOptional: field.Optional,
		IsArray:    field.Type.Name == "Array",
	}
	if escapedName != field.Name {
		fd.Alias = field.Name
	}

	baseType := typeToPython(field.Type, table, data)
	validateDomain := field.Domains["validate"]
	fieldConstraints := collectValidation(
		validateDomain,
		field.Default,
		field.Type,
		table,
		data,
	)

	if bounds, ok := sizedIntBounds(field.Type, table); ok {
		hasGe := lo.ContainsBy(fieldConstraints, func(c string) bool {
			return strings.HasPrefix(c, "ge=")
		})
		hasLe := lo.ContainsBy(fieldConstraints, func(c string) bool {
			return strings.HasPrefix(c, "le=")
		})
		if !hasGe {
			fieldConstraints = append(fieldConstraints, "ge="+bounds.min)
		}
		if !hasLe {
			fieldConstraints = append(fieldConstraints, "le="+bounds.max)
		}
	}

	// Check if this field should be excluded from serialization (Pydantic v2)
	isExcluded := lo.Contains(excludedFields, field.Name)
	if isExcluded {
		fieldConstraints = append(fieldConstraints, "exclude=True")
	}

	if fd.IsArray {
		fd.PyType = fmt.Sprintf("list[%s]", baseType)
	} else {
		fd.PyType = baseType
	}

	// Optional (?) fields become T | None in Python (no pointer distinction). A field
	// typed as an optional type parameter (e.g. details Details, where Details?) is
	// likewise optional: an unspecialized struct may omit it. A required collection
	// defaults an absent value to its empty form, so it is always present and
	// iterable; the server omits a nil ("not loaded") collection from the wire, so
	// the default is what fills it. CollectionKind sees through aliases and
	// type-parameter constraints so an alias- or type-parameter-typed collection is
	// defaulted too.
	isOptionalTypeParam := field.Type.IsTypeParam() &&
		field.Type.TypeParam != nil && field.Type.TypeParam.Optional
	if field.Optional || isOptionalTypeParam {
		fd.PyType = fd.PyType + " | None"
	} else if kind, isCollection := resolution.CollectionKind(field.Type, table); isCollection {
		if kind == "Array" {
			fieldConstraints = ensureDefaultFactory(fieldConstraints, "list")
		} else {
			fieldConstraints = ensureDefaultFactory(fieldConstraints, "dict")
		}
	}

	fd.Default = buildDefault(
		field,
		fieldConstraints,
		fd.Alias,
		data,
		isOptionalTypeParam,
	)

	return fd
}

// ensureDefaultFactory appends a default_factory constraint unless one is already
// present, so a required collection with no declared default still defaults to empty
// when the field is absent.
func ensureDefaultFactory(constraints []string, factory string) []string {
	for _, c := range constraints {
		if strings.HasPrefix(c, "default_factory=") {
			return constraints
		}
	}
	return append(constraints, "default_factory="+factory)
}

func buildDefault(
	field resolution.Field,
	constraints []string,
	alias string,
	data *templateData,
	optionalTypeParam bool,
) string {
	isAnyOptional := field.Optional || optionalTypeParam

	// When field is optional, filter out any default= constraints from validation
	// since we'll be using default=None for the optional field
	if isAnyOptional {
		constraints = lo.Filter(constraints, func(c string, _ int) bool {
			return !strings.HasPrefix(c, "default=")
		})
	}

	if alias != "" {
		constraints = append(constraints, fmt.Sprintf("alias=\"%s\"", alias))
	}

	hasConstraints := len(constraints) > 0

	if isAnyOptional {
		if hasConstraints {
			data.imports.addPydantic("Field")
			return fmt.Sprintf(
				" = Field(default=None, %s)",
				strings.Join(constraints, ", "),
			)
		}
		return " = None"
	}

	if hasConstraints {
		// A lone immutable scalar default needs no Field() wrapper: emit a bare
		// default so `name: str = ""` reads more directly than Field(default="").
		// Mutable defaults use default_factory (never default=), so this is safe.
		if len(constraints) == 1 && strings.HasPrefix(constraints[0], "default=") {
			return " = " + strings.TrimPrefix(constraints[0], "default=")
		}
		data.imports.addPydantic("Field")
		return fmt.Sprintf(" = Field(%s)", strings.Join(constraints, ", "))
	}

	return ""
}

func collectValidation(
	domain resolution.Domain,
	defaultVal *resolution.ExpressionValue,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) []string {
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) && defaultVal == nil {
		return nil
	}
	var constraints []string
	isString := resolution.IsPrimitive(typeRef.Name) &&
		resolution.IsStringPrimitive(typeRef.Name)
	isNumber := resolution.IsPrimitive(typeRef.Name) &&
		resolution.IsNumberPrimitive(typeRef.Name)
	if isString {
		if rules.MinLength != nil {
			constraints = append(
				constraints,
				fmt.Sprintf("min_length=%d", *rules.MinLength),
			)
		}
		if rules.MaxLength != nil {
			constraints = append(
				constraints,
				fmt.Sprintf("max_length=%d", *rules.MaxLength),
			)
		}
	}
	if isNumber {
		if rules.Min != nil {
			if rules.Min.IsInt {
				constraints = append(constraints, fmt.Sprintf("ge=%d", rules.Min.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("ge=%f", rules.Min.Float))
			}
		}
		if rules.Max != nil {
			if rules.Max.IsInt {
				constraints = append(constraints, fmt.Sprintf("le=%d", rules.Max.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("le=%f", rules.Max.Float))
			}
		}
	}
	if defaultVal != nil {
		switch defaultVal.Kind {
		case resolution.ValueKindBool:
			if defaultVal.BoolValue {
				constraints = append(constraints, "default=True")
			} else {
				constraints = append(constraints, "default=False")
			}
		case resolution.ValueKindInt:
			// Type-aware: uuid with default 0 → UUID(int=0)
			if isUUIDType(typeRef, table) && defaultVal.IntValue == 0 {
				data.imports.addUUID("UUID")
				constraints = append(constraints, "default=UUID(int=0)")
			} else if wrapper := resolveDistinctWrapper(typeRef, table, data); wrapper != "" {
				// Wrap int defaults in the distinct type constructor (e.g., TimeSpan(0))
				constraints = append(
					constraints,
					fmt.Sprintf("default=%s(%d)", wrapper, defaultVal.IntValue),
				)
			} else {
				constraints = append(
					constraints,
					fmt.Sprintf("default=%d", defaultVal.IntValue),
				)
			}
		case resolution.ValueKindFloat:
			constraints = append(
				constraints,
				fmt.Sprintf("default=%f", defaultVal.FloatValue),
			)
		case resolution.ValueKindString:
			constraints = append(
				constraints,
				fmt.Sprintf("default=%q", defaultVal.StringValue),
			)
		case resolution.ValueKindIdent:
			// Handle identifier-based defaults like "now" for timestamps
			if defaultVal.IdentValue == "now" && isTimeStampType(typeRef, table) {
				constraints = append(constraints, "default_factory=telem.TimeStamp.now")
			}
			// Handle "create" for auto-generating keys. uuid keys generate a UUID
			// object; string keys generate a stringified UUID. uuid is a string
			// primitive, so check it first.
			// Use key.ResolvePrimitive to handle type aliases like `Key distinct string`.
			primitive := key.ResolvePrimitive(typeRef, table)
			if defaultVal.IdentValue == "create" {
				if primitive == "uuid" {
					data.imports.addUUID("uuid4")
					constraints = append(constraints, "default_factory=uuid4")
				} else if isString || primitive == "string" {
					data.imports.addUUID("uuid4")
					constraints = append(
						constraints,
						"default_factory=lambda: str(uuid4())",
					)
				}
			}
			if ev, ok := validation.ResolveEnumVariant(
				defaultVal.IdentValue,
				typeRef,
				table,
			); ok {
				variantRef := enumVariantToPython(ev, table, data)
				constraints = append(constraints, fmt.Sprintf("default=%s", variantRef))
			}
		case resolution.ValueKindArray:
			// Lists are mutable, so they must use default_factory, never default=.
			if len(defaultVal.Elements) == 0 {
				constraints = append(constraints, "default_factory=list")
			} else {
				constraints = append(
					constraints,
					fmt.Sprintf(
						"default_factory=lambda: %s",
						pyDefaultLiteral(typeRef, *defaultVal, table, data),
					),
				)
			}
		case resolution.ValueKindStruct:
			// Records (dict[str, Any]) and maps are mutable dicts, so they use
			// default_factory. An empty default becomes dict; a populated one a dict
			// literal. Other struct types default to a model constructor.
			kind, _ := resolution.CollectionKind(typeRef, table)
			if typeRef.Name == "record" || kind == "Map" {
				if len(defaultVal.Fields) == 0 {
					constraints = append(constraints, "default_factory=dict")
				} else {
					constraints = append(
						constraints,
						fmt.Sprintf(
							"default_factory=lambda: %s",
							pyDefaultLiteral(typeRef, *defaultVal, table, data),
						),
					)
				}
			} else {
				// Models are mutable, so they must use default_factory, never default=.
				constraints = append(
					constraints,
					fmt.Sprintf(
						"default_factory=lambda: %s",
						pyDefaultLiteral(typeRef, *defaultVal, table, data),
					),
				)
			}
		}
	}
	return constraints
}

// pyDefaultLiteral renders a default value as a Python literal. typeRef is the
// declared type of the value, used to resolve enum variants, array element
// types, and nested struct field types. Arrays and structs recurse.
func pyDefaultLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	table *resolution.Table,
	data *templateData,
) string {
	switch val.Kind {
	case resolution.ValueKindString:
		return fmt.Sprintf("%q", val.StringValue)
	case resolution.ValueKindInt:
		return fmt.Sprintf("%d", val.IntValue)
	case resolution.ValueKindFloat:
		return fmt.Sprintf("%f", val.FloatValue)
	case resolution.ValueKindBool:
		if val.BoolValue {
			return "True"
		}
		return "False"
	case resolution.ValueKindIdent:
		if ev, ok := validation.ResolveEnumVariant(val.IdentValue, typeRef, table); ok {
			return enumVariantToPython(ev, table, data)
		}
		return val.IdentValue
	case resolution.ValueKindArray:
		elem := pyArrayElementType(typeRef)
		parts := make([]string, 0, len(val.Elements))
		for _, el := range val.Elements {
			parts = append(parts, pyDefaultLiteral(elem, el, table, data))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case resolution.ValueKindStruct:
		if kind, _ := resolution.CollectionKind(
			typeRef,
			table,
		); typeRef.Name == "record" ||
			kind == "Map" {
			return pyRecordLiteral(val, table, data)
		}
		return pyStructLiteral(typeRef, val, table, data)
	}
	return ""
}

// pyRecordLiteral renders a record default as a Python dict literal. Record
// values carry no declared field types, so each entry's value is rendered from
// its own kind.
func pyRecordLiteral(
	val resolution.ExpressionValue,
	table *resolution.Table,
	data *templateData,
) string {
	if len(val.Fields) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(val.Fields))
	for _, fv := range val.Fields {
		parts = append(
			parts,
			fmt.Sprintf(
				"%q: %s",
				fv.Name,
				pyDefaultLiteral(resolution.TypeRef{}, fv.Value, table, data),
			),
		)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// pyStructLiteral renders a struct default as a Python model constructor call,
// resolving each field's value against its declared type in the struct named by
// typeRef.
func pyStructLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	table *resolution.Table,
	data *templateData,
) string {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "None"
	}
	className := getPyName(resolved)
	if resolved.Namespace != data.Namespace {
		outputPath := output.GetPath(resolved, "py")
		if outputPath == "" {
			outputPath = resolved.Namespace
		}
		className = addCrossNamespaceImport(
			toPythonModulePath(outputPath),
			className,
			data,
		)
	}
	fieldsByName := map[string]resolution.Field{}
	for _, f := range resolution.UnifiedFields(resolved, table) {
		fieldsByName[f.Name] = f
	}
	parts := make([]string, 0, len(val.Fields))
	for _, fv := range val.Fields {
		parts = append(
			parts,
			fmt.Sprintf(
				"%s=%s",
				keywords.Escape(fv.Name),
				pyDefaultLiteral(fieldsByName[fv.Name].Type, fv.Value, table, data),
			),
		)
	}
	return className + "(" + strings.Join(parts, ", ") + ")"
}

// pyArrayElementType returns the element type of an array type reference, or the
// reference itself when it is not an array.
func pyArrayElementType(typeRef resolution.TypeRef) resolution.TypeRef {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0]
	}
	return typeRef
}

func enumVariantToPython(
	ev validation.EnumVariant,
	table *resolution.Table,
	data *templateData,
) string {
	// String-valued enums are emitted as a Literal alias with no runtime class to
	// dot into, so reference the variant's string value directly (the field's type
	// is that Literal). Only integer enums become IntEnum classes whose members can
	// be accessed as Enum.variant.
	if form, ok := ev.Type.Form.(resolution.EnumForm); ok && !form.IsIntEnum {
		return fmt.Sprintf("%q", ev.Variant.StringValue())
	}
	enumName := domain.GetName(ev.Type, "py")
	variantRef := fmt.Sprintf("%s.%s", enumName, ev.Variant.Name)
	if ev.Type.Namespace != data.Namespace {
		outputPath := enum.FindOutputPath(ev.Type, table, "py")
		if outputPath == "" {
			outputPath = ev.Type.Namespace
		}
		modulePath := toPythonModulePath(outputPath)
		qualifiedEnum := addCrossNamespaceImport(modulePath, enumName, data)
		variantRef = fmt.Sprintf("%s.%s", qualifiedEnum, ev.Variant.Name)
	}
	return variantRef
}

// isUUIDType checks if a type reference is or resolves to the uuid primitive type.
func isUUIDType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	// Direct uuid primitive
	if typeRef.Name == "uuid" {
		return true
	}
	// Check if it's a distinct type or alias that resolves to uuid
	typ, ok := table.Get(typeRef.Name)
	if !ok {
		return false
	}
	switch form := typ.Form.(type) {
	case resolution.DistinctForm:
		return form.Base.Name == "uuid"
	case resolution.AliasForm:
		return form.Target.Name == "uuid"
	}
	return false
}

// resolveDistinctWrapper returns the Python type name for a distinct type that wraps
// a primitive, or empty string if the type is not a distinct wrapper. This is used to
// wrap default values in the constructor (e.g., TimeSpan(0) instead of just 0).
func resolveDistinctWrapper(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if resolution.IsPrimitive(typeRef.Name) {
		return ""
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return ""
	}
	if _, isDistinct := resolved.Form.(resolution.DistinctForm); !isDistinct {
		return ""
	}
	return typeToPython(typeRef, table, data)
}

// isTimeStampType checks if a type reference is or resolves to a TimeStamp type.
func isTimeStampType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	name := typeRef.Name
	// Check for direct TimeStamp reference (with or without namespace)
	if name == "TimeStamp" || strings.HasSuffix(name, ".TimeStamp") {
		return true
	}
	// Check if it resolves to a TimeStamp
	typ, ok := table.Get(name)
	if !ok {
		return false
	}
	switch form := typ.Form.(type) {
	case resolution.DistinctForm:
		return form.Base.Name == "TimeStamp" ||
			strings.HasSuffix(form.Base.Name, ".TimeStamp")
	case resolution.AliasForm:
		return form.Target.Name == "TimeStamp" ||
			strings.HasSuffix(form.Target.Name, ".TimeStamp")
	}
	return false
}

type intBounds struct {
	min string
	max string
}

var sizedIntBoundsMap = map[string]intBounds{
	"int8":   {min: "-128", max: "127"},
	"int16":  {min: "-32768", max: "32767"},
	"int32":  {min: "-2147483648", max: "2147483647"},
	"int64":  {min: "-9223372036854775808", max: "9223372036854775807"},
	"uint8":  {min: "0", max: "255"},
	"uint12": {min: "0", max: "4095"},
	"uint16": {min: "0", max: "65535"},
	"uint20": {min: "0", max: "1048575"},
	"uint32": {min: "0", max: "4294967295"},
	"uint64": {min: "0", max: "18446744073709551615"},
}

func sizedIntBounds(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) (intBounds, bool) {
	primitive := key.ResolvePrimitive(typeRef, table)
	if primitive == "" {
		primitive = typeRef.Name
	}
	b, ok := sizedIntBoundsMap[primitive]
	return b, ok
}

func addCrossNamespaceImport(modulePath, typeName string, data *templateData) string {
	parts := strings.Split(modulePath, ".")
	if len(parts) >= 2 {
		parentPath := strings.Join(parts[:len(parts)-1], ".")
		moduleName := parts[len(parts)-1]
		alias := data.imports.addModuleImport(parentPath, moduleName)
		return fmt.Sprintf("%s.%s", alias, typeName)
	}
	// Fallback for single-level module path (rare case)
	alias := data.imports.addModuleImport("", modulePath)
	return fmt.Sprintf("%s.%s", alias, typeName)
}

func typeToPython(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	// Handle type parameter references (e.g., V, Details in Status<Details?, V>)
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		tp := typeRef.TypeParam
		// Type params with defaults are substituted with their default value
		if tp.HasDefault() && tp.Default != nil {
			return typeToPython(*tp.Default, table, data)
		}
		// Return the type parameter name (e.g., "V", "Details")
		return tp.Name
	}

	// Handle primitives directly
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToPython(typeRef.Name, data)
	}

	// Handle Array type. The bare element type is returned (not list[...]); field
	// processing wraps array-typed fields in list[...] itself off the element type,
	// so wrapping here would double-wrap. Alias contexts that need list[...] use
	// typeDefBaseToPython / typeRefToPythonAlias instead.
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemType := typeToPython(typeRef.TypeArgs[0], table, data)
		// Fixed-size array -> tuple type
		if typeRef.ArraySize != nil {
			data.imports.addTyping("Tuple")
			elements := make([]string, *typeRef.ArraySize)
			for i := range elements {
				elements[i] = elemType
			}
			return fmt.Sprintf("Tuple[%s]", strings.Join(elements, ", "))
		}
		return elemType
	}

	// Handle Map type
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := typeToPython(typeRef.TypeArgs[0], table, data)
		valueType := typeToPython(typeRef.TypeArgs[1], table, data)
		return fmt.Sprintf("dict[%s, %s]", keyType, valueType)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		data.imports.addTyping("Any")
		return "Any"
	}

	pyName := getPyName(resolved)
	switch resolved.Form.(type) {
	case resolution.StructForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			pyName = addCrossNamespaceImport(modulePath, pyName, data)
		}
		if len(typeRef.TypeArgs) > 0 {
			structForm := resolved.Form.(resolution.StructForm)
			var args []string
			for i, arg := range typeRef.TypeArgs {
				if i < len(structForm.TypeParams) &&
					structForm.TypeParams[i].HasDefault() {
					continue
				}
				args = append(args, typeToPython(arg, table, data))
			}
			if len(args) > 0 {
				return fmt.Sprintf("%s[%s]", pyName, strings.Join(args, ", "))
			}
		}
		return pyName

	case resolution.EnumForm:
		if resolved.Namespace != data.Namespace {
			outputPath := enum.FindOutputPath(resolved, table, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, pyName, data)
		}
		return pyName

	case resolution.DistinctForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, pyName, data)
		}
		return pyName

	case resolution.AliasForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, pyName, data)
		}
		return pyName

	case resolution.UnionForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, pyName, data)
		}
		return pyName

	default:
		data.imports.addTyping("Any")
		return "Any"
	}
}

func typeRefToPythonAlias(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	// Array and map alias targets (e.g. Nodes = Node[], Authorities = map<...>) are
	// builtin generics that do not resolve to a struct, so handle them here rather
	// than falling through to the scalar/struct path below.
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 &&
		typeRef.ArraySize == nil {
		return fmt.Sprintf("list[%s]", typeToPython(typeRef.TypeArgs[0], table, data))
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		return fmt.Sprintf(
			"dict[%s, %s]",
			typeToPython(typeRef.TypeArgs[0], table, data),
			typeToPython(typeRef.TypeArgs[1], table, data),
		)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return typeToPython(typeRef, table, data)
	}

	_, isStruct := resolved.Form.(resolution.StructForm)
	if !isStruct {
		return typeToPython(typeRef, table, data)
	}

	// Get the base type name with proper import handling
	// Use getPyName to respect @py name directives (e.g., GoStatus -> Status)
	pyName := getPyName(resolved)
	var baseName string
	if resolved.Namespace != data.Namespace {
		outputPath := output.GetPath(resolved, "py")
		if outputPath == "" {
			outputPath = resolved.Namespace
		}
		modulePath := toPythonModulePath(outputPath)
		baseName = addCrossNamespaceImport(modulePath, pyName, data)
	} else {
		baseName = pyName
	}

	// If no type arguments, just return the base name
	if len(typeRef.TypeArgs) == 0 {
		return baseName
	}

	// Build type arguments for generic: Status[Details] or Status[Details, Data]
	var typeArgs []string
	for _, arg := range typeRef.TypeArgs {
		typeArgs = append(typeArgs, typeToPython(arg, table, data))
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
}

func toPythonModulePath(repoPath string) string {
	prefixes := []string{
		"client/py/",
		"alamos/py/",
		"freighter/py/",
		"x/py/",
	}

	path := repoPath
	for _, prefix := range prefixes {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			path = after
			break
		}
	}

	return strings.ReplaceAll(path, "/", ".")
}

func primitiveToPython(primitive string, data *templateData) string {
	mapping := primitiveMapper.Map(primitive)
	if mapping.TargetType == "any" {
		data.imports.addTyping("Any")
		return "Any"
	}
	for _, imp := range mapping.Imports {
		switch imp.Category {
		case "uuid":
			data.imports.addUUID(imp.Name)
		case "typing":
			data.imports.addTyping(imp.Name)
		case "synnax":
			data.imports.addSynnax(imp.Name)
		}
	}
	return mapping.TargetType
}

type importManager struct {
	// fieldNames tracks all field names to detect conflicts with module imports.
	fieldNames set.Set[string]
	uuid       []string
	typing     []string
	enum       []string
	pydantic   []string
	synnax     []string
	// ontology holds imports from synnax.ontology.payload.
	ontology []string
	// namespaces maps alias to path.
	namespaces []namespaceImportData
	// modules holds module imports (for "from parent import module").
	modules []moduleImportData
}

func newImportManager() *importManager {
	return &importManager{
		fieldNames: make(set.Set[string]),
	}
}

// registerFieldName adds a field name to the set of known field names.
// This is used to detect conflicts with module import names.
func (m *importManager) registerFieldName(name string) {
	m.fieldNames.Add(name)
}

func (m *importManager) addUUID(name string) {
	if !lo.Contains(m.uuid, name) {
		m.uuid = append(m.uuid, name)
	}
}

func (m *importManager) addTyping(name string) {
	if !lo.Contains(m.typing, name) {
		m.typing = append(m.typing, name)
	}
}

func (m *importManager) addEnum(name string) {
	if !lo.Contains(m.enum, name) {
		m.enum = append(m.enum, name)
	}
}

func (m *importManager) addPydantic(name string) {
	if !lo.Contains(m.pydantic, name) {
		m.pydantic = append(m.pydantic, name)
	}
}

func (m *importManager) addSynnax(name string) {
	if !lo.Contains(m.synnax, name) {
		m.synnax = append(m.synnax, name)
	}
}

func (m *importManager) addOntology(name string) {
	if !lo.Contains(m.ontology, name) {
		m.ontology = append(m.ontology, name)
	}
}

func (m *importManager) addNamespace(alias, path string) {
	if !lo.ContainsBy(
		m.namespaces,
		func(n namespaceImportData) bool { return n.Alias == alias },
	) {
		m.namespaces = append(
			m.namespaces,
			namespaceImportData{Alias: alias, Path: path},
		)
	}
}

// addModuleImport adds a module import and returns the alias to use for referencing types.
// If the module name conflicts with a field name, an alias with underscore suffix is used.
func (m *importManager) addModuleImport(parentPath, moduleName string) string {
	// Check if we already have this module imported
	for _, mod := range m.modules {
		if mod.Module == moduleName {
			return mod.Alias
		}
	}

	// Determine if we need an alias to avoid field name conflicts
	alias := moduleName
	if m.fieldNames.Contains(moduleName) {
		alias = moduleName + "_"
	}

	m.modules = append(m.modules, moduleImportData{
		Module: moduleName,
		Parent: parentPath,
		Alias:  alias,
	})
	return alias
}

// AddImport implements resolver.ImportAdder interface.
// It routes imports to the appropriate category-specific method.
func (m *importManager) AddImport(category, path, name string) {
	switch category {
	case "uuid":
		m.addUUID(name)
	case "typing":
		m.addTyping(name)
	case "enum":
		m.addEnum(name)
	case "pydantic":
		m.addPydantic(name)
	case "synnax":
		m.addSynnax(name)
	case "ontology":
		m.addOntology(name)
	case "internal":
		// For cross-namespace imports, use module imports
		// Note: the alias is returned but not used here; type references
		// use addCrossNamespaceImport which handles aliasing properly
		_ = m.addModuleImport(path, name)
	}
}

type templateData struct {
	imports       *importManager
	Ontology      *ontologyData
	Namespace     string
	OutputPath    string
	Structs       []structData
	Enums         []enumData
	TypeDefs      []typeDefData
	SortedDecls   []sortedDeclData
	TypeVars      []typeVarData
	RebuildModels []string
}

type sortedDeclData struct {
	TypeDef   typeDefData
	Struct    structData
	Union     unionData
	IsTypeDef bool
	IsStruct  bool
	IsUnion   bool
}

type typeDefData struct {
	// Name is the type definition name.
	Name string
	// BaseType is the Python base type (e.g., "int", "str").
	BaseType string
	// IsDistinct indicates whether to use NewType (true) or TypeAlias (false). Currently always false.
	IsDistinct bool
}

type ontologyData struct {
	TypeName   string
	KeyType    string
	StructName string
}

func (d *templateData) UUIDImports() []string     { return d.imports.uuid }
func (d *templateData) TypingImports() []string   { return d.imports.typing }
func (d *templateData) EnumImports() []string     { return d.imports.enum }
func (d *templateData) PydanticImports() []string { return d.imports.pydantic }
func (d *templateData) SynnaxImports() []string   { return d.imports.synnax }
func (d *templateData) OntologyImports() []string { return d.imports.ontology }

func (d *templateData) NamespaceImports() []namespaceImportData { return d.imports.namespaces }

func (d *templateData) ModuleImports() []moduleImportData { return d.imports.modules }

type keyFieldData struct {
	Name   string
	PyType string
}

type typeParamData struct {
	// Name is the TypeVar name (e.g., "V").
	Name string
	// Constraint is the Python bound type, or empty.
	Constraint string
	// IsOptional is true if optional (defaults to Any).
	IsOptional bool
}

type typeVarData struct {
	Name       string
	Constraint string
}

type structData struct {
	Name            string
	Doc             string
	PyName          string
	AliasOf         string
	KeyField        string
	Fields          []fieldData
	TypeParams      []typeParamData
	ExtendsNames    []string
	Skip            bool
	IsAlias         bool
	IsGeneric       bool
	HasExtends      bool
	HasKeywordAlias bool
}

type fieldData struct {
	Name       string
	Alias      string
	Doc        string
	PyType     string
	Default    string
	IsOptional bool
	IsArray    bool
}

type enumData struct {
	Name          string
	LiteralValues string
	Values        []enumValueData
	IsIntEnum     bool
}

type enumValueData struct {
	Name      string
	Value     string
	IntValue  int64
	IsIntEnum bool
}

type namespaceImportData struct {
	Alias string
	Path  string
}

type moduleImportData struct {
	// Module is the module name to import (e.g., "status").
	Module string
	// Parent is the parent path to import from (e.g., "synnax").
	Parent string
	// Alias is the alias for the module (e.g., "color_" to avoid field name conflicts).
	Alias string
}

func googleDocstring(docText string, fields []fieldData) string {
	fds := make([]doc.FieldDoc, 0, len(fields))
	for _, f := range fields {
		if f.Doc != "" {
			fds = append(fds, doc.FieldDoc{Name: f.Name, Doc: f.Doc})
		}
	}
	return doc.FormatPyDocstringGoogle(docText, fds)
}

func hasFieldOrTypeDoc(docText string, fields []fieldData) bool {
	if docText != "" {
		return true
	}
	for _, f := range fields {
		if f.Doc != "" {
			return true
		}
	}
	return false
}

func formatGoogleDocstring(s structData) string {
	return googleDocstring(s.Doc, s.Fields)
}

func hasDocumentation(s structData) bool {
	return hasFieldOrTypeDoc(s.Doc, s.Fields)
}

// formatVariantDocstring renders a union variant's summary docstring. A variant
// inherits its fields from the union base and payload classes, which carry the
// field-level docs, so the variant docstring is a bare summary with no
// Attributes section.
func formatVariantDocstring(v unionVariantData) string {
	return googleDocstring(v.Doc, nil)
}

// pyDocComment renders doc text as a Python line comment, capitalizing the first
// letter (Python convention omits the identifier name) and prefixing every line
// with "# ". It is used to document module-level aliases that cannot carry a
// docstring, such as a discriminated union's Annotated alias.
func pyDocComment(docText string) string {
	if docText == "" {
		return ""
	}
	r := []rune(docText)
	docText = strings.ToUpper(string(r[0])) + string(r[1:])
	lines := strings.Split(docText, "\n")
	for i := range lines {
		lines[i] = "# " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// genericArg returns the Python type argument for a type parameter in Generic[...].
func genericArg(tpd typeParamData) string {
	return tpd.Name
}

// hasTypeParams checks if there are any type params.
func hasTypeParams(params []typeParamData) bool {
	return len(params) > 0
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

var templateFuncs = template.FuncMap{
	"title":                  lo.Capitalize,
	"join":                   strings.Join,
	"upper":                  strings.ToUpper,
	"screamingSnake":         toScreamingSnake,
	"formatGoogleDocstring":  formatGoogleDocstring,
	"hasDocumentation":       hasDocumentation,
	"formatVariantDocstring": formatVariantDocstring,
	"pyDocComment":           pyDocComment,
	"genericArg":             genericArg,
	"hasTypeParams":          hasTypeParams,
}

var fileTemplate = template.Must(
	template.New("python").
		Funcs(templateFuncs).
		Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations
{{- if .UUIDImports }}
from uuid import {{ join .UUIDImports ", " }}
{{- end }}
{{- if .TypingImports }}
from typing import {{ join .TypingImports ", " }}
{{- end }}
{{- if .EnumImports }}
from enum import {{ join .EnumImports ", " }}
{{- end }}
{{- if .PydanticImports }}
from pydantic import {{ join .PydanticImports ", " }}
{{- end }}
{{- if .SynnaxImports }}
from synnax.telem import {{ join .SynnaxImports ", " }}
{{- end }}
{{- if .OntologyImports }}
from synnax.ontology.payload import {{ join .OntologyImports ", " }}
{{- end }}
{{- range .NamespaceImports }}
from {{ .Path }} import {{ .Alias }}
{{- end }}
{{- range .ModuleImports }}
{{- if eq .Module .Alias }}
from {{ .Parent }} import {{ .Module }}
{{- else }}
from {{ .Parent }} import {{ .Module }} as {{ .Alias }}
{{- end }}
{{- end }}
{{- range .TypeDefs }}
{{- if .IsDistinct }}

{{ .Name }} = NewType("{{ .Name }}", {{ .BaseType }})
{{- else }}

{{ .Name }}: TypeAlias = {{ .BaseType }}
{{- end }}
{{- end }}
{{- range .Enums }}
{{- if .IsIntEnum }}


class {{ .Name }}(IntEnum):
{{- range .Values }}
    {{ .Name }} = {{ .IntValue }}
{{- end }}
{{- else }}
{{- $enumName := .Name }}
{{- range .Values }}

{{ screamingSnake $enumName }}_{{ upper .Name }}: Literal["{{ .Value }}"] = "{{ .Value }}"
{{- end }}


{{ .Name }} = Literal[{{ .LiteralValues }}]
{{- end }}
{{- end }}
{{- range .TypeVars }}

{{ .Name }} = TypeVar("{{ .Name }}"{{ if .Constraint }}, bound={{ .Constraint }}{{ end }})
{{- end }}
{{- range .SortedDecls }}
{{- if .IsTypeDef }}
{{- if .TypeDef.IsDistinct }}

{{ .TypeDef.Name }} = NewType("{{ .TypeDef.Name }}", {{ .TypeDef.BaseType }})
{{- else }}

{{ .TypeDef.Name }}: TypeAlias = {{ .TypeDef.BaseType }}
{{- end }}
{{- else if .IsStruct }}
{{- with .Struct }}
{{- if not .Skip }}
{{- if .IsAlias }}


{{ .PyName }} = {{ .AliasOf }}
{{- else if .HasExtends }}


class {{ .PyName }}({{ range $i, $n := .ExtendsNames }}{{ if $i }}, {{ end }}{{ $n }}{{ end }}{{ if hasTypeParams .TypeParams }}, Generic[{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ genericArg $p }}{{ end }}]{{ end }}):
{{- if hasDocumentation . }}
{{ formatGoogleDocstring . }}
{{- end }}
{{- if .HasKeywordAlias }}
    model_config = ConfigDict(populate_by_name=True)
{{ end }}
{{- if or .Fields .KeyField }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .KeyField }}

    def __hash__(self) -> int:
        return hash(self.{{ .KeyField }})
{{- end }}
{{- else if not (hasDocumentation .) }}
    pass
{{- end }}
{{- else if hasTypeParams .TypeParams }}


class {{ .PyName }}(BaseModel, Generic[{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ genericArg $p }}{{ end }}]):
{{- if hasDocumentation . }}
{{ formatGoogleDocstring . }}
{{- end }}
{{- if .HasKeywordAlias }}
    model_config = ConfigDict(populate_by_name=True)
{{ end }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .KeyField }}

    def __hash__(self) -> int:
        return hash(self.{{ .KeyField }})
{{- end }}
{{- else }}


class {{ .PyName }}(BaseModel):
{{- if hasDocumentation . }}
{{ formatGoogleDocstring . }}
{{- end }}
{{- if .HasKeywordAlias }}
    model_config = ConfigDict(populate_by_name=True)
{{ end }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .KeyField }}

    def __hash__(self) -> int:
        return hash(self.{{ .KeyField }})
{{- end }}
{{- if and (not .Fields) (not .KeyField) (not (hasDocumentation .)) (not .HasKeywordAlias) }}
    pass
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- else if .IsUnion }}
{{- with .Union }}
{{- $disc := .DiscName }}
{{- range .Variants }}


class {{ .ClassName }}({{ range $i, $p := .Parents }}{{ if $i }}, {{ end }}{{ $p }}{{ end }}):
{{- if .Doc }}
{{ formatVariantDocstring . }}
{{- end }}
    {{ $disc }}: Literal["{{ .Value }}"]
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- end }}


{{ if .Doc }}{{ pyDocComment .Doc }}
{{ end -}}
{{ .Name }} = Annotated[
    Union[{{ range $i, $v := .Variants }}{{ if $i }}, {{ end }}{{ $v.ClassName }}{{ end }}],
    Field(discriminator="{{ .DiscName }}"),
]
{{- end }}
{{- end }}
{{- end }}
{{- if .RebuildModels }}

{{ range .RebuildModels }}
{{ . }}.model_rebuild()
{{- end }}
{{- end }}
{{- if .Ontology }}


ONTOLOGY_TYPE = ID(type="{{ .Ontology.TypeName }}")


def ontology_id(key: {{ .Ontology.KeyType }}) -> ID:
    return ID(type="{{ .Ontology.TypeName }}", key=str(key))
{{- end }}
`),
)
