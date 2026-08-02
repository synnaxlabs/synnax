// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// File is one analyzed version file.
type File struct {
	Chain Chain
	// N is the version this file declares.
	N int
	// Table resolves every reference the file's declarations make: sibling
	// versions under their vK namespaces, direct dependency pins under the
	// dependency's resource name, and every pinned surface additionally under
	// an internal "<domain>/<resource>@vK" namespace, so deep walks resolve
	// exactly the pinned shapes even across transitive pin skew.
	Table *resolution.Table
	// Defined holds the file's full declarations — types whose persisted
	// shape changed at N — in namespace vN.
	Defined []resolution.Type
	// Aliases maps each unchanged type name to the version file defining it.
	Aliases map[string]Alias
	// Pins maps dependency live paths to the version this file pins.
	Pins map[string]int
}

// Alias is one `Name = vK.Name` line: the type is unchanged at this version
// and defined at Version under Name.
type Alias struct {
	Version int
	Name    string
}

// Definition locates a name's defining declaration within a chain surface.
type Definition struct {
	// Version is the defining version.
	Version int
	// Type is the defining declaration, in its vN namespace.
	Type resolution.Type
	// Doc is the chain-resolved documentation: the nearest @doc declared at
	// or before Version for this name.
	Doc string
}

// Resolver analyzes version files and their chains, caching per-file and
// per-surface results. It is not safe for concurrent use.
type Resolver struct {
	chains    map[string]Chain
	loader    analyzer.FileLoader
	files     map[string]*File
	surfaces  map[string]map[string]Definition
	resolving set.Set[string]
}

// NewResolver creates a Resolver over the discovered chains, loading files
// through loader.
func NewResolver(chains map[string]Chain, loader analyzer.FileLoader) *Resolver {
	return &Resolver{
		chains:    chains,
		loader:    loader,
		files:     make(map[string]*File),
		surfaces:  make(map[string]map[string]Definition),
		resolving: set.New[string](),
	}
}

// Chains returns the discovered chains, keyed by live import path.
func (r *Resolver) Chains() map[string]Chain { return r.chains }

// chainTargetRe matches a chain-alias target's declared source text
// ("v0.Key", "v2.Status<D>"). It runs on the pre-resolution AST text: the
// analyzer's bare-name lookup fallback can silently retarget a resolved
// reference to a different version than the file declared.
var chainTargetRe = regexp.MustCompile(
	`^v(0|[1-9][0-9]*)\.([A-Za-z_][A-Za-z0-9_]*)(<.*>)?$`,
)

// File analyzes livePath's version n file, memoized.
func (r *Resolver) File(ctx context.Context, livePath string, n int) (*File, error) {
	chain, ok := r.chains[livePath]
	if !ok {
		return nil, errors.Newf("no version chain for %s", livePath)
	}
	if n < 0 || n > chain.Current() {
		return nil, errors.Newf("%s has no version %d", livePath, n)
	}
	key := chain.FilePath(n)
	if f, ok := r.files[key]; ok {
		return f, nil
	}
	if r.resolving.Contains(key) {
		return nil, errors.Newf("version dependency cycle through %s", key)
	}
	r.resolving.Add(key)
	defer r.resolving.Remove(key)

	source, filePath, err := r.loader.Load(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load %s", key)
	}
	ast, parseDiag := parser.Parse(source)
	if parseDiag != nil && !parseDiag.Ok() {
		return nil, errors.Newf(
			"failed to parse %s:\n%s", filePath, parseDiag.String(),
		)
	}

	b := &tableBuilder{r: r, table: resolution.NewTable(), done: set.New[string]()}

	// Sibling versions seed the table with their defined declarations only,
	// so an alias targeting a non-defining version fails resolution — the
	// alias-to-definer rule enforced structurally.
	for k := range n {
		sib, err := r.File(ctx, livePath, k)
		if err != nil {
			return nil, err
		}
		rw := rewriter("", pinRewrites(sib.Pins))
		for _, t := range sib.Defined {
			if err := b.table.Add(copyType(t, t.Namespace, rw)); err != nil {
				return nil, errors.Wrapf(err, "failed to seed %s v%d", livePath, k)
			}
		}
		for depLive, pv := range sib.Pins {
			if err := b.addSurface(ctx, depLive, pv, pinNS(depLive, pv)); err != nil {
				return nil, err
			}
		}
	}

	// Direct pins register under the dependency's plain resource namespace
	// (what the file's own references use) and its internal pinned namespace
	// (what deep walks through copies use).
	pins := make(map[string]int)
	for _, imp := range ast.AllImportStmt() {
		p := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		depLive, err := LiveFromFilePath(p)
		if err != nil {
			return nil, errors.Newf(
				"%s imports %q: version files may only import other version files",
				filePath, p,
			)
		}
		_, pv, _ := paths.VersionFile(p)
		pins[depLive] = pv
		b.table.MarkImported(p)
		if err := b.addSurface(ctx, depLive, pv, resourceOf(depLive)); err != nil {
			return nil, err
		}
		if err := b.addSurface(ctx, depLive, pv, pinNS(depLive, pv)); err != nil {
			return nil, err
		}
	}

	ns := fmt.Sprintf("v%d", n)
	diag := analyzer.AnalyzeSeeded(ctx, source, filePath, ns, r.loader, b.table)
	if diag != nil && !diag.Ok() {
		return nil, errors.Newf(
			"analysis of %s failed:\n%s", filePath, diag.String(),
		)
	}

	f := &File{
		Chain:   chain,
		N:       n,
		Table:   b.table,
		Aliases: declaredAliases(ast, n),
		Pins:    pins,
	}
	for _, t := range b.table.TypesInNamespace(ns) {
		if _, ok := f.Aliases[t.Name]; ok {
			continue
		}
		f.Defined = append(f.Defined, t)
	}
	r.files[key] = f
	return f, nil
}

