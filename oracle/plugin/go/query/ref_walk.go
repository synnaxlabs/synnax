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
	"strings"

	"github.com/synnaxlabs/oracle/internal/casing"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// stepKind is the kind of one step from an entry to a @ref field it holds.
type stepKind uint8

const (
	// stepField reads a struct field.
	stepField stepKind = iota
	// stepArray ranges over the elements of an array.
	stepArray
	// stepVariant switches a union to one of its variants.
	stepVariant
	// stepLeaf reads the @ref field itself.
	stepLeaf
)

// refStep is one step from an entry to a @ref field it holds.
type refStep struct {
	kind stepKind
	// field is the Go name of the field a field or leaf step reads.
	field string
	// optional is true when a field step reads a pointer to a struct.
	optional bool
	// elem is the type of an array step's elements.
	elem resolution.TypeRef
	// union and variant name the union type and the variant a variant step switches to.
	union   resolution.Type
	variant string
}

func (s refStep) id() string {
	switch s.kind {
	case stepArray:
		return "[]"
	case stepVariant:
		return "|" + s.union.QualifiedName + "." + s.variant
	default:
		return s.field
	}
}

// leafMode is how a @ref field holds its targets.
type leafMode uint8

const (
	// leafRequired always holds one target.
	leafRequired leafMode = iota
	// leafZero holds no target while it is the zero value it defaults to.
	leafZero
	// leafOptional holds no target while it is nil.
	leafOptional
	// leafSlice holds one target per element.
	leafSlice
)

// refSite is one path from an entry to a @ref field.
type refSite struct {
	steps []refStep
	field resolution.Field
	mode  leafMode
}

// inArray reports whether the site reaches its field through an array.
func (s refSite) inArray() bool {
	return slices.ContainsFunc(s.steps, func(st refStep) bool {
		return st.kind == stepArray
	})
}

// walkedRef groups the sites that share a path, which make up one reference.
type walkedRef struct {
	path  string
	sites []refSite
}

// refWalker finds every @ref field that a type holds through its fields, arrays,
// optional structs, and unions.
type refWalker struct {
	table *resolution.Table
	refs  []*walkedRef
	// declared holds each struct field the walk found a @ref on.
	declared set.Set[fieldDecl]
	// onPath holds the types on the current path, which catches a recursive type.
	onPath set.Set[string]
}

func newRefWalker(table *resolution.Table) *refWalker {
	return &refWalker{
		table:    table,
		declared: make(set.Set[fieldDecl]),
		onPath:   make(set.Set[string]),
	}
}

// walk returns the references that typ holds, in the order its fields declare them.
func (w *refWalker) walk(typ resolution.Type) ([]*walkedRef, error) {
	if _, ok := typ.Form.(resolution.StructForm); !ok {
		return nil, errors.Newf(
			"%s: @retrieve references needs a struct",
			typ.QualifiedName,
		)
	}
	if err := w.walkStruct(typ, nil, nil); err != nil {
		return nil, err
	}
	return w.refs, nil
}

func (w *refWalker) walkStruct(
	typ resolution.Type,
	steps []refStep,
	path []string,
) error {
	if w.onPath.Contains(typ.QualifiedName) {
		if w.holdsRef(typ, make(set.Set[string])) {
			return errors.Newf(
				"%s: @ref fields inside the recursive type %s are not supported",
				typ.QualifiedName,
				typ.Name,
			)
		}
		return nil
	}
	w.onPath.Add(typ.QualifiedName)
	defer w.onPath.Remove(typ.QualifiedName)
	for _, f := range resolution.UnifiedFields(typ, w.table) {
		if err := w.walkField(typ, f, steps, path); err != nil {
			return err
		}
	}
	return nil
}

func (w *refWalker) walkField(
	owner resolution.Type,
	f resolution.Field,
	steps []refStep,
	path []string,
) error {
	path = append(slices.Clone(path), f.Name)
	isSlice := f.Type.Name == "Array" && len(f.Type.TypeArgs) == 1
	if plugindomain.HasDomainFromField(f, "ref") {
		w.declared.Add(fieldDecl{typ: declarer(owner, f.Name, w.table), field: f.Name})
		site := refSite{
			steps: append(slices.Clone(steps), refStep{
				kind:  stepLeaf,
				field: naming.GetFieldName(f),
			}),
			field: f,
			mode:  leafModeOf(f, isSlice),
		}
		w.add(strings.Join(path, "."), site)
		return nil
	}
	elem := f.Type
	if isSlice {
		elem = elem.TypeArgs[0]
	}
	resolved, ref, ok := resolution.Dealias(elem, w.table)
	if !ok {
		return nil
	}
	fieldStep := refStep{
		kind:     stepField,
		field:    naming.GetFieldName(f),
		optional: f.Optional && !isSlice,
	}
	next := append(slices.Clone(steps), fieldStep)
	if isSlice {
		next = append(next, refStep{kind: stepArray, elem: ref})
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		if len(form.TypeParams) > 0 {
			return nil
		}
		return w.walkStruct(resolved, next, path)
	case resolution.UnionForm:
		return w.walkUnion(resolved, form, next, path)
	}
	return nil
}

