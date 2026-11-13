// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package explore serves a browsable map of analyzed schemas: every type, its fields,
// and the references between types.
package explore

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
)

// resourceID is the type of a reference that may point at more than one resource type.
const resourceID = "resource.ID"

// Model is the JSON document the explorer page renders.
type Model struct {
	Namespaces []Namespace `json:"namespaces"`
	Types      []Type      `json:"types"`
}

// Namespace is one schema file.
type Namespace struct {
	Name string `json:"name"`
	// Domain is the schema directory the file lives in: synnax, arc, or x.
	Domain string `json:"domain"`
	// File is the repo-relative path of the schema.
	File string `json:"file"`
	// Outputs maps a language to the directories its bindings generate into.
	Outputs map[string][]string `json:"outputs,omitempty"`
	// Annotations are the file-level domains, such as @pb.
	Annotations []string `json:"annotations,omitempty"`
	// Versions holds the numbers of the resource's version files, ascending.
	Versions []int `json:"versions,omitempty"`
}

// Type is one declared type.
type Type struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	// Kind is struct, union, enum, alias, distinct, or filter.
	Kind string `json:"kind"`
	Doc  string `json:"doc,omitempty"`
	// Resource is the ontology resource type the struct registers as.
	Resource string `json:"resource,omitempty"`
	// Key is the type of the key field, declared or inherited.
	Key string `json:"key,omitempty"`
	// Owner is the resource that this key type identifies.
	Owner string `json:"owner,omitempty"`
	// Persisted is true when the type has a generated codec (@go marshal).
	Persisted bool `json:"persisted,omitempty"`
	// Params are the type parameters as written in the schema.
	Params  []string `json:"params,omitempty"`
	Extends []Ref    `json:"extends,omitempty"`
	// Omits names the inherited fields the struct removes.
	Omits   []string `json:"omits,omitempty"`
	Fields  []Field  `json:"fields,omitempty"`
	Actions []Action `json:"actions,omitempty"`
	// Discriminator is the field whose value selects a union variant.
	Discriminator string    `json:"discriminator,omitempty"`
	Variants      []Variant `json:"variants,omitempty"`
	Values        []Value   `json:"values,omitempty"`
	// Target is the type an alias names, a distinct type wraps, or a filter selects.
	Target *Ref `json:"target,omitempty"`
	// Annotations are the domains declared on the type itself, such as @go marshal.
	Annotations []string `json:"annotations,omitempty"`
}

// Ref is a reference to a type. Name is a qualified type name, a primitive, or a type
// parameter.
type Ref struct {
	Name string `json:"name"`
	Args []Ref  `json:"args,omitempty"`
	// Size is the length of a fixed-size array.
	Size *int64 `json:"size,omitempty"`
	// Param is true when Name is a type parameter.
	Param bool `json:"param,omitempty"`
}

// Field is a struct field, an action field, or a filter line.
type Field struct {
	Name string `json:"name"`
	// Type is nil when an extending struct overrides a field without restating it.
	Type     *Ref `json:"type,omitempty"`
	Optional bool `json:"optional,omitempty"`
	// Default is the default value as written in the schema.
	Default string `json:"default,omitempty"`
	Doc     string `json:"doc,omitempty"`
	// Key is true when the field is the type's @key.
	Key bool `json:"key,omitempty"`
	// Link is set when the field is a stored reference to another resource.
	Link *Link `json:"link,omitempty"`
	// Annotations are the domains declared on the field, such as @validate required.
	Annotations []string `json:"annotations,omitempty"`
}

// Link describes a stored reference declared with @ref.
type Link struct {
	// Targets are the resources the reference may point at.
	Targets []string `json:"targets,omitempty"`
	// Kind is place or use.
	Kind string `json:"kind,omitempty"`
	// OnDelete is restrict, cascade, or detach.
	OnDelete string `json:"on_delete,omitempty"`
	// Acyclic is true when a chain of these references may not return to its start.
	Acyclic bool `json:"acyclic,omitempty"`
}

// Action is a wire mutation declared on a struct or union.
type Action struct {
	Name   string  `json:"name"`
	Doc    string  `json:"doc,omitempty"`
	Fields []Field `json:"fields,omitempty"`
}

// Variant is one shape of a union.
type Variant struct {
	Name string `json:"name"`
	Type Ref    `json:"type"`
	Doc  string `json:"doc,omitempty"`
	// Fields holds the payload of a variant declared inline in the union body.
	Fields []Field `json:"fields,omitempty"`
}

