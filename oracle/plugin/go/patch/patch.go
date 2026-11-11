// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package patch provides an Oracle plugin that generates the Patch methods that apply
// path actions to typed documents. A document is a struct with @patch fields, and path
// actions edit only those fields. A struct with @patch on the type is a document whose
// fields all take path actions, and it also gets the HashContent methods that hash its
// content. Every struct and union that a document reaches gets the methods over all of
// its fields, unless it is marked @go json hand.
package patch

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const patchDomain = "patch"

// Plugin generates Patch methods for the structs that @patch fields reach.
type Plugin struct{ Options Options }

// Options configures the go/patch plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "patch.gen.go"} }

// New creates a new go/patch plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (*Plugin) Name() string       { return "go/patch" }
func (*Plugin) Domains() []string  { return []string{"go"} }
func (*Plugin) Requires() []string { return []string{"go/types"} }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	c := collector{
		table: req.Resolutions,
		nodes: make(map[string]*node),
		roots: make(set.Set[string]),
	}
	if err := c.collect(); err != nil {
		return nil, err
	}
	byPath := make(map[string][]*node)
	for _, n := range c.nodes {
		path, err := definerPath(req, n.typ)
		if err != nil {
			return nil, err
		}
		byPath[path] = append(byPath[path], n)
	}
	resp := &plugin.Response{}
	for _, path := range slices.Sorted(maps.Keys(byPath)) {
		nodes := byPath[path]
		slices.SortFunc(nodes, func(a, b *node) int {
			return strings.Compare(a.typ.QualifiedName, b.typ.QualifiedName)
		})
		content, err := render(naming.DerivePackageName(path), nodes, req.Resolutions)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to generate patch methods for %s",
				path,
			)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    path + "/" + p.Options.FileNamePattern,
			Content: content,
		})
	}
	deletions, err := stale(req, byPath)
	if err != nil {
		return nil, err
	}
	for _, path := range deletions {
		resp.Deletions = append(resp.Deletions, path+"/"+p.Options.FileNamePattern)
	}
	return resp, nil
}

// node is a struct or union that gets generated methods.
type node struct {
	typ resolution.Type
	// fields are the fields of a struct that Patch edits.
	fields []resolution.Field
	// patched, hashed, and keyed report that the type gets Patch, HashContent, and
	// PatchKey.
	patched, hashed, keyed bool
}

type collector struct {
	table *resolution.Table
	nodes map[string]*node
	roots set.Set[string]
	queue []demand
}

// demand asks for methods on a type that a document reaches.
type demand struct {
	typ                    resolution.Type
	patched, hashed, keyed bool
}

// collect finds the documents and every type that they reach.
func (c *collector) collect() error {
	for _, t := range c.table.StructTypes() {
		whole := domain.HasDomainFromType(t, patchDomain)
		var fields []resolution.Field
		if whole {
			fields = wireFields(t, c.table)
		} else if document(t) {
			fields = slices.DeleteFunc(
				slices.Clone(resolution.UnifiedFields(t, c.table)),
				func(f resolution.Field) bool {
					return !domain.HasDomainFromField(f, patchDomain)
				},
			)
		}
		if len(fields) == 0 {
			continue
		}
		n := &node{typ: t, fields: fields}
		c.roots.Add(t.QualifiedName)
		c.nodes[t.QualifiedName] = n
		if err := c.mark(n, true, whole, false); err != nil {
			return err
		}
	}
	for len(c.queue) > 0 {
		d := c.queue[0]
		c.queue = c.queue[1:]
		name := d.typ.QualifiedName
		if c.roots.Contains(name) {
			return errors.Newf(
				"%s is a document, so no other document can hold it",
				name,
			)
		}
		n, ok := c.nodes[name]
		if !ok {
			n = &node{typ: d.typ}
			if _, isStruct := d.typ.Form.(resolution.StructForm); isStruct {
				n.fields = wireFields(d.typ, c.table)
			}
			c.nodes[name] = n
		}
		if err := c.mark(n, d.patched, d.hashed, d.keyed); err != nil {
			return err
		}
	}
	return nil
}

