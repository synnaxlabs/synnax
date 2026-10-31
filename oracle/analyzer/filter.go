// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/pluralize"
	"github.com/synnaxlabs/x/set"
)

const (
	// filterKeyword separates a filter definition's name from its target.
	filterKeyword = "filter"
	// filterName is the name of every filter definition, which the generators and the
	// clients rely on.
	filterName = "Where"
)

// treeFields are the fields every expanded filter starts with.
var treeFields = []string{"and", "or", "not"}

// builderNames are the names that the generated builders declare, so a line cannot take
// them. A line also cannot start with includePrefix. The checks compare snake_case
// names, since each generator converts the case of a line.
var builderNames = []string{
	"where", "limit", "offset", "search", "one", "filter", "builder",
}

const includePrefix = "include_"

// collectFilter registers a filter definition as a struct holding the tree fields and
// the typed lines. expandFilters adds the bare lines once the target resolves.
func collectFilter(c *analysisCtx, def parser.IFilterDefContext) {
	idents := def.AllIDENT()
	if len(idents) < 2 || def.TypeRef() == nil {
		return
	}
	name := idents[0].GetText()
	if kw := idents[1].GetText(); kw != filterKeyword {
		c.report(diagnostics.Errorf(
			def, "%s: expected 'filter' before the target, got %q", name, kw,
		))
		return
	}
	if name != filterName {
		c.report(diagnostics.Errorf(
			def, "filter %s: a filter must be named %s", name, filterName,
		))
		return
	}
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		c.report(diagnostics.Errorf(def, "duplicate type definition: %s", qname))
		return
	}
	if !plainTypeRef(def.TypeRef()) {
		c.report(diagnostics.Errorf(
			def,
			"filter %s: the target must be a plain type reference, got %q",
			name,
			def.TypeRef().GetText(),
		))
		return
	}
	filter := &resolution.Filter{Target: collectTypeRef(def.TypeRef(), nil)}
	self := resolution.TypeRef{Name: name}
	form := resolution.StructForm{Fields: []resolution.Field{
		treeField("and", arrayOf(self), "matches when every child matches."),
		treeField("or", arrayOf(self), "matches when any child matches."),
		treeField("not", self, "matches when the child does not match."),
	}}
	domains := make(map[string]resolution.Domain)
	if body := def.FilterBody(); body != nil {
		names := set.New(treeFields...)
		for _, fd := range body.AllFieldDef() {
			line, ok := collectFilterLine(c, name, fd)
			if !ok {
				continue
			}
			snake := casing.TypeSnake(line.Name)
			if names.Contains(snake) {
				msg := "filter %s: duplicate line %q"
				if slices.Contains(treeFields, snake) {
					msg = "filter %s: %q is reserved for the tree"
				}
				c.report(diagnostics.Errorf(fd, msg, name, line.Name))
				continue
			}
			if slices.Contains(builderNames, snake) ||
				strings.HasPrefix(snake, includePrefix) {
				c.report(diagnostics.Errorf(
					fd,
					"filter %s: %q is reserved for the builder",
					name,
					line.Name,
				))
				continue
			}
			names.Add(snake)
			filter.Lines = append(filter.Lines, line)
			if line.HasType() {
				form.Fields = append(form.Fields, lineField(line, line.Type))
			}
		}
		for _, d := range body.AllDomain() {
			addDomain(domains, collectDomain(d))
		}
	}
	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          form,
		Domains:       mergeDomains(c.fileDomains, domains),
		AST:           def,
		Filter:        filter,
	}))
}

func collectFilterLine(
	c *analysisCtx, filterName string, def parser.IFieldDefContext,
) (resolution.Field, bool) {
	field := collectField(def, nil)
	var problem string
	switch {
	case field.Optional:
		problem = "is always optional; remove the ?"
	case field.Default != nil:
		problem = "cannot have a default"
	case len(field.OmittedDomains) > 0:
		problem = "cannot remove domains"
	}
	if problem != "" {
		c.report(diagnostics.Errorf(
			def, "filter %s: line %s %s", filterName, field.Name, problem,
		))
		return resolution.Field{}, false
	}
	return field, true
}