// Value is one enum member.
type Value struct {
	Name string `json:"name"`
	// Value is the member's string or integer value, formatted as text.
	Value string `json:"value"`
	Doc   string `json:"doc,omitempty"`
}

// Build returns the model of every declared type in table. chains supplies the version
// numbers of each versioned resource, keyed by live import path.
func Build(table *resolution.Table, chains map[string]versions.Chain) Model {
	res := newResources(table)
	namespaces := make(map[string]*namespaceBuilder)
	var m Model
	for _, t := range table.Types {
		if t.Synthetic {
			continue
		}
		switch t.Form.(type) {
		case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
			continue
		}
		nb, ok := namespaces[t.Namespace]
		if !ok {
			nb = newNamespaceBuilder(t, chains)
			namespaces[t.Namespace] = nb
		}
		nb.collect(t.Domains)
		m.Types = append(m.Types, buildType(t, table, res))
	}
	slices.SortFunc(m.Types, func(a, b Type) int { return strings.Compare(a.ID, b.ID) })
	for _, name := range slices.Sorted(maps.Keys(namespaces)) {
		m.Namespaces = append(m.Namespaces, namespaces[name].build())
	}
	return m
}

type namespaceBuilder struct {
	ns          Namespace
	outputs     map[string]set.Set[string]
	annotations set.Set[string]
}

func newNamespaceBuilder(
	t resolution.Type,
	chains map[string]versions.Chain,
) *namespaceBuilder {
	file := path.Clean(t.FilePath)
	live := strings.TrimSuffix(file, ".oracle")
	domain := ""
	if parts := strings.Split(live, "/"); len(parts) > 1 {
		domain = parts[1]
	}
	return &namespaceBuilder{
		ns: Namespace{
			Name:     t.Namespace,
			Domain:   domain,
			File:     file,
			Versions: chains[live].Numbers,
		},
		outputs:     make(map[string]set.Set[string]),
		annotations: make(set.Set[string]),
	}
}

func (b *namespaceBuilder) collect(domains map[string]resolution.Domain) {
	for _, d := range domains {
		if out, ok := d.Expressions.Find("output"); ok && len(out.Values) > 0 {
			if b.outputs[d.Name] == nil {
				b.outputs[d.Name] = make(set.Set[string])
			}
			b.outputs[d.Name].Add(out.Values[0].StringValue)
		}
		if _, file := d.AST.(parser.IFileDomainContext); file {
			b.annotations.Add(annotations(d)...)
		}
	}
}

func (b *namespaceBuilder) build() Namespace {
	ns := b.ns
	if len(b.outputs) > 0 {
		ns.Outputs = make(map[string][]string, len(b.outputs))
		for lang, dirs := range b.outputs {
			ns.Outputs[lang] = slices.Sorted(maps.Keys(dirs))
		}
	}
	if len(b.annotations) > 0 {
		ns.Annotations = slices.Sorted(maps.Keys(b.annotations))
	}
	return ns
}

// resources resolves the targets of stored references by the rule the Go generator
// applies: a key type identifies the resource whose record fields declare it as @key,
// and a resource.ID reference names its resource types with @ref types.
type resources struct {
	// byName maps a resource type name to the type registered as it. A persisted type
	// wins over a wire-only one, such as channel.Channel over channel.APIChannel.
	byName map[string]resolution.Type
	// byKey maps a key type to the resource type name it identifies.
	byKey map[string]string
}

func newResources(table *resolution.Table) resources {
	r := resources{
		byName: make(map[string]resolution.Type),
		byKey:  make(map[string]string),
	}
	for _, t := range slices.Concat(table.StructTypes(), table.UnionTypes()) {
		name := resourceName(t)
		if name == "" {
			continue
		}
		if prev, ok := r.byName[name]; !ok || (persisted(t) && !persisted(prev)) {
			r.byName[name] = t
		}
		fields, _ := resolution.RecordFields(t, table)
		for _, f := range fields {
			if _, ok := f.Domains["key"]; ok {
				if _, taken := r.byKey[f.Type.Name]; !taken {
					r.byKey[f.Type.Name] = name
				}
				break
			}
		}
	}
	return r
}

// owner returns the resource that key identifies, or "" when it identifies none.
func (r resources) owner(key string) string {
	return r.named(r.byKey[key])
}

func (r resources) named(name string) string {
	if t, ok := r.byName[name]; ok {
		return t.QualifiedName
	}
	return ""
}