// declaredAliases extracts the file's chain-alias lines from the parse tree:
// alias declarations whose declared target text names an earlier version.
func declaredAliases(ast parser.ISchemaContext, n int) map[string]Alias {
	aliases := make(map[string]Alias)
	for _, def := range ast.AllDefinition() {
		s := def.StructDef()
		if s == nil {
			continue
		}
		sa, ok := s.(*parser.StructAliasContext)
		if !ok {
			continue
		}
		m := chainTargetRe.FindStringSubmatch(sa.TypeRef().GetText())
		if m == nil {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil || v >= n {
			continue
		}
		aliases[sa.IDENT().GetText()] = Alias{Version: v, Name: m[2]}
	}
	return aliases
}

// Surface resolves livePath's complete namespace at version n: every name
// present at n mapped to its defining declaration and chain-resolved doc.
func (r *Resolver) Surface(
	ctx context.Context, livePath string, n int,
) (map[string]Definition, error) {
	key := livePath + "@" + strconv.Itoa(n)
	if s, ok := r.surfaces[key]; ok {
		return s, nil
	}
	f, err := r.File(ctx, livePath, n)
	if err != nil {
		return nil, err
	}
	var prev map[string]Definition
	if n > 0 {
		if prev, err = r.Surface(ctx, livePath, n-1); err != nil {
			return nil, err
		}
	}
	surf := make(map[string]Definition, len(f.Defined)+len(f.Aliases))
	for _, t := range f.Defined {
		if t.Synthetic {
			continue
		}
		d := Definition{Version: n, Type: t, Doc: doc.Get(t.Domains)}
		if d.Doc == "" {
			if p, ok := prev[t.Name]; ok {
				d.Doc = p.Doc
			}
		}
		surf[t.Name] = d
	}
	for name, a := range f.Aliases {
		target, err := r.File(ctx, livePath, a.Version)
		if err != nil {
			return nil, err
		}
		def, ok := findDefined(target, a.Name)
		if !ok {
			return nil, errors.Newf(
				"%s v%d aliases %s to v%d, which does not define it",
				livePath, n, name, a.Version,
			)
		}
		d := Definition{Version: a.Version, Type: def}
		if p, ok := prev[name]; ok {
			d.Doc = p.Doc
		} else {
			d.Doc = doc.Get(def.Domains)
		}
		surf[name] = d
	}
	r.surfaces[key] = surf
	return surf, nil
}

func findDefined(f *File, name string) (resolution.Type, bool) {
	for _, t := range f.Defined {
		if t.Name == name {
			return t, true
		}
	}
	return resolution.Type{}, false
}

// tableBuilder accumulates one version file's resolution table, tracking
// which surface namespaces are already registered.
type tableBuilder struct {
	r     *Resolver
	table *resolution.Table
	done  set.Set[string]
}

// addSurface registers livePath's surface at version n into the table under
// namespace ns, then recursively registers the transitive pinned surfaces the
// copied declarations reference.
func (b *tableBuilder) addSurface(
	ctx context.Context, livePath string, n int, ns string,
) error {
	if b.done.Contains(ns) {
		return nil
	}
	b.done.Add(ns)
	surf, err := b.r.Surface(ctx, livePath, n)
	if err != nil {
		return err
	}
	for name, def := range surf {
		definer, err := b.r.File(ctx, livePath, def.Version)
		if err != nil {
			return err
		}
		ct := copyType(def.Type, ns, rewriter(ns, pinRewrites(definer.Pins)))
		ct.Name = name
		ct.QualifiedName = ns + "." + name
		if err := b.table.Add(ct); err != nil {
			return errors.Wrapf(
				err, "failed to register %s v%d as %s", livePath, n, ns,
			)
		}
		for depLive, pv := range definer.Pins {
			if err := b.addSurface(ctx, depLive, pv, pinNS(depLive, pv)); err != nil {
				return err
			}
		}
	}
	return nil
}

// pinNS is the internal namespace a pinned dependency surface registers
// under. The livePath-based form keeps same-named resources from different
// domains distinct; "@" never appears in schema identifiers, so these names
// cannot collide with user namespaces.
func pinNS(livePath string, n int) string {
	return strings.TrimPrefix(livePath, "schemas/") + "@v" + strconv.Itoa(n)
}

// resourceOf returns the resource qualifier of a live import path.
func resourceOf(livePath string) string { return path.Base(livePath) }