// mark gives n the methods that the flags name, and queues the types that a new method
// reaches.
func (c *collector) mark(n *node, patched, hashed, keyed bool) error {
	n.keyed = n.keyed || keyed
	patched, hashed = patched && !n.patched, hashed && !n.hashed
	n.patched, n.hashed = n.patched || patched, n.hashed || hashed
	if !patched && !hashed {
		return nil
	}
	for _, f := range c.fieldsOf(n) {
		if patched {
			if err := c.reachPatch(n.typ, f); err != nil {
				return err
			}
		}
		if hashed {
			if err := c.reachHash(f.Type); err != nil {
				return errors.Wrapf(err, "%s.%s", n.typ.QualifiedName, f.Name)
			}
		}
	}
	return nil
}

// fieldsOf returns the fields of a struct, or of every variant of a union.
func (c *collector) fieldsOf(n *node) []resolution.Field {
	form, ok := n.typ.Form.(resolution.UnionForm)
	if !ok {
		return n.fields
	}
	var fields []resolution.Field
	for _, v := range form.Variants {
		fields = append(fields, variantFields(form, v, c.table)...)
	}
	return fields
}

// reachPatch queues the type that f holds when its Patch method takes the path, and
// returns an error for a field that path actions cannot edit.
func (c *collector) reachPatch(owner resolution.Type, f resolution.Field) error {
	if wireOmitted(f) {
		return errors.Newf(
			"%s.%s: a field without a JSON form cannot be @patch",
			owner.QualifiedName, f.Name,
		)
	}
	if f.Optional && typemap.Nilable(f.Type, c.table) {
		return errors.Newf(
			"%s.%s: path actions do not support an optional list or map",
			owner.QualifiedName, f.Name,
		)
	}
	if t, ok := patchable(f.Type, c.table); ok {
		c.queue = append(c.queue, demand{typ: t, patched: true})
	} else if t, ok := keyedList(f.Type, c.table); ok {
		c.queue = append(c.queue, demand{typ: t, patched: true, keyed: true})
	}
	return nil
}

// reachHash queues every struct and union that a value of ref holds.
func (c *collector) reachHash(ref resolution.TypeRef) error {
	sh, err := shapeOf(ref, c.table)
	if err != nil {
		return err
	}
	for sh.elem != nil {
		sh = *sh.elem
	}
	if sh.kind == shapeNested {
		c.queue = append(c.queue, demand{typ: sh.typ, hashed: true})
	}
	return nil
}

// wireFields returns the fields of t that have a JSON form.
func wireFields(t resolution.Type, table *resolution.Table) []resolution.Field {
	return slices.DeleteFunc(
		slices.Clone(resolution.UnifiedFields(t, table)),
		wireOmitted,
	)
}

// variantFields returns the fields of a union variant that have a JSON form. A field
// that the variant declares again replaces the one it inherits.
func variantFields(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	table *resolution.Table,
) []resolution.Field {
	var fields []resolution.Field
	at := make(map[string]int)
	for _, of := range resolver.VariantFields(form, v, "go", table) {
		if wireOmitted(of.Field) {
			continue
		}
		if i, ok := at[of.Field.Name]; ok {
			fields[i] = of.Field
			continue
		}
		at[of.Field.Name] = len(fields)
		fields = append(fields, of.Field)
	}
	return fields
}

// document reports whether t declares path actions: an action that aliases or extends
// patch.Set. A struct that only inherits @patch fields, such as a create request, is
// not one.
func document(t resolution.Type) bool {
	form, ok := t.Form.(resolution.StructForm)
	isSet := func(r resolution.TypeRef) bool { return r.Name == "patch.Set" }
	return ok && slices.ContainsFunc(form.Actions, func(a resolution.Action) bool {
		return isSet(a.Alias) || slices.ContainsFunc(a.Extends, isSet)
	})
}

// patchable returns the struct or union that ref holds when it gets a Patch method. Any
// other value is one that only a whole set replaces.
func patchable(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.Type, bool) {
	if ref.IsTypeParam() || ref.ArraySize != nil {
		return resolution.Type{}, false
	}
	typ, _, ok := resolution.Dealias(ref, table)
	if !ok || omit.IsSkipped(typ, "go") || handJSON(typ) {
		return resolution.Type{}, false
	}
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		return typ, !form.IsGeneric()
	case resolution.UnionForm:
		return typ, true
	default:
		return resolution.Type{}, false
	}
}