func resourceName(t resolution.Type) string {
	if d, ok := t.Domains["resource"]; ok {
		if e, ok := d.Expressions.Find("type"); ok && len(e.Values) > 0 {
			return e.Values[0].StringValue
		}
	}
	return ""
}

func persisted(t resolution.Type) bool {
	if d, ok := t.Domains["go"]; ok {
		_, ok = d.Expressions.Find("marshal")
		return ok
	}
	return false
}

// keyOf returns the type of t's key: a declared or inherited @key field, or a computed
// key declared with @retrieve key.
func keyOf(t resolution.Type, table *resolution.Table) string {
	if r, ok := resolution.RetrieveKey(t); ok {
		return r.Name
	}
	fields, _ := resolution.RecordFields(t, table)
	for _, f := range fields {
		if _, ok := f.Domains["key"]; ok {
			return f.Type.Name
		}
	}
	return ""
}

func buildType(t resolution.Type, table *resolution.Table, res resources) Type {
	out := Type{
		ID:          t.QualifiedName,
		Name:        t.Name,
		Namespace:   t.Namespace,
		Doc:         doc(t.Domains),
		Resource:    resourceName(t),
		Owner:       res.owner(t.QualifiedName),
		Persisted:   persisted(t),
		Annotations: ownAnnotations(t.Domains),
	}
	switch form := t.Form.(type) {
	case resolution.StructForm:
		out.Kind = "struct"
		out.Params = params(form.TypeParams)
		out.Extends = refs(form.Extends)
		out.Omits = form.OmittedFields
		out.Fields = fields(form.Fields, res)
		out.Actions = actions(form.Actions, res)
		out.Key = keyOf(t, table)
		if t.Filter != nil {
			out.Kind = "filter"
			target := ref(t.Filter.Target)
			out.Target = &target
			out.Fields = fields(t.Filter.Lines, res)
		}
	case resolution.UnionForm:
		out.Kind = "union"
		out.Key = keyOf(t, table)
		out.Discriminator = form.Discriminator
		out.Extends = append(refs(form.Extends), refs(form.Included)...)
		out.Actions = actions(form.Actions, res)
		for _, v := range form.Variants {
			variant := Variant{Name: v.Name, Type: ref(v.Type), Doc: doc(v.Domains)}
			if v.Inline {
				payload := table.MustGet(v.Type.Name).Form.(resolution.StructForm)
				variant.Fields = fields(payload.Fields, res)
			}
			out.Variants = append(out.Variants, variant)
		}
	case resolution.EnumForm:
		out.Kind = "enum"
		out.Extends = refs(form.Extends)
		for _, v := range form.Values {
			out.Values = append(out.Values, Value{
				Name:  v.Name,
				Value: fmt.Sprint(v.Value),
				Doc:   doc(v.Domains),
			})
		}
	case resolution.AliasForm:
		out.Kind = "alias"
		out.Params = params(form.TypeParams)
		target := ref(form.Target)
		out.Target = &target
	case resolution.DistinctForm:
		out.Kind = "distinct"
		out.Params = params(form.TypeParams)
		target := ref(form.Base)
		out.Target = &target
	default:
		panic(fmt.Sprintf("explore: unknown form %T of %s", t.Form, t.QualifiedName))
	}
	return out
}

func fields(in []resolution.Field, res resources) []Field {
	out := make([]Field, 0, len(in))
	for _, f := range in {
		field := Field{
			Name:        f.Name,
			Optional:    f.Optional,
			Doc:         doc(f.Domains),
			Annotations: ownAnnotations(f.Domains),
		}
		if f.HasType() {
			r := ref(f.Type)
			field.Type = &r
		}
		if f.Default != nil {
			field.Default = value(*f.Default)
		}
		_, field.Key = f.Domains["key"]
		if d, ok := f.Domains["ref"]; ok {
			field.Link = link(d, f.Type, res)
		}
		out = append(out, field)
	}
	return out
}

func link(d resolution.Domain, typ resolution.TypeRef, res resources) *Link {
	l := &Link{}
	if target := elem(typ).Name; target != resourceID {
		if owner := res.owner(target); owner != "" {
			l.Targets = []string{owner}
		}
	} else if e, ok := d.Expressions.Find("types"); ok {
		for _, v := range e.Values {
			if owner := res.named(unquoted(v)); owner != "" {
				l.Targets = append(l.Targets, owner)
			}
		}
	}
	if e, ok := d.Expressions.Find("kind"); ok && len(e.Values) > 0 {
		l.Kind = unquoted(e.Values[0])
	}
	if e, ok := d.Expressions.Find("on_delete"); ok && len(e.Values) > 0 {
		l.OnDelete = unquoted(e.Values[0])
	}
	_, l.Acyclic = d.Expressions.Find("acyclic")
	return l
}