func treeField(name string, typ resolution.TypeRef, doc string) resolution.Field {
	return resolution.Field{
		Name:     name,
		Type:     typ,
		Optional: true,
		Domains:  map[string]resolution.Domain{"doc": docValue(doc)},
	}
}

// lineField returns the struct field a line expands to. A line without a doc gets one.
func lineField(line resolution.Field, typ resolution.TypeRef) resolution.Field {
	domains := cloneDomains(line.Domains)
	if _, ok := domains["doc"]; !ok {
		domains["doc"] = docValue(fmt.Sprintf("selects by %s.", line.Name))
	}
	return resolution.Field{
		AST:      line.AST,
		Name:     line.Name,
		Type:     typ,
		Optional: true,
		Domains:  domains,
	}
}

func cloneDomains(m map[string]resolution.Domain) map[string]resolution.Domain {
	out := make(map[string]resolution.Domain, len(m)+1)
	maps.Copy(out, m)
	return out
}

func arrayOf(elem resolution.TypeRef) resolution.TypeRef {
	return resolution.TypeRef{Name: "Array", TypeArgs: []resolution.TypeRef{elem}}
}

func docValue(doc string) resolution.Domain {
	return resolution.Domain{
		Name: "doc",
		Expressions: resolution.Expressions{{
			Name: "value",
			Values: []resolution.ExpressionValue{{
				Kind:        resolution.ValueKindString,
				StringValue: doc,
			}},
		}},
	}
}

// declaresFilter reports whether the file declares a filter, whose bare lines read the
// types of the filter schema.
func declaresFilter(ast parser.ISchemaContext) bool {
	return lo.SomeBy(ast.AllDefinition(), func(def parser.IDefinitionContext) bool {
		return def.FilterDef() != nil
	})
}

// importsFilterSchema reports whether the file imports a schema in the filter
// namespace.
func importsFilterSchema(ast parser.ISchemaContext) bool {
	for _, imp := range ast.AllImportStmt() {
		path := imp.STRING_LIT().GetText()
		if paths.DeriveNamespace(path[1:len(path)-1]) == resolution.FilterNamespace {
			return true
		}
	}
	return false
}

// expandFilters resolves each filter's target and adds its bare lines as fields typed
// by their operator types, in declared line order.
func expandFilters(c *analysisCtx) {
	for i := range c.table.Types {
		if typ := c.table.Types[i]; typ.Namespace == c.namespace && typ.Filter != nil {
			c.table.Types[i] = expandFilter(c, typ)
		}
	}
}

func expandFilter(c *analysisCtx, typ resolution.Type) resolution.Type {
	filter := *typ.Filter
	filter.Lines = slices.Clone(filter.Lines)
	typ.Filter = &filter
	def := typ.AST.(parser.IFilterDefContext)
	written := def.TypeRef().GetText()
	report := func(format string) resolution.Type {
		c.report(diagnostics.Errorf(def, "filter %s: "+format, typ.Name, written))
		return typ
	}
	resolveTypeRef(c, &typ, &filter.Target)
	declared, ok := c.table.Get(filter.Target.Name)
	if !ok {
		return report("unknown target %s")
	}
	if declared.Namespace != typ.Namespace {
		return report("the target %s must be declared in the same schema")
	}
	target, fields, ok := filterTarget(filter.Target, c.table)
	if !ok {
		return report("the target %s must be a struct or a union")
	}
	hasBare := lo.SomeBy(filter.Lines, func(l resolution.Field) bool {
		return !l.HasType()
	})
	if hasBare && !importsFilterSchema(c.ast) {
		c.report(diagnostics.Errorf(
			def,
			"filter %s: bare lines need an import of the filter schema "+
				"(\"schemas/x/filter\")",
			typ.Name,
		))
		return typ
	}
	form := typ.Form.(resolution.StructForm)
	typed := make(map[string]resolution.Field)
	for _, f := range form.Fields[len(treeFields):] {
		typed[f.Name] = f
	}
	expanded := slices.Clone(form.Fields[:len(treeFields)])
	for i, line := range filter.Lines {
		if line.HasType() {
			f := typed[line.Name]
			filter.Lines[i].Type = f.Type
			expanded = append(expanded, f)
			continue
		}
		op, ok := bareLineOperator(c, typ.Name, target, fields, line)
		if !ok {
			continue
		}
		resolveTypeRef(c, &typ, &op)
		expanded = append(expanded, lineField(line, op))
	}
	form.Fields = expanded
	typ.Form = form
	return typ
}