func (w *refWalker) walkUnion(
	union resolution.Type,
	form resolution.UnionForm,
	steps []refStep,
	path []string,
) error {
	if w.onPath.Contains(union.QualifiedName) {
		if w.holdsRef(union, make(set.Set[string])) {
			return errors.Newf(
				"%s: @ref fields inside the recursive type %s are not supported",
				union.QualifiedName,
				union.Name,
			)
		}
		return nil
	}
	w.onPath.Add(union.QualifiedName)
	defer w.onPath.Remove(union.QualifiedName)
	for _, v := range form.Variants {
		next := append(slices.Clone(steps), refStep{
			kind:    stepVariant,
			union:   union,
			variant: v.Name,
		})
		for _, of := range variantFields(form, v, w.table) {
			if err := w.walkField(of.owner, of.field, next, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// holdsRef reports whether typ holds a @ref field anywhere in its fields.
func (w *refWalker) holdsRef(typ resolution.Type, seen set.Set[string]) bool {
	if seen.Contains(typ.QualifiedName) {
		return false
	}
	seen.Add(typ.QualifiedName)
	var fields []resolution.Field
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		fields = resolution.UnifiedFields(typ, w.table)
	case resolution.UnionForm:
		for _, v := range form.Variants {
			for _, of := range variantFields(form, v, w.table) {
				fields = append(fields, of.field)
			}
		}
	}
	for _, f := range fields {
		if plugindomain.HasDomainFromField(f, "ref") {
			return true
		}
		elem := f.Type
		if elem.Name == "Array" && len(elem.TypeArgs) == 1 {
			elem = elem.TypeArgs[0]
		}
		if t, _, ok := resolution.Dealias(elem, w.table); ok && w.holdsRef(t, seen) {
			return true
		}
	}
	return false
}

func (w *refWalker) add(path string, site refSite) {
	for _, r := range w.refs {
		if r.path == path {
			r.sites = append(r.sites, site)
			return
		}
	}
	w.refs = append(w.refs, &walkedRef{path: path, sites: []refSite{site}})
}

// ownedField is a field with the type that declares it.
type ownedField struct {
	owner resolution.Type
	field resolution.Field
}

// variantFields returns the fields of a union variant: the fields of the union's bases,
// then the fields of the variant's payload.
func variantFields(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	table *resolution.Table,
) []ownedField {
	var out []ownedField
	addBases := func(bases []resolution.TypeRef) {
		for _, ref := range bases {
			base, _, ok := resolution.Dealias(ref, table)
			if !ok {
				continue
			}
			for _, f := range resolution.UnifiedFields(base, table) {
				out = append(out, ownedField{owner: base, field: f})
			}
		}
	}
	payload, ok := v.Type.Resolve(table)
	if !ok {
		addBases(form.Extends)
		return out
	}
	if !v.Inline {
		addBases(form.Extends)
		for _, f := range resolution.UnifiedFields(payload, table) {
			out = append(out, ownedField{owner: payload, field: f})
		}
		return out
	}
	inherited, flattened := resolver.VariantBases(form, v, "go", table)
	addBases(inherited)
	payloadForm := payload.Form.(resolution.StructForm)
	bases := slices.Concat(form.Extends, payloadForm.Extends)
	for _, f := range flattened {
		owner := baseDeclaring(bases, f.Name, table)
		out = append(out, ownedField{owner: owner, field: f})
	}
	for _, f := range payloadForm.Fields {
		out = append(out, ownedField{owner: payload, field: f})
	}
	return out
}

// baseDeclaring returns the first of bases whose fields include the field named name.
func baseDeclaring(
	bases []resolution.TypeRef,
	name string,
	table *resolution.Table,
) resolution.Type {
	for _, ref := range bases {
		base, _, ok := resolution.Dealias(ref, table)
		if !ok {
			continue
		}
		if slices.ContainsFunc(
			resolution.UnifiedFields(base, table),
			func(f resolution.Field) bool {
				return f.Name == name
			},
		) {
			return base
		}
	}
	return resolution.Type{}
}

// declarer returns the qualified name of the struct that declares the field named name
// in typ: typ itself, or the nearest type it extends that does.
func declarer(typ resolution.Type, name string, table *resolution.Table) string {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return typ.QualifiedName
	}
	if _, own := form.Field(name); own {
		return typ.QualifiedName
	}
	for _, ext := range form.Extends {
		parent, _, ok := resolution.Dealias(ext, table)
		if !ok {
			continue
		}
		if slices.ContainsFunc(
			resolution.UnifiedFields(parent, table),
			func(f resolution.Field) bool {
				return f.Name == name
			},
		) {
			return declarer(parent, name, table)
		}
	}
	return typ.QualifiedName
}

func leafModeOf(f resolution.Field, isSlice bool) leafMode {
	switch {
	case isSlice:
		return leafSlice
	case f.Optional:
		return leafOptional
	case isZeroDefault(f.Default):
		return leafZero
	default:
		return leafRequired
	}
}

// nestedRef is a reference that the walk found, with the Go code of its Targets and
// Detach functions.
type nestedRef struct {
	refInfo
	// KeyType and EntryType are the Go types of the key and of the holding entry.
	KeyType   string
	EntryType string
	// TargetsBody is the body of the reference's Targets function.
	TargetsBody string
	// DetachBody is the body of the reference's Detach function, empty when the
	// reference cannot drop a target.
	DetachBody string
	// usesSlices is true when the generated code calls the slices package.
	usesSlices bool
}

// renderer writes the Go code that walks the sites of one reference.
type renderer struct {
	r          *resolver.Resolver
	ctx        *resolver.Context
	info       refInfo
	vars       int
	usesSlices bool
}

func (rd *renderer) nextVar(prefix string) string {
	rd.vars++
	return fmt.Sprintf("%s%d", prefix, rd.vars)
}

func (rd *renderer) target(expr string) string {
	if !rd.info.IsID {
		return rd.info.TargetOf(expr)
	}
	if strings.HasPrefix(expr, "*") {
		expr = "(" + expr + ")"
	}
	return fmt.Sprintf("gorp.Target{Table: string(%s.Type), Key: %s.Key}", expr, expr)
}

// stepNode is a node of the tree that the sites of one reference form.
type stepNode struct {
	step     refStep
	children []*stepNode
}

func buildTree(sites [][]refStep) []*stepNode {
	var roots []*stepNode
	for _, steps := range sites {
		level := &roots
		for _, s := range steps {
			var node *stepNode
			for _, n := range *level {
				if n.step.id() == s.id() {
					node = n
					break
				}
			}
			if node == nil {
				node = &stepNode{step: s}
				*level = append(*level, node)
			}
			level = &node.children
		}
	}
	return roots
}

// variantGoType returns the Go type of the variant a variant step switches to.
func (rd *renderer) variantGoType(s refStep) string {
	unionGo := rd.r.ResolveTypeRef(
		resolution.TypeRef{Name: s.union.QualifiedName},
		rd.ctx,
	)
	name := s.union.Name
	if override := plugindomain.GetStringFromType(
		s.union,
		"go",
		"name",
	); override != "" {
		name = override
	}
	prefix, _ := strings.CutSuffix(unionGo, name)
	return prefix + casing.VariantTypeName(name, s.variant)
}

// walk writes code that reaches every leaf under nodes from expr. When mutable is true,
// the code can assign to each leaf it reaches.
func (rd *renderer) walk(
	b *strings.Builder,
	nodes []*stepNode,
	expr string,
	mutable bool,
	leaf func(b *strings.Builder, expr string),
) {
	var variants []*stepNode
	for _, n := range nodes {
		switch n.step.kind {
		case stepLeaf:
			leaf(b, expr+"."+n.step.field)
		case stepField:
			next := expr + "." + n.step.field
			if n.step.optional {
				fmt.Fprintf(b, "if %s != nil {\n", next)
				rd.walk(b, n.children, next, mutable, leaf)
				b.WriteString("}\n")
				continue
			}
			rd.walk(b, n.children, next, mutable, leaf)
		case stepArray:
			if mutable {
				i := rd.nextVar("i")
				fmt.Fprintf(b, "for %s := range %s {\n", i, expr)
				rd.walk(b, n.children, fmt.Sprintf("%s[%s]", expr, i), mutable, leaf)
			} else {
				v := rd.nextVar("v")
				fmt.Fprintf(b, "for _, %s := range %s {\n", v, expr)
				rd.walk(b, n.children, v, mutable, leaf)
			}
			b.WriteString("}\n")
		case stepVariant:
			variants = append(variants, n)
		}
	}
	if len(variants) == 0 {
		return
	}
	v := rd.nextVar("v")
	fmt.Fprintf(b, "switch %s := %s.Variant.(type) {\n", v, expr)
	for _, n := range variants {
		fmt.Fprintf(b, "case %s:\n", rd.variantGoType(n.step))
		rd.walk(b, n.children, v, mutable, leaf)
		if mutable {
			fmt.Fprintf(b, "%s.Variant = %s\n", expr, v)
		}
	}
	b.WriteString("}\n")
}

func (rd *renderer) targetsBody(ref *walkedRef) string {
	var b strings.Builder
	fmt.Fprintf(&b, "var targets []%s\n", rd.info.TargetType())
	steps := make([][]refStep, len(ref.sites))
	for i, s := range ref.sites {
		steps[i] = s.steps
	}
	mode := ref.sites[0].mode
	rd.walk(&b, buildTree(steps), "e", false, func(b *strings.Builder, expr string) {
		switch mode {
		case leafRequired:
			fmt.Fprintf(b, "targets = append(targets, %s)\n", rd.target(expr))
		case leafZero:
			fmt.Fprintf(
				b,
				"if %s != %s {\ntargets = append(targets, %s)\n}\n",
				expr,
				rd.info.ZeroOperand(),
				rd.target(expr),
			)
		case leafOptional:
			fmt.Fprintf(
				b,
				"if %s != nil {\ntargets = append(targets, %s)\n}\n",
				expr,
				rd.target("*"+expr),
			)
		case leafSlice:
			k := rd.nextVar("k")
			fmt.Fprintf(
				b,
				"for _, %s := range %s {\ntargets = append(targets, %s)\n}\n",
				k,
				expr,
				rd.target(k),
			)
		}
	})
	b.WriteString("return targets")
	return b.String()
}

// detachBody returns the code that removes the target t from e, or nothing when the
// sites of ref hold no array to remove a required field's element from.
func (rd *renderer) detachBody(ref *walkedRef) (string, error) {
	var b strings.Builder
	mode := ref.sites[0].mode
	if mode != leafRequired {
		steps := make([][]refStep, len(ref.sites))
		for i, s := range ref.sites {
			steps[i] = s.steps
		}
		rd.walk(&b, buildTree(steps), "e", true, func(b *strings.Builder, expr string) {
			switch mode {
			case leafZero:
				fmt.Fprintf(
					b,
					"if %s == t {\n%s = %s\n}\n",
					rd.target(expr),
					expr,
					rd.info.Zero,
				)
			case leafOptional:
				fmt.Fprintf(
					b,
					"if %s != nil && %s == t {\n%s = nil\n}\n",
					expr,
					rd.target("*"+expr),
					expr,
				)
			case leafSlice:
				rd.usesSlices = true
				k := rd.nextVar("k")
				fmt.Fprintf(
					b,
					"%s = slices.DeleteFunc(%s, func(%s %s) bool {\nreturn %s == t\n})\n",
					expr,
					expr,
					k,
					rd.info.ElemGoType,
					rd.target(k),
				)
			}
		})
		return b.String(), nil
	}
	// A required field drops with the element of the nearest array that holds it, so
	// every site needs an array and the same path to it.
	var (
		prefix   []refStep
		suffixes [][]refStep
	)
	for i, s := range ref.sites {
		at := -1
		for j, st := range s.steps {
			if st.kind == stepArray {
				at = j
			}
		}
		if at < 0 {
			return "", nil
		}
		head := s.steps[:at]
		if i == 0 {
			prefix = head
		} else if !slices.EqualFunc(prefix, head, func(a, b refStep) bool {
			return a.id() == b.id()
		}) {
			return "", errors.Newf(
				"@ref field %s reaches its arrays through different paths, so it "+
					"cannot detach",
				ref.path,
			)
		}
		suffixes = append(suffixes, s.steps[at+1:])
	}
	elem := rd.r.ResolveTypeRef(ref.sites[0].steps[len(prefix)].elem, rd.ctx)
	end := refStep{kind: stepLeaf}
	head := append(slices.Clone(prefix[:len(prefix)-1]), end)
	head[len(head)-1].field = prefix[len(prefix)-1].field
	rd.usesSlices = true
	rd.walk(
		&b,
		buildTree([][]refStep{head}),
		"e",
		true,
		func(b *strings.Builder, expr string) {
			v := rd.nextVar("v")
			fmt.Fprintf(
				b,
				"%s = slices.DeleteFunc(%s, func(%s %s) bool {\n",
				expr,
				expr,
				v,
				elem,
			)
			rd.walk(
				b,
				buildTree(suffixes),
				v,
				false,
				func(b *strings.Builder, expr string) {
					fmt.Fprintf(b, "if %s == t {\nreturn true\n}\n", rd.target(expr))
				},
			)
			b.WriteString("return false\n})\n")
		},
	)
	return b.String(), nil
}

// renderNestedRefs renders the references that walked found on typ, whose entries have
// the Go type entryType and keys of the Go type keyType.
func renderNestedRefs(
	typ resolution.Type,
	walked []*walkedRef,
	keyType, entryType string,
	rt resourceTypes,
	table *resolution.Table,
	r *resolver.Resolver,
	ctx *resolver.Context,
) ([]nestedRef, error) {
	out := make([]nestedRef, 0, len(walked))
	for _, ref := range walked {
		field := ref.sites[0].field
		mode := ref.sites[0].mode
		for _, s := range ref.sites[1:] {
			if s.mode != mode {
				return nil, errors.Newf(
					"%s: @ref field %s holds targets in different shapes",
					typ.QualifiedName,
					ref.path,
				)
			}
		}
		named := field
		named.Name = ref.path
		inArray := mode == leafRequired && !slices.ContainsFunc(
			ref.sites,
			func(s refSite) bool { return !s.inArray() },
		)
		info, err := extractRef(typ, named, "", inArray, rt, table, r, ctx)
		if err != nil {
			return nil, err
		}
		var structField strings.Builder
		for seg := range strings.SplitSeq(ref.path, ".") {
			structField.WriteString(naming.ToPascalCase(seg))
		}
		info.StructField = naming.LowerFirst(structField.String())
		rd := &renderer{r: r, ctx: ctx, info: info}
		nr := nestedRef{
			refInfo:     info,
			KeyType:     keyType,
			EntryType:   entryType,
			TargetsBody: rd.targetsBody(ref),
		}
		detach, err := rd.detachBody(ref)
		if err != nil {
			return nil, errors.Wrap(err, typ.QualifiedName)
		}
		nr.DetachBody, nr.usesSlices = detach, rd.usesSlices
		out = append(out, nr)
	}
	return out, nil
}

// referencesInfo holds extracted data about a @retrieve references type, which a
// generic table stores and which only needs its references generated.
type referencesInfo struct {
	GoName  string
	KeyType string
	Refs    []nestedRef
}

// FuncName returns the name of the unexported function that returns the references.
func (r referencesInfo) FuncName() string {
	return naming.LowerFirst(r.GoName) + "References"
}

func (r referencesInfo) needsSlices() bool {
	return slices.ContainsFunc(
		r.Refs,
		func(ref nestedRef) bool { return ref.usesSlices },
	)
}

func extractReferencesInfo(
	typ resolution.Type,
	table *resolution.Table,
	rt resourceTypes,
	r *resolver.Resolver,
	ctx *resolver.Context,
) (referencesInfo, error) {
	if form, ok := typ.Form.(resolution.StructForm); ok && len(form.TypeParams) > 0 {
		return referencesInfo{}, errors.Newf(
			"%s: @retrieve references does not support a generic type",
			typ.QualifiedName,
		)
	}
	var keyField *resolution.Field
	fields := resolution.UnifiedFields(typ, table)
	for i := range fields {
		if plugindomain.HasDomainFromField(fields[i], "key") {
			keyField = &fields[i]
			break
		}
	}
	if keyField == nil {
		return referencesInfo{}, errors.Newf(
			"%s: @retrieve references needs a @key field",
			typ.QualifiedName,
		)
	}
	walked, err := newRefWalker(table).walk(typ)
	if err != nil {
		return referencesInfo{}, err
	}
	if len(walked) == 0 {
		return referencesInfo{}, errors.Newf(
			"%s: @retrieve references needs a @ref field",
			typ.QualifiedName,
		)
	}
	goName := naming.GetGoName(typ)
	keyType := r.ResolveTypeRef(keyField.Type, ctx)
	refs, err := renderNestedRefs(typ, walked, keyType, goName, rt, table, r, ctx)
	if err != nil {
		return referencesInfo{}, err
	}
	return referencesInfo{GoName: goName, KeyType: keyType, Refs: refs}, nil
}