// elem unwraps arrays to the referenced element type.
func elem(r resolution.TypeRef) resolution.TypeRef {
	for r.Name == "Array" && len(r.TypeArgs) == 1 {
		r = r.TypeArgs[0]
	}
	return r
}

func actions(in []resolution.Action, res resources) []Action {
	out := make([]Action, 0, len(in))
	for _, a := range in {
		out = append(out, Action{
			Name:   a.Name,
			Doc:    doc(a.Domains),
			Fields: fields(a.Fields, res),
		})
	}
	return out
}

func ref(r resolution.TypeRef) Ref {
	if r.IsTypeParam() {
		return Ref{Name: r.TypeParam.Name, Param: true}
	}
	return Ref{Name: r.Name, Args: refs(r.TypeArgs), Size: r.ArraySize}
}

func refs(in []resolution.TypeRef) []Ref {
	if len(in) == 0 {
		return nil
	}
	out := make([]Ref, len(in))
	for i, r := range in {
		out[i] = ref(r)
	}
	return out
}

func params(in []resolution.TypeParam) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		s := p.Name
		if p.Optional {
			s += "?"
		}
		if p.Constraint != nil {
			s += " extends " + refString(*p.Constraint)
		}
		if p.Default != nil {
			s += " = " + refString(*p.Default)
		}
		out = append(out, s)
	}
	return out
}

func refString(r resolution.TypeRef) string {
	if r.IsTypeParam() {
		return r.TypeParam.Name
	}
	if len(r.TypeArgs) == 0 {
		return r.Name
	}
	args := make([]string, len(r.TypeArgs))
	for i, a := range r.TypeArgs {
		args[i] = refString(a)
	}
	return r.Name + "<" + strings.Join(args, ", ") + ">"
}

// doc returns the @doc text with each paragraph joined onto one line.
func doc(domains map[string]resolution.Domain) string {
	d, ok := domains["doc"]
	if !ok {
		return ""
	}
	e, ok := d.Expressions.Find("value")
	if !ok || len(e.Values) == 0 {
		return ""
	}
	paragraphs := strings.Split(e.Values[0].StringValue, "\n\n")
	for i, p := range paragraphs {
		paragraphs[i] = strings.Join(strings.Fields(p), " ")
	}
	return strings.Join(paragraphs, "\n\n")
}

// ownAnnotations renders every domain declared on the type or field itself. File
// domains
// and output paths belong to the namespace.
func ownAnnotations(domains map[string]resolution.Domain) []string {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(domains)) {
		d := domains[name]
		if name == "doc" {
			continue
		}
		if _, file := d.AST.(parser.IFileDomainContext); file {
			continue
		}
		out = append(out, annotations(d)...)
	}
	return out
}

func annotations(d resolution.Domain) []string {
	var out []string
	for _, e := range d.Expressions {
		if e.Name == "output" {
			continue
		}
		parts := []string{"@" + d.Name, e.Name}
		for _, v := range e.Values {
			parts = append(parts, value(v))
		}
		out = append(out, strings.Join(parts, " "))
	}
	if len(d.Expressions) == 0 {
		out = append(out, "@"+d.Name)
	}
	return out
}

func unquoted(v resolution.ExpressionValue) string {
	if v.Kind == resolution.ValueKindString {
		return v.StringValue
	}
	return value(v)
}

func value(v resolution.ExpressionValue) string {
	switch v.Kind {
	case resolution.ValueKindString:
		return strconv.Quote(v.StringValue)
	case resolution.ValueKindInt:
		return strconv.FormatInt(v.IntValue, 10)
	case resolution.ValueKindFloat:
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	case resolution.ValueKindBool:
		return strconv.FormatBool(v.BoolValue)
	case resolution.ValueKindIdent:
		return v.IdentValue
	case resolution.ValueKindArray:
		out := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			out[i] = value(e)
		}
		return "[" + strings.Join(out, ", ") + "]"
	case resolution.ValueKindStruct:
		out := make([]string, len(v.Fields))
		for i, f := range v.Fields {
			out[i] = f.Name + " = " + value(f.Value)
		}
		return "{" + strings.Join(out, ", ") + "}"
	default:
		panic(fmt.Sprintf("explore: unknown value kind %d", v.Kind))
	}
}