// filterTarget dealiases a filter's target and returns it with the fields its records
// hold.
func filterTarget(
	ref resolution.TypeRef, table *resolution.Table,
) (resolution.Type, []resolution.Field, bool) {
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return resolution.Type{}, nil, false
	}
	fields, ok := resolution.RecordFields(target, table)
	return target, fields, ok
}

// bareLineOperator returns the operator type a bare line expands to, chosen by the type
// of the target field it names.
func bareLineOperator(
	c *analysisCtx,
	filterName string,
	target resolution.Type,
	fields []resolution.Field,
	line resolution.Field,
) (resolution.TypeRef, bool) {
	report := func(format string, args ...any) (resolution.TypeRef, bool) {
		c.report(diagnostics.Errorf(
			line.AST.(parser.IFieldDefContext),
			"filter %s: "+format,
			append([]any{filterName}, args...)...,
		))
		return resolution.TypeRef{}, false
	}
	field, ok := lo.Find(fields, func(f resolution.Field) bool {
		return f.Name == line.Name
	})
	if !ok {
		if key, ok := resolution.RetrieveKey(target); ok && line.Name == "key" {
			return operator(resolution.OperatorSet, key), true
		}
		if form, ok := target.Form.(resolution.UnionForm); ok &&
			line.Name == form.Discriminator {
			return report(
				"%s takes its %s from an unnamed enum; declare an enum and borrow it "+
					"with `union on %s <Enum>`",
				target.Name,
				line.Name,
				line.Name,
			)
		}
		return report(
			"%s has no field %s; give the line a type to make it a typed line",
			target.Name,
			line.Name,
		)
	}
	if field.Optional {
		return report(
			"%s.%s is optional, which bare lines do not support",
			target.Name,
			field.Name,
		)
	}
	ref := field.Type
	if ref.IsTypeParam() {
		return report(
			"%s.%s has the type parameter %s, which has no default; give the line a "+
				"type to make it a typed line",
			target.Name,
			field.Name,
			ref.TypeParam.Name,
		)
	}
	if ref.Name == "Array" && len(ref.TypeArgs) == 1 {
		return operator(resolution.OperatorContains, ref.TypeArgs[0]), true
	}
	if _, isKey := field.Domains["key"]; isKey {
		return operator(resolution.OperatorSet, ref), true
	}
	if resolved, _, ok := resolution.Dealias(ref, c.table); ok {
		_, isKeyType := resolved.Domains["key"]
		_, isEnum := resolved.Form.(resolution.EnumForm)
		if isKeyType || isEnum {
			return operator(resolution.OperatorSet, ref), true
		}
	}
	base := resolution.PrimitiveBase(ref, c.table)
	distinct := resolution.IsDistinct(ref, c.table)
	switch {
	case base == "string" && !distinct:
		return resolution.TypeRef{Name: resolution.OperatorString}, true
	case base == "bool" && !distinct:
		return resolution.TypeRef{Name: resolution.OperatorBool}, true
	case base == "uuid" || base == "string" || base == "bool":
		return operator(resolution.OperatorSet, ref), true
	case resolution.IsNumberPrimitive(base):
		return operator(resolution.OperatorOrdered, ref), true
	}
	return report(
		"%s.%s has no operators for its type; give the line a type to make it a "+
			"typed line",
		target.Name,
		field.Name,
	)
}

func operator(name string, arg resolution.TypeRef) resolution.TypeRef {
	return resolution.TypeRef{Name: name, TypeArgs: []resolution.TypeRef{arg}}
}