// keyedList returns the entry type of a list whose entries paths name by a string key.
func keyedList(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.Type, bool) {
	if ref.Name != "Array" || ref.ArraySize != nil || len(ref.TypeArgs) != 1 {
		return resolution.Type{}, false
	}
	entry, ok := patchable(ref.TypeArgs[0], table)
	if !ok {
		return resolution.Type{}, false
	}
	switch form := entry.Form.(type) {
	case resolution.StructForm:
		_, ok = keyField(wireFields(entry, table), table)
	case resolution.UnionForm:
		ok = len(form.Variants) > 0
		for _, v := range form.Variants {
			if _, has := keyField(variantFields(form, v, table), table); !has {
				ok = false
			}
		}
	}
	return entry, ok
}

// keyField returns the field named key when it holds a string.
func keyField(
	fields []resolution.Field,
	table *resolution.Table,
) (resolution.Field, bool) {
	for _, f := range fields {
		if f.Name != keyName || f.Optional {
			continue
		}
		sh, err := shapeOf(f.Type, table)
		return f, err == nil && sh.kind == shapePrimitive && sh.primitive == "string"
	}
	return resolution.Field{}, false
}

// keyName names the field that identifies a list entry. HashContent leaves it out.
const keyName = "key"

// handJSON reports whether t has a hand-written Go JSON decoder. Only a whole set runs
// the decoder, so a path cannot go into t.
func handJSON(t resolution.Type) bool {
	return domain.GetStringFromType(t, "go", "json") == "hand"
}

func wireOmitted(f resolution.Field) bool {
	return domain.GetStringFromField(f, "go", "wire") == "omit"
}

// shapeKind is the form of a Go value that HashContent writes.
type shapeKind int

const (
	shapePrimitive shapeKind = iota
	shapeEnum
	shapeList
	shapeArray
	shapeNested
)

// shape is the form of the Go value of a type reference.
type shape struct {
	kind shapeKind
	// primitive is the Oracle primitive of a primitive.
	primitive string
	// converted reports that the Go type is not the Go type of the primitive, so a
	// write converts it.
	converted bool
	// intEnum reports that an enum holds integers.
	intEnum bool
	// elem is the entry shape of a list or array.
	elem *shape
	// typ is the struct or union of a nested value.
	typ resolution.Type
}

// shapeOf returns the shape of the Go value of ref, or an error for a value that
// HashContent cannot write.
func shapeOf(ref resolution.TypeRef, table *resolution.Table) (shape, error) {
	if ref.IsTypeParam() {
		return shape{}, errors.New("content hashing does not support a type parameter")
	}
	switch ref.Name {
	case "Array":
		elem, err := shapeOf(ref.TypeArgs[0], table)
		if err != nil {
			return shape{}, err
		}
		kind := shapeList
		if ref.ArraySize != nil {
			kind = shapeArray
		}
		return shape{kind: kind, elem: &elem}, nil
	case "Map":
		return shape{}, errors.New("content hashing does not support a map")
	}
	typ, ok := ref.Resolve(table)
	if !ok {
		return shape{}, errors.Newf("%s does not resolve", ref.Name)
	}
	switch form := typ.Form.(type) {
	case resolution.PrimitiveForm:
		if !hashable.Contains(form.Name) {
			return shape{}, errors.Newf(
				"content hashing does not support the %s primitive",
				form.Name,
			)
		}
		return shape{
			kind:      shapePrimitive,
			primitive: form.Name,
			converted: !native.Contains(form.Name),
		}, nil
	case resolution.AliasForm:
		_, target, ok := resolution.Dealias(ref, table)
		if !ok {
			return shape{}, errors.Newf("%s does not resolve", ref.Name)
		}
		return shapeOf(target, table)
	case resolution.DistinctForm:
		base := resolution.SubstituteTypeRef(
			form.Base,
			resolution.TypeArgMap(form.TypeParams, ref.TypeArgs),
		)
		sh, err := shapeOf(base, table)
		if err != nil {
			return shape{}, err
		}
		if sh.kind == shapeNested {
			return shape{}, errors.Newf(
				"%s: a distinct struct type has no generated methods",
				typ.QualifiedName,
			)
		}
		sh.converted = true
		return sh, nil
	case resolution.EnumForm:
		return shape{kind: shapeEnum, intEnum: form.IsIntEnum}, nil
	case resolution.StructForm:
		if form.IsGeneric() {
			return shape{}, errors.Newf(
				"%s: content hashing does not support a generic struct",
				typ.QualifiedName,
			)
		}
		if omit.IsSkipped(typ, "go") {
			return shape{}, errors.Newf(
				"%s: content hashing does not support a hand-written struct",
				typ.QualifiedName,
			)
		}
		return shape{kind: shapeNested, typ: typ}, nil
	case resolution.UnionForm:
		return shape{kind: shapeNested, typ: typ}, nil
	default:
		return shape{}, errors.Newf(
			"%s: content hashing does not support %T",
			typ.QualifiedName,
			form,
		)
	}
}

