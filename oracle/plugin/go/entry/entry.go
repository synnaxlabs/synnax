// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package entry provides an Oracle plugin that generates the Gorp entry methods of each
// type marked @go entry: GorpKey, CustomTypeName, and ResourceID. It emits them into
// every versions/vN package that declares the type, so a frozen package keeps the
// methods its own version files describe.
package entry

import (
	"bytes"
	"go/format"
	"maps"
	"path"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Options configures the go/entry plugin.
type Options struct {
	// FileNamePattern is the basename written for each version package.
	FileNamePattern string
	// ResourceImportPath is the Go package supplying resource.ID and the resource.Type
	// constants the generated ResourceID methods use. Its package name must be
	// resource.
	ResourceImportPath string
	// ResourceType is the qualified name of the enum whose members name the resource
	// types.
	ResourceType string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:    "entry.gen.go",
		ResourceImportPath: "github.com/synnaxlabs/synnax/pkg/service/resource",
		ResourceType:       "resource.Type",
	}
}

// Plugin generates Gorp entry methods for types marked @go entry.
type Plugin struct{ options Options }

// New creates a new go/entry plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{options: opts} }

func (*Plugin) Name() string       { return "go/entry" }
func (*Plugin) Domains() []string  { return []string{"go"} }
func (*Plugin) Requires() []string { return []string{"go/types"} }

const uuidImportPath = "github.com/google/uuid"

// primitiveKeys are the primitives a key may resolve to.
var primitiveKeys = set.New(
	"string", "uuid", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32",
	"uint64",
)

// Generate emits one file per version package that declares a @go entry type.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	chainPaths, err := gotypes.ChainPaths(req)
	if err != nil {
		return nil, err
	}
	resp := &plugin.Response{}
	candidates := make(set.Set[string])
	for _, origPath := range slices.Sorted(maps.Keys(chainPaths)) {
		cp := chainPaths[origPath]
		dirs, err := versioning.VersionDirs(req.RepoRoot, origPath)
		if err != nil {
			return nil, err
		}
		for _, k := range dirs {
			candidates.Add(versioning.VersionedPath(origPath, k))
		}
		for _, k := range cp.Numbers {
			if k > cp.Current || (cp.Ended && k == cp.Current) {
				break
			}
			file, err := p.versionFile(req, origPath, cp.LivePath, k)
			if err != nil {
				return nil, err
			}
			if file.Path != "" {
				resp.Files = append(resp.Files, file)
			}
		}
	}
	for _, f := range resp.Files {
		candidates.Remove(path.Dir(f.Path))
	}
	for _, dir := range slices.Sorted(maps.Keys(candidates)) {
		resp.Deletions = append(resp.Deletions, dir+"/"+p.options.FileNamePattern)
	}
	return resp, nil
}