// synthesizeRetrieveTypes derives the retrieve request and response of a live schema's
// filter. A schema declares at most one filter.
func synthesizeRetrieveTypes(c *analysisCtx) {
	if _, _, inVersionFile := paths.VersionFile(c.filePath); inVersionFile {
		return
	}
	where, ok := lo.Find(
		c.table.TypesInNamespace(c.namespace),
		func(t resolution.Type) bool { return t.Filter != nil },
	)
	if !ok {
		return
	}
	if expr, ok := where.Domains["go"].Expressions.Find("request_output"); ok &&
		(len(expr.Values) != 1 || expr.Values[0].Kind != resolution.ValueKindString) {
		c.report(diagnostics.Errorf(
			where.AST.(parser.IFilterDefContext),
			"filter %s: @go request_output takes one path",
			where.Name,
		))
		return
	}
	target, fields, ok := filterTarget(where.Filter.Target, c.table)
	if !ok {
		return
	}
	plural := casing.TypeSnake(pluralize.String(target.Name))
	words := strings.ReplaceAll(plural, "_", " ")
	// The request goes wherever the tree it carries goes, unless the filter names a
	// request output.
	domains := atRequestOutput(
		layDomains(c.fileDomains, where.Domains, "output", "omit"),
		where,
	)
	request := []resolution.Field{
		optionalField(
			"where",
			resolution.TypeRef{Name: where.QualifiedName},
			fmt.Sprintf(
				"selects the %s to retrieve. An absent where selects all of them.",
				words,
			),
		),
	}
	if lo.SomeBy(fields, func(f resolution.Field) bool {
		_, ok := f.Domains["search"]
		return ok
	}) {
		request = append(request, defaultField(
			"search_term",
			resolution.TypeRef{Name: "string"},
			fmt.Sprintf(
				"keeps the %s that match the term, closest match first.", words,
			),
			resolution.ExpressionValue{Kind: resolution.ValueKindString},
		))
	}
	request = append(
		request,
		optionalField(
			"limit",
			resolution.TypeRef{Name: "uint32"},
			fmt.Sprintf("is the most %s to return.", words),
		),
		defaultField(
			"offset",
			resolution.TypeRef{Name: "uint32"},
			fmt.Sprintf("is the number of matching %s to skip.", words),
			resolution.ExpressionValue{Kind: resolution.ValueKindInt},
		),
	)
	response, records, ok := retrieveResponse(c, where, target, fields, plural, words)
	if !ok {
		return
	}
	for _, f := range records {
		if _, ok := f.Domains["include"]; ok {
			request = append(request, defaultField(
				"include_"+f.Name,
				resolution.TypeRef{Name: "bool"},
				fmt.Sprintf("includes the %s of each record.", f.Name),
				resolution.ExpressionValue{Kind: resolution.ValueKindBool},
			))
		}
	}
	created := []resolution.Type{{
		Name: "RetrieveRequest",
		Form: resolution.StructForm{Fields: request},
		Domains: withDoc(domains, fmt.Sprintf(
			"is a request to retrieve %s.", words,
		)),
	}, response}
	derived := true
	for _, t := range created {
		t.Namespace = c.namespace
		t.QualifiedName = c.namespace + "." + t.Name
		t.FilePath = c.filePath
		if _, exists := c.table.Get(t.QualifiedName); exists {
			c.report(diagnostics.Errorf(
				where.AST.(parser.IFilterDefContext),
				"filter %s: %s derives %s, which the schema already declares",
				where.Name,
				where.Name,
				t.QualifiedName,
			))
			derived = false
			continue
		}
		resolveTypeRefs(c, &t)
		lo.Must0(c.table.Add(t))
	}
	if !derived {
		return
	}
	filter := *where.Filter
	filter.Request = c.namespace + ".RetrieveRequest"
	filter.Response = c.namespace + "." + response.Name
	where.Filter = &filter
	replaceType(c.table, where)
}