// hashable is the set of primitives that HashContent writes.
var hashable = set.New(
	"string", "bool", "uuid", "bytes", "any", "record",
	"int8", "int16", "int32", "int64",
	"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
	"float32", "float64",
)

// native is the set of primitives whose Go type a Hash method takes without a
// conversion.
var native = set.New("string", "bool", "int64", "uint64", "float64", "bytes", "any")

// definerPath returns the @go output path of the package that declares t: the version
// that defines it for a type in a version chain, or its own output path otherwise.
func definerPath(req *plugin.Request, t resolution.Type) (string, error) {
	root := output.GetPath(t, "go")
	if root == "" {
		return "", errors.Newf("%s has no @go output path", t.QualifiedName)
	}
	if req.Versions == nil {
		return root, nil
	}
	livePath := strings.TrimSuffix(t.FilePath, ".oracle")
	chain, ok := req.Versions.Chains()[livePath]
	if !ok {
		return root, nil
	}
	surface, err := req.Versions.Surface(
		context.Background(),
		livePath,
		chain.Current(),
	)
	if err != nil {
		return "", err
	}
	if def, ok := surface[t.Name]; ok {
		return versioning.VersionedPath(root, def.Version), nil
	}
	return versioning.VersionedPath(root, chain.Current()), nil
}