// versionFile emits the entry methods of the types that version k of livePath declares.
// It returns a zero File when the version declares no @go entry type.
func (p *Plugin) versionFile(
	req *plugin.Request,
	origPath, livePath string,
	k int,
) (plugin.File, error) {
	f, err := req.Versions.File(livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	table, ns, err := gotypes.ChainFrozenTable(req, origPath, livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	vkPath := versioning.VersionedPath(origPath, k)
	data := fileData{
		Package: naming.DerivePackageName(vkPath),
		imports: make(set.Set[string]),
	}
	for _, name := range f.Order {
		if _, aliased := f.Aliases[name]; aliased {
			continue
		}
		t, ok := table.Get(ns + "." + name)
		if !ok || t.Synthetic || !domain.HasExprFromType(t, "go", "entry") {
			continue
		}
		e, err := p.newEntry(req, table, ns, t, &data)
		if err != nil {
			return plugin.File{}, errors.Wrapf(err, "%s in %s", t.Name, vkPath)
		}
		data.Entries = append(data.Entries, e)
	}
	if len(data.Entries) == 0 {
		return plugin.File{}, nil
	}
	data.Imports = slices.Sorted(maps.Keys(data.imports))
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return plugin.File{}, err
	}
	content, err := format.Source(buf.Bytes())
	if err != nil {
		return plugin.File{}, errors.Wrapf(err, "format entry methods for %s", vkPath)
	}
	return plugin.File{
		Path:    vkPath + "/" + p.options.FileNamePattern,
		Content: content,
	}, nil
}

// fileData is the template input for one version package.
type fileData struct {
	Package string
	Imports []string
	Entries []entryData
	imports set.Set[string]
}

// entryData is the template input for one @go entry type.
type entryData struct {
	// Name is the Go type name.
	Name string
	// Params holds the receiver's type parameter list, such as "[Details]".
	Params   string
	Receiver string
	// Table is the name @go entry gives the type's Gorp table, or empty for the Go
	// type name.
	Table string
	// KeyType is the Go type GorpKey returns, empty when the type declares no key.
	KeyType string
	// Key reads the key from the receiver.
	Key string
	// ResourceType is the resource.Type constant, empty without @resource type.
	ResourceType string
	// ResourceKey converts the key to the string a resource.ID holds.
	ResourceKey string
}

func (p *Plugin) newEntry(
	req *plugin.Request,
	table *resolution.Table,
	ns string,
	t resolution.Type,
	data *fileData,
) (entryData, error) {
	goName := naming.GetGoName(t)
	e := entryData{
		Name:     goName,
		Receiver: strings.ToLower(goName[:1]),
		Table:    domain.GetStringFromType(t, "go", "entry"),
	}
	var (
		field  resolution.Field
		found  bool
		access = e.Receiver
	)
	switch form := t.Form.(type) {
	case resolution.StructForm:
		field, found = keyField(t, table)
		e.Params = typeParams(form)
	case resolution.UnionForm:
		var err error
		if field, found, err = unionKey(form, table); err != nil {
			return entryData{}, err
		}
		access += ".Base()"
	default:
		return entryData{}, errors.New("@go entry applies only to a struct or union")
	}
	resourceType := domain.GetStringFromType(t, "resource", "type")
	if !found {
		if resourceType != "" {
			return entryData{}, errors.New(
				"@resource type needs a @key field to generate ResourceID",
			)
		}
		return e, nil
	}
	k, err := resolveKey(field.Type, table, ns)
	if err != nil {
		return entryData{}, err
	}
	if k.goType == "uuid.UUID" {
		data.imports.Add(uuidImportPath)
	}
	e.KeyType = k.goType
	e.Key = access + "." + naming.GetFieldName(field)
	if resourceType == "" {
		return e, nil
	}
	constant, err := p.resourceConstant(req, resourceType)
	if err != nil {
		return entryData{}, err
	}
	data.imports.Add(p.options.ResourceImportPath)
	e.ResourceType = constant
	e.ResourceKey = k.toString(e.Key, data)
	return e, nil
}

// typeParams returns the receiver type parameter list of a generic struct, such as
// "[Details]", or "" for a struct that is not generic.
func typeParams(form resolution.StructForm) string {
	params := resolution.NonDefaultedTypeParams(form.TypeParams)
	if len(params) == 0 {
		return ""
	}
	names := make([]string, len(params))
	for i, tp := range params {
		names[i] = tp.Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// unionKey returns the @key field of the base every variant of a union inherits.
func unionKey(
	form resolution.UnionForm,
	table *resolution.Table,
) (resolution.Field, bool, error) {
	ref, ok := resolver.SharedBase(form, "go", table)
	if !ok {
		return resolution.Field{}, false, errors.New(
			"@go entry on a union needs one base that every variant inherits",
		)
	}
	base, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return resolution.Field{}, false, errors.Newf(
			"unresolved union base %s", ref.Name,
		)
	}
	field, found := keyField(base, table)
	return field, found, nil
}

// keyField returns the @key field of a struct, its own or inherited through extends.
func keyField(t resolution.Type, table *resolution.Table) (resolution.Field, bool) {
	return lo.Find(resolution.UnifiedFields(t, table), key.HasKey)
}

// keyInfo describes the Go type of a key field.
type keyInfo struct {
	// goType is the Go type GorpKey returns.
	goType string
	// primitive is the primitive the key's type resolves to.
	primitive string
	// distinct is true when the key's type is a distinct type rather than the
	// primitive or an alias of it.
	distinct bool
}

// resolveKey resolves a key field's type: a primitive, or a type the version declares.
func resolveKey(
	ref resolution.TypeRef,
	table *resolution.Table,
	ns string,
) (keyInfo, error) {
	var k keyInfo
	if resolution.IsPrimitive(ref.Name) {
		k.goType = ref.Name
		if ref.Name == "uuid" {
			k.goType = "uuid.UUID"
		}
	} else {
		t, ok := ref.Resolve(table)
		if !ok || t.Namespace != ns {
			return keyInfo{}, errors.Newf(
				"key type %s must be declared in the same version",
				ref.Name,
			)
		}
		k.goType = naming.GetGoName(t)
	}
	for !resolution.IsPrimitive(ref.Name) {
		t, ok := ref.Resolve(table)
		if !ok {
			return keyInfo{}, errors.Newf("unresolved key type %s", ref.Name)
		}
		switch form := t.Form.(type) {
		case resolution.AliasForm:
			ref = form.Target
		case resolution.DistinctForm:
			k.distinct = true
			ref = form.Base
		default:
			return keyInfo{}, errors.Newf("unsupported key type %s", ref.Name)
		}
	}
	k.primitive = ref.Name
	if !primitiveKeys.Contains(k.primitive) {
		return keyInfo{}, errors.Newf("unsupported key type %s", k.primitive)
	}
	return k, nil
}

// toString returns the expression that converts the key at access to the decimal or
// canonical string a resource.ID holds.
func (k keyInfo) toString(access string, data *fileData) string {
	switch {
	case k.primitive == "string" && k.distinct:
		return "string(" + access + ")"
	case k.primitive == "string":
		return access
	case k.primitive == "uuid" && k.distinct:
		data.imports.Add(uuidImportPath)
		return "uuid.UUID(" + access + ").String()"
	case k.primitive == "uuid":
		return access + ".String()"
	case strings.HasPrefix(k.primitive, "uint"):
		data.imports.Add("strconv")
		return "strconv.FormatUint(uint64(" + access + "), 10)"
	default:
		data.imports.Add("strconv")
		return "strconv.FormatInt(int64(" + access + "), 10)"
	}
}

// resourceConstant returns the Go constant of the resource type member whose value is
// value.
func (p *Plugin) resourceConstant(req *plugin.Request, value string) (string, error) {
	enum, ok := req.Resolutions.Get(p.options.ResourceType)
	if !ok {
		return "", errors.Newf(
			"resource type enum %s not found",
			p.options.ResourceType,
		)
	}
	form, ok := enum.Form.(resolution.EnumForm)
	if !ok {
		return "", errors.Newf("%s is not an enum", p.options.ResourceType)
	}
	for _, v := range form.Values {
		if v.StringValue() == value {
			return "resource." + naming.GetGoName(
				enum,
			) + naming.ToPascalCase(
				v.Name,
			), nil
		}
	}
	return "", errors.Newf("%s has no member %q", p.options.ResourceType, value)
}

var fileTemplate = template.Must(
	template.New("go-entry").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .Imports}}

import (
{{- range .Imports}}
	"{{.}}"
{{- end}}
)
{{- end}}
{{- range .Entries}}
{{- if .Table}}

// CustomTypeName returns the name of the Gorp table that stores {{.Name}}.
func ({{.Name}}{{.Params}}) CustomTypeName() string { return "{{.Table}}" }
{{- end}}
{{- if .KeyType}}

// GorpKey implements gorp.Entry.
func ({{.Receiver}} {{.Name}}{{.Params}}) GorpKey() {{.KeyType}} { return {{.Key}} }
{{- end}}
{{- if .ResourceType}}

// ResourceID returns the resource identifier of the {{.Name}}.
func ({{.Receiver}} {{.Name}}{{.Params}}) ResourceID() resource.ID {
	return resource.ID{Type: {{.ResourceType}}, Key: {{.ResourceKey}}}
}
{{- end}}
{{- end}}
`),
)