// retrieveResponse derives the response and returns it with the fields of the records
// it holds. A `@response type T` on the filter replaces the target with T, and the
// response takes T's outputs.
func retrieveResponse(
	c *analysisCtx,
	where, target resolution.Type,
	fields []resolution.Field,
	plural, words string,
) (resolution.Type, []resolution.Field, bool) {
	elem := resolution.TypeRef{Name: target.QualifiedName}
	// A generic target's table holds records of every type argument.
	if form, ok := target.Form.(resolution.StructForm); ok {
		for _, param := range form.TypeParams {
			if param.HasDefault() {
				break
			}
			elem.TypeArgs = append(elem.TypeArgs, resolution.TypeRef{Name: "any"})
		}
	}
	domains := c.fileDomains
	if dom, ok := where.Domains["response"]; ok {
		def := where.AST.(parser.IFilterDefContext)
		expr, ok := dom.Expressions.Find("type")
		if !ok || len(expr.Values) == 0 ||
			expr.Values[0].Kind != resolution.ValueKindIdent {
			c.report(diagnostics.Errorf(
				def, "filter %s: @response takes `type <Type>`", where.Name,
			))
			return resolution.Type{}, nil, false
		}
		elem = resolution.TypeRef{Name: expr.Values[0].IdentValue}
		resolveTypeRef(c, &where, &elem)
		override, ok := c.table.Get(elem.Name)
		if !ok {
			c.report(diagnostics.Errorf(
				def,
				"filter %s: unknown response type %s",
				where.Name,
				expr.Values[0].IdentValue,
			))
			return resolution.Type{}, nil, false
		}
		if fields, ok = resolution.RecordFields(override, c.table); !ok {
			c.report(diagnostics.Errorf(
				def,
				"filter %s: the response type %s must be a struct or a union",
				where.Name,
				elem.Name,
			))
			return resolution.Type{}, nil, false
		}
		domains = layDomains(c.fileDomains, override.Domains, "output", "omit")
	}
	// The response exists only where the request does.
	domains = atRequestOutput(layDomains(domains, where.Domains, "omit"), where)
	return resolution.Type{
		Name: "RetrieveResponse",
		Form: resolution.StructForm{Fields: []resolution.Field{requiredField(
			plural,
			arrayOf(elem),
			fmt.Sprintf("are the %s the request selects.", words),
		)}},
		Domains: withDoc(domains, fmt.Sprintf(
			"holds the %s a RetrieveRequest selects.", words,
		)),
	}, fields, true
}

// atRequestOutput returns domains with the Go output replaced by the filter's `@go
// request_output`, which places the request and response in another package than the
// tree.
func atRequestOutput(
	domains map[string]resolution.Domain, where resolution.Type,
) map[string]resolution.Domain {
	expr, ok := where.Domains["go"].Expressions.Find("request_output")
	if !ok {
		return domains
	}
	return layDomains(domains, map[string]resolution.Domain{"go": {
		Name:        "go",
		Expressions: resolution.Expressions{{Name: "output", Values: expr.Values}},
	}}, "output")
}

// layDomains returns file with the expressions named in names of each domain in from
// laid over it.
func layDomains(
	file, from map[string]resolution.Domain, names ...string,
) map[string]resolution.Domain {
	out := cloneDomains(file)
	for name, d := range from {
		kept := lo.Filter(d.Expressions, func(e resolution.Expression, _ int) bool {
			return lo.Contains(names, e.Name)
		})
		if len(kept) == 0 {
			continue
		}
		addDomain(out, resolution.Domain{Name: name, Expressions: kept})
	}
	return out
}

func withDoc(
	domains map[string]resolution.Domain, doc string,
) map[string]resolution.Domain {
	out := cloneDomains(domains)
	out["doc"] = docValue(doc)
	return out
}

func optionalField(name string, typ resolution.TypeRef, doc string) resolution.Field {
	f := requiredField(name, typ, doc)
	f.Optional = true
	return f
}

func defaultField(
	name string,
	typ resolution.TypeRef,
	doc string,
	value resolution.ExpressionValue,
) resolution.Field {
	f := requiredField(name, typ, doc)
	f.Default = &value
	return f
}

func requiredField(name string, typ resolution.TypeRef, doc string) resolution.Field {
	return resolution.Field{
		Name:    name,
		Type:    typ,
		Domains: map[string]resolution.Domain{"doc": docValue(doc)},
	}
}

func replaceType(table *resolution.Table, typ resolution.Type) {
	for i := range table.Types {
		if table.Types[i].QualifiedName == typ.QualifiedName {
			table.Types[i] = typ
			return
		}
	}
}