// stale returns every package path that could hold a patch file from an earlier run and
// holds none in this one.
func stale(req *plugin.Request, emitted map[string][]*node) ([]string, error) {
	candidates := make(set.Set[string])
	for _, t := range req.Resolutions.StructTypes() {
		root := output.GetPath(t, "go")
		if root == "" {
			continue
		}
		candidates.Add(root)
		if req.RepoRoot == "" {
			continue
		}
		versions, err := versioning.VersionDirs(req.RepoRoot, root)
		if err != nil {
			return nil, err
		}
		for _, k := range versions {
			candidates.Add(versioning.VersionedPath(root, k))
		}
	}
	var paths []string
	for _, path := range slices.Sorted(maps.Keys(candidates)) {
		if _, ok := emitted[path]; !ok {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

type fileData struct {
	Package string
	Patch   bool
	Content bool
	Structs []structData
	Unions  []unionData
}

// structData is the template view of a struct, or of a union variant.
type structData struct {
	GoName   string
	Receiver string
	// Fields are the fields that Patch edits, or nil when the struct gets no Patch.
	Fields  []fieldData
	Patched bool
	// Key is the PatchKey expression, or empty when the struct gets no PatchKey.
	Key string
	// Hash is the body of HashContent, or nil when the struct gets no HashContent.
	Hash   []string
	Hashed bool
}

type unionData struct {
	GoName        string
	Receiver      string
	Discriminator string
	Patched       bool
	Hashed        bool
	Keyed         bool
	Variants      []variantData
}

type variantData struct {
	Tag    string
	Struct structData
	// Key is the selector of the variant's key, from a variable named v.
	Key string
}

type fieldData struct {
	JSONName string
	// Selector is the Go selector of the field from the receiver, through the parents
	// that the receiver embeds.
	Selector string
	Kind     kind
	// Defaulted reports that a decode may set the field to a value other than its zero
	// value, so a removal leaves that value.
	Defaulted bool
}

// kind is how a Patch method applies an action to a field.
type kind string

const (
	kindValue          kind = "value"
	kindOptionalValue  kind = "optionalValue"
	kindStruct         kind = "struct"
	kindOptionalStruct kind = "optionalStruct"
	kindList           kind = "list"
	kindAny            kind = "any"
	kindOptionalAny    kind = "optionalAny"
)

func render(pkg string, nodes []*node, table *resolution.Table) ([]byte, error) {
	data := fileData{Package: pkg}
	for _, n := range nodes {
		data.Patch = data.Patch || n.patched
		data.Content = data.Content || n.hashed
		switch form := n.typ.Form.(type) {
		case resolution.UnionForm:
			ud, err := unionView(n, form, table)
			if err != nil {
				return nil, err
			}
			data.Unions = append(data.Unions, ud)
		default:
			path := func(name string) []string { return embedPath(n.typ, name, table) }
			sd, err := structView(naming.GetGoName(n.typ), n.fields, path, n, table)
			if err != nil {
				return nil, errors.Wrap(err, n.typ.QualifiedName)
			}
			data.Structs = append(data.Structs, sd)
		}
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func structView(
	goName string,
	fields []resolution.Field,
	path func(name string) []string,
	n *node,
	table *resolution.Table,
) (structData, error) {
	selector := func(f resolution.Field) string {
		return strings.Join(append(path(f.Name), naming.GetFieldName(f)), ".")
	}
	sd := structData{
		GoName:   goName,
		Receiver: receiver(goName),
		Patched:  n.patched,
		Hashed:   n.hashed,
	}
	if n.patched {
		for _, f := range fields {
			sd.Fields = append(sd.Fields, fieldData{
				JSONName:  casing.FieldSnake(f.Name),
				Selector:  selector(f),
				Kind:      fieldKind(f, table),
				Defaulted: defaulted(f, table, make(set.Set[string])),
			})
		}
	}
	if n.keyed {
		key, err := keySelector(sd.Receiver, fields, selector, table)
		if err != nil {
			return structData{}, err
		}
		sd.Key = key
	}
	if n.hashed {
		for _, f := range fields {
			if f.Name == keyName {
				continue
			}
			lines, err := hashField(sd.Receiver+"."+selector(f), f, table)
			if err != nil {
				return structData{}, errors.Wrapf(err, "%s", f.Name)
			}
			sd.Hash = append(sd.Hash, lines...)
		}
	}
	return sd, nil
}

func unionView(
	n *node,
	form resolution.UnionForm,
	table *resolution.Table,
) (unionData, error) {
	goName := naming.GetGoName(n.typ)
	ud := unionData{
		GoName:        goName,
		Receiver:      receiver(goName),
		Discriminator: casing.FieldSnake(form.Discriminator),
		Patched:       n.patched,
		Hashed:        n.hashed,
		Keyed:         n.keyed,
	}
	for _, v := range form.Variants {
		fields := variantFields(form, v, table)
		variant := &node{patched: n.patched, hashed: n.hashed}
		path := func(name string) []string { return variantPath(form, v, name, table) }
		sd, err := structView(
			casing.VariantTypeName(goName, v.Name),
			fields,
			path,
			variant,
			table,
		)
		if err != nil {
			return unionData{}, errors.Wrapf(err, "%s %s", n.typ.QualifiedName, v.Name)
		}
		vd := variantData{Tag: v.Name, Struct: sd}
		if n.keyed {
			selector := func(f resolution.Field) string {
				return strings.Join(append(path(f.Name), naming.GetFieldName(f)), ".")
			}
			if vd.Key, err = keySelector("v", fields, selector, table); err != nil {
				return unionData{}, err
			}
		}
		ud.Variants = append(ud.Variants, vd)
	}
	return ud, nil
}

// keySelector returns the expression for the string key of a value named recv.
func keySelector(
	recv string,
	fields []resolution.Field,
	selector func(resolution.Field) string,
	table *resolution.Table,
) (string, error) {
	f, ok := keyField(fields, table)
	if !ok {
		return "", errors.New("a keyed list entry needs a string key")
	}
	sel := recv + "." + selector(f)
	sh, err := shapeOf(f.Type, table)
	if err != nil {
		return "", err
	}
	if sh.converted {
		return "string(" + sel + ")", nil
	}
	return sel, nil
}

func fieldKind(f resolution.Field, table *resolution.Table) kind {
	if sh, err := shapeOf(f.Type, table); err == nil &&
		sh.kind == shapePrimitive && sh.primitive == "any" && !sh.converted {
		if f.Optional {
			return kindOptionalAny
		}
		return kindAny
	}
	if _, ok := patchable(f.Type, table); ok {
		if f.Optional {
			return kindOptionalStruct
		}
		return kindStruct
	}
	if _, ok := keyedList(f.Type, table); ok {
		return kindList
	}
	if f.Optional {
		return kindOptionalValue
	}
	return kindValue
}

// defaulted reports whether a decode may set the field f to a value other than its
// zero value before it reads its input. visited guards against recursive types.
func defaulted(
	f resolution.Field,
	table *resolution.Table,
	visited set.Set[string],
) bool {
	if f.Optional {
		return false
	}
	if f.Default != nil {
		return !zeroDefault(*f.Default, f.Type, table)
	}
	ref := f.Type
	if ref.IsTypeParam() || ref.Name == "Array" || ref.Name == "Map" {
		return false
	}
	typ, _, ok := resolution.Dealias(ref, table)
	if !ok || visited.Contains(typ.QualifiedName) {
		return false
	}
	if _, ok = typ.Form.(resolution.StructForm); !ok {
		return false
	}
	visited.Add(typ.QualifiedName)
	return slices.ContainsFunc(
		resolution.UnifiedFields(typ, table),
		func(f resolution.Field) bool { return defaulted(f, table, visited) },
	)
}

// zeroDefault reports whether the default v of a field of type ref leaves the Go zero
// value. An ident default other than a union variant or a string enum member, such as
// create or an integer enum member, sets nothing.
func zeroDefault(
	v resolution.ExpressionValue,
	ref resolution.TypeRef,
	table *resolution.Table,
) bool {
	switch v.Kind {
	case resolution.ValueKindIdent:
		if _, ok := validation.ResolveUnionVariant(v.IdentValue, ref, table); ok {
			return false
		}
		ev, ok := validation.ResolveEnumVariant(v.IdentValue, ref, table)
		if !ok {
			return true
		}
		form, _ := ev.Type.Form.(resolution.EnumForm)
		return form.IsIntEnum
	case resolution.ValueKindString:
		return v.StringValue == ""
	case resolution.ValueKindInt:
		return v.IntValue == 0
	case resolution.ValueKindFloat:
		return v.FloatValue == 0
	case resolution.ValueKindBool:
		return !v.BoolValue
	case resolution.ValueKindArray:
		return len(v.Elements) == 0
	default:
		return false
	}
}

// hashField returns the statements that write the field at sel to h.
func hashField(
	sel string,
	f resolution.Field,
	table *resolution.Table,
) ([]string, error) {
	sh, err := shapeOf(f.Type, table)
	if err != nil {
		return nil, err
	}
	if !f.Optional || typemap.Nilable(f.Type, table) {
		return hashValue(sel, false, sh, 0), nil
	}
	lines := []string{
		"if " + sel + " == nil {",
		"\th.Bool(false)",
		"} else {",
		"\th.Bool(true)",
	}
	lines = append(lines, indent(hashValue(sel, true, sh, 0))...)
	return append(lines, "}"), nil
}

// hashValue returns the statements that write the value at sel, or at the pointer sel
// when ptr is set, to h.
func hashValue(sel string, ptr bool, sh shape, depth int) []string {
	value, addressable := sel, sel
	if ptr {
		value, addressable = "*"+sel, "(*"+sel+")"
	}
	switch sh.kind {
	case shapeNested:
		return []string{sel + ".HashContent(h)"}
	case shapeEnum:
		if sh.intEnum {
			return []string{"h.Int(int64(" + value + "))"}
		}
		return []string{"h.String(string(" + value + "))"}
	case shapeList, shapeArray:
		i := fmt.Sprintf("i%d", depth)
		var lines []string
		if sh.kind == shapeList {
			lines = append(lines, "h.Len(len("+addressable+"))")
		}
		lines = append(lines, "for "+i+" := range "+addressable+" {")
		lines = append(
			lines,
			indent(hashValue(addressable+"["+i+"]", false, *sh.elem, depth+1))...,
		)
		return append(lines, "}")
	}
	call := func(method, goType string) []string {
		if sh.converted {
			return []string{"h." + method + "(" + goType + "(" + value + "))"}
		}
		return []string{"h." + method + "(" + value + ")"}
	}
	switch sh.primitive {
	case "string":
		return call("String", "string")
	case "bool":
		return call("Bool", "bool")
	case "uuid":
		return []string{"h.Bytes(" + addressable + "[:])"}
	case "bytes":
		return call("Bytes", "[]byte")
	case "any":
		return []string{"h.Any(" + value + ")"}
	case "record":
		return []string{"h.Any(map[string]any(" + value + "))"}
	case "float32", "float64":
		return convert("Float", "float64", value, sh.primitive, sh.converted)
	case "uint8", "uint12", "uint16", "uint20", "uint32", "uint64":
		return convert("Uint", "uint64", value, sh.primitive, sh.converted)
	default:
		return convert("Int", "int64", value, sh.primitive, sh.converted)
	}
}

// convert returns the call to the Hash method that takes the Go type goType.
func convert(method, goType, value, primitive string, converted bool) []string {
	if !converted && primitive == goType {
		return []string{"h." + method + "(" + value + ")"}
	}
	return []string{"h." + method + "(" + goType + "(" + value + "))"}
}

func indent(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "\t" + l
	}
	return out
}

// embedPath returns the names of the embedded parents that a value of t reaches the
// field named name through, as the Go types plugin lays them out.
func embedPath(t resolution.Type, name string, table *resolution.Table) []string {
	form, ok := t.Form.(resolution.StructForm)
	if !ok || len(form.Extends) == 0 || !gotypes.CanEmbed(form, table) {
		return nil
	}
	if declares(form.Extends, form.Fields, name, table) {
		return nil
	}
	return throughBases(form.Extends, name, table)
}

// variantPath returns the names of the embedded types that a value of the variant
// reaches the field named name through, as the Go types plugin lays them out.
func variantPath(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	name string,
	table *resolution.Table,
) []string {
	payload, ok := v.Type.Resolve(table)
	if !ok {
		return throughBases(form.Extends, name, table)
	}
	if !v.Inline {
		if path := throughBases(form.Extends, name, table); path != nil {
			return path
		}
		return append(
			[]string{naming.GetGoName(payload)},
			embedPath(payload, name, table)...,
		)
	}
	pform := payload.Form.(resolution.StructForm)
	inherited, flattened := resolver.VariantBases(form, v, "go", table)
	if declares(inherited, slices.Concat(flattened, pform.Fields), name, table) {
		return nil
	}
	return throughBases(inherited, name, table)
}

// declares reports whether a struct that embeds bases declares the field named name
// itself, rather than through a base.
func declares(
	bases []resolution.TypeRef,
	fields []resolution.Field,
	name string,
	table *resolution.Table,
) bool {
	return !resolver.DefaultOnlyOverrides(bases, fields, table).Contains(name) &&
		slices.ContainsFunc(fields, named(name))
}

// throughBases returns the embedded path to the field named name through the first of
// bases that holds it.
func throughBases(
	bases []resolution.TypeRef,
	name string,
	table *resolution.Table,
) []string {
	for _, ref := range bases {
		embedded, ok := ref.Resolve(table)
		if !ok {
			continue
		}
		base, _, ok := resolution.Dealias(ref, table)
		if !ok ||
			!slices.ContainsFunc(resolution.UnifiedFields(base, table), named(name)) {
			continue
		}
		return append(
			[]string{naming.GetGoName(embedded)},
			embedPath(base, name, table)...,
		)
	}
	return nil
}

func named(name string) func(resolution.Field) bool {
	return func(f resolution.Field) bool { return f.Name == name }
}

// receiver returns the initials of goName, with a "v" suffix when they would shadow a
// name that a generated method uses.
func receiver(goName string) string {
	var initials []byte
	for i, c := range goName {
		if i == 0 || (c >= 'A' && c <= 'Z') {
			initials = append(initials, byte(c|0x20))
		}
	}
	name := string(initials)
	if reserved.Contains(name) {
		return name + "v"
	}
	return name
}

var reserved = set.New(
	"c", "content", "def", "err", "h", "next", "ok", "patch", "v",
)

// Generated methods call nested methods directly, never through a generic helper, so
// the compiler can keep the document on the stack.
var fileTemplate = template.Must(template.New("go-patch").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{if and .Patch .Content}}
import (
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/patch"
)
{{- else if .Patch}}
import "github.com/synnaxlabs/x/patch"
{{- else if .Content}}
import "github.com/synnaxlabs/x/content"
{{- end}}
{{- range .Structs}}
{{- template "struct" .}}
{{- end}}
{{- range .Unions}}
{{- $u := .}}
{{- $r := .Receiver}}
{{- if .Patched}}

// Patch applies the path action at c, which is not done, to the value.
func ({{$r}} *{{.GoName}}) Patch(c patch.Cursor) error {
	switch v := {{$r}}.Variant.(type) {
{{- range .Variants}}
	case {{.Struct.GoName}}:
		c, ok := c.Guard("{{$u.Discriminator}}", "{{.Tag}}")
		if !ok {
			return nil
		}
		if c.Segment() == "{{$u.Discriminator}}" {
			return patch.Discriminator({{$r}}, c)
		}
		if err := v.Patch(c); err != nil {
			return err
		}
		{{$r}}.Variant = v
		return nil
{{- end}}
	default:
		return patch.Absent({{$r}}, c, "{{.Discriminator}}")
	}
}
{{- end}}
{{- if .Keyed}}

// PatchKey returns the key that paths name the value by in a list.
func ({{$r}} *{{.GoName}}) PatchKey() string {
	switch v := {{$r}}.Variant.(type) {
{{- range .Variants}}
	case {{.Struct.GoName}}:
		return {{.Key}}
{{- end}}
	}
	return ""
}
{{- end}}
{{- if .Hashed}}

// HashContent writes the content of the value to h, without its keys.
func ({{$r}} *{{.GoName}}) HashContent(h *content.Hash) {
	switch v := {{$r}}.Variant.(type) {
{{- range .Variants}}
	case {{.Struct.GoName}}:
		h.String("{{.Tag}}")
		v.HashContent(h)
{{- end}}
	default:
		h.String("")
	}
}
{{- end}}
{{- range .Variants}}
{{- template "struct" .Struct}}
{{- end}}
{{- end}}
{{define "struct"}}
{{- $r := .Receiver}}
{{- $t := .GoName}}
{{- if .Patched}}

// Patch applies the path action at c, which is not done, to the value.
func ({{$r}} *{{.GoName}}) Patch(c patch.Cursor) error {
	switch c.Segment() {
{{- range .Fields}}
	case "{{.JSONName}}":
{{- if and (eq .Kind "struct") .Defaulted}}
		if c = c.Next(); c.Done() {
			return patch.Defaulted(&{{$r}}.{{.Selector}}, c, func(def *{{$t}}) {
				{{$r}}.{{.Selector}} = def.{{.Selector}}
			})
		}
		return {{$r}}.{{.Selector}}.Patch(c)
{{- else if eq .Kind "struct"}}
		if c = c.Next(); c.Done() {
			return patch.Value(&{{$r}}.{{.Selector}}, c)
		}
		return {{$r}}.{{.Selector}}.Patch(c)
{{- else if eq .Kind "optionalStruct"}}
		if c = c.Next(); c.Done() || {{$r}}.{{.Selector}} == nil {
			return patch.OptionalValue(&{{$r}}.{{.Selector}}, c)
		}
		next := *{{$r}}.{{.Selector}}
		if err := next.Patch(c); err != nil {
			return err
		}
		{{$r}}.{{.Selector}} = &next
		return nil
{{- else if eq .Kind "list"}}
		return patch.List(&{{$r}}.{{.Selector}}, c.Next())
{{- else if eq .Kind "any"}}
		return patch.Any(&{{$r}}.{{.Selector}}, c.Next())
{{- else if eq .Kind "optionalAny"}}
		return patch.OptionalAny(&{{$r}}.{{.Selector}}, c.Next())
{{- else if eq .Kind "optionalValue"}}
		return patch.OptionalValue(&{{$r}}.{{.Selector}}, c.Next())
{{- else if .Defaulted}}
		return patch.Defaulted(&{{$r}}.{{.Selector}}, c.Next(), func(def *{{$t}}) {
			{{$r}}.{{.Selector}} = def.{{.Selector}}
		})
{{- else}}
		return patch.Value(&{{$r}}.{{.Selector}}, c.Next())
{{- end}}
{{- end}}
	default:
		return c.UnknownField()
	}
}
{{- end}}
{{- if .Key}}

// PatchKey returns the key that paths name the value by in a list.
func ({{$r}} *{{.GoName}}) PatchKey() string { return {{.Key}} }
{{- end}}
{{- if .Hashed}}

// HashContent writes the content of the value to h, without its keys.
func ({{$r}} *{{.GoName}}) HashContent(h *content.Hash) {
{{- range .Hash}}
	{{.}}
{{- end}}
}
{{- end}}
{{- end}}
`,
))
