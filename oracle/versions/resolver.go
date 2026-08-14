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
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

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
	// Live holds the namespaces of the live schemas the file imports
	// directly. Their shapes carry no version, so a frozen surface takes them
	// as they stand.
	Live []string
	// LivePaths holds the live import paths behind Live, index-aligned.
	LivePaths []string
	// Order holds the file's declaration names in source order.
	Order []string
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
// per-surface results. Safe for concurrent use.
type Resolver struct {
	// mu serializes resolution; the recursive internals run under one hold.
	mu        sync.Mutex
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

// RepoRoot returns the repository root the resolver loads files from.
func (r *Resolver) RepoRoot() string { return r.loader.RepoRoot() }

// chainTargetRe matches a chain-alias target's declared source text
// ("v0.Key", "v2.Status<D>"). It runs on the pre-resolution AST text: the
// analyzer's bare-name lookup fallback can silently retarget a resolved
// reference to a different version than the file declared.
var chainTargetRe = regexp.MustCompile(
	`^v(0|[1-9][0-9]*)\.([A-Za-z_][A-Za-z0-9_]*)(<.*>)?$`,
)

// File analyzes livePath's version n file, memoized.
func (r *Resolver) File(ctx context.Context, livePath string, n int) (*File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file(ctx, livePath, n)
}

// Definer returns the namespace and name version n of livePath resolves name to,
// following the file's alias lines to the version that declares it. It reports false
// when version n does not carry the name.
func (r *Resolver) Definer(
	ctx context.Context,
	livePath string,
	n int,
	name string,
) (namespace, definer string, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.file(ctx, livePath, n)
	if err != nil {
		return "", "", false
	}
	if alias, isAlias := f.Aliases[name]; isAlias {
		v, def, err := r.followAlias(ctx, livePath, n, name, alias)
		if err != nil {
			return "", "", false
		}
		return DepNS(livePath, v), def.Name, true
	}
	for _, t := range f.Defined {
		if t.Name == name {
			return DepNS(livePath, n), name, true
		}
	}
	return "", "", false
}

func (r *Resolver) file(ctx context.Context, livePath string, n int) (*File, error) {
	chain, ok := r.chains[livePath]
	if !ok {
		return nil, errors.Newf("no version chain for %s", livePath)
	}
	if n < chain.First() || n > chain.Current() {
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
	for _, k := range chain.Numbers {
		if k >= n {
			break
		}
		sib, err := r.file(ctx, livePath, k)
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
			if err := b.addSurface(ctx, depLive, pv, DepNS(depLive, pv)); err != nil {
				return nil, err
			}
		}
	}

	// Direct pins register under the dependency's plain resource namespace
	// (what the file's own references use) and its internal pinned namespace
	// (what deep walks through copies use).
	pins := make(map[string]int)
	var live, livePaths []string
	for _, imp := range ast.AllImportStmt() {
		p := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		depLive, err := LiveFromFilePath(p)
		if err != nil {
			live = append(live, resourceOf(p))
			livePaths = append(livePaths, p)
			// A live import resolves the dependency's current surface like any
			// other import: the one shape of an unversioned resource, or the
			// merged live projection of a versioned one — the home of every
			// resolved (read-time) reference.
			continue
		}
		_, pv, _ := paths.VersionFile(p)
		pins[depLive] = pv
		b.table.MarkImported(p)
		if err := b.addSurface(ctx, depLive, pv, resourceOf(depLive)); err != nil {
			return nil, err
		}
		if err := b.addSurface(ctx, depLive, pv, DepNS(depLive, pv)); err != nil {
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
		Chain:     chain,
		N:         n,
		Table:     b.table,
		Aliases:   declaredAliases(ast, n),
		Pins:      pins,
		Live:      live,
		LivePaths: livePaths,
		Order:     declarationOrder(ast),
	}
	for _, t := range b.table.TypesInNamespace(ns) {
		if _, ok := f.Aliases[t.Name]; ok {
			continue
		}
		f.Defined = append(f.Defined, t)
	}
	if f.Tombstone() {
		if n != chain.Current() {
			return nil, errors.Newf(
				"%s is empty, but the chain continues to v%d; only the last "+
					"version file may be a tombstone", filePath, chain.Current(),
			)
		}
		if n == chain.First() {
			return nil, errors.Newf(
				"%s is empty and the chain's only version; delete the chain "+
					"directory instead of ending an empty chain", filePath,
			)
		}
	}
	r.files[key] = f
	return f, nil
}

// Tombstone reports whether the file declares nothing: no types and no alias lines. An
// empty file ends its chain — the resource left the persisted world at this version.
// Only the chain's last version file may be one.
func (f *File) Tombstone() bool { return len(f.Defined) == 0 && len(f.Aliases) == 0 }

// Ended reports whether livePath's chain has ended: its last version file is a
// tombstone. An ended chain keeps every earlier version frozen, has no current version,
// and leaves the live schema hand-owned.
func (r *Resolver) Ended(ctx context.Context, livePath string) (bool, error) {
	chain, ok := r.chains[livePath]
	if !ok {
		return false, errors.Newf("no version chain for %s", livePath)
	}
	f, err := r.File(ctx, livePath, chain.Current())
	if err != nil {
		return false, err
	}
	return f.Tombstone(), nil
}

// declarationOrder lists the file's top-level declaration names in source
// order.
func declarationOrder(ast parser.ISchemaContext) []string {
	var names []string
	for _, def := range ast.AllDefinition() {
		if name, ok := definitionName(def); ok {
			names = append(names, name)
		}
	}
	return names
}

// declaredAliases extracts the file's chain-alias lines from the parse tree: alias
// declarations whose declared target text names an earlier version.
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

// Surface resolves livePath's complete namespace at version n: every name present at n
// mapped to its defining declaration and chain-resolved doc.
func (r *Resolver) Surface(
	ctx context.Context, livePath string, n int,
) (map[string]Definition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.surface(ctx, livePath, n)
}

func (r *Resolver) surface(
	ctx context.Context, livePath string, n int,
) (map[string]Definition, error) {
	key := livePath + "@" + strconv.Itoa(n)
	if s, ok := r.surfaces[key]; ok {
		return s, nil
	}
	f, err := r.file(ctx, livePath, n)
	if err != nil {
		return nil, err
	}
	var prev map[string]Definition
	if pred, ok := f.Chain.Predecessor(n); ok {
		if prev, err = r.surface(ctx, livePath, pred); err != nil {
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
		// An alias may target any version that carries the name; intermediate alias
		// lines follow transitively to the defining version.
		definer, def, err := r.followAlias(ctx, livePath, n, name, a)
		if err != nil {
			return nil, err
		}
		d := Definition{Version: definer, Type: def}
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

// followAlias resolves an alias line to its defining version and declaration, following
// intermediate alias lines transitively. Callers hold r.mu.
func (r *Resolver) followAlias(
	ctx context.Context,
	livePath string,
	n int,
	name string,
	a Alias,
) (int, resolution.Type, error) {
	cur := a
	for {
		target, err := r.file(ctx, livePath, cur.Version)
		if err != nil {
			return 0, resolution.Type{}, err
		}
		if def, ok := findDefined(target, cur.Name); ok {
			return cur.Version, def, nil
		}
		next, ok := target.Aliases[cur.Name]
		if !ok {
			return 0, resolution.Type{}, errors.Newf(
				"%s v%d aliases %s to v%d, which does not carry it",
				livePath, n, name, cur.Version,
			)
		}
		if next.Version >= cur.Version {
			return 0, resolution.Type{}, errors.Newf(
				"%s v%d alias %s does not descend the chain at v%d",
				livePath, n, name, cur.Version,
			)
		}
		cur = next
	}
}

func findDefined(f *File, name string) (resolution.Type, bool) {
	for _, t := range f.Defined {
		if t.Name == name {
			return t, true
		}
	}
	return resolution.Type{}, false
}

// tableBuilder accumulates one version file's resolution table, tracking which surface
// namespaces are already registered.
type tableBuilder struct {
	r     *Resolver
	table *resolution.Table
	done  set.Set[string]
}

// addSurface registers livePath's surface at version n into the table under namespace
// ns, then recursively registers the transitive pinned surfaces the copied declarations
// reference. Synthetic support types (inline union variant payloads) from every
// contributing definer file are copied alongside the surface members.
func (b *tableBuilder) addSurface(
	ctx context.Context, livePath string, n int, ns string,
) error {
	if b.done.Contains(ns) {
		return nil
	}
	b.done.Add(ns)
	// A pin registered under its plain resource namespace claims that resource: an
	// unversioned schema the file also imports may reach it live, and analyzing that
	// file again would redeclare every type. Marking the live path resolves those
	// references to the pinned surface. Internal namespaces hold copies no source
	// reference names, so they claim nothing.
	if ns == resourceOf(livePath) {
		b.table.MarkImported(livePath)
	}
	surf, err := b.r.surface(ctx, livePath, n)
	if err != nil {
		return err
	}
	definers := make(set.Set[int])
	for _, name := range slices.Sorted(maps.Keys(surf)) {
		def := surf[name]
		definers.Add(def.Version)
		definer, err := b.r.file(ctx, livePath, def.Version)
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
			if err := b.addSurface(ctx, depLive, pv, DepNS(depLive, pv)); err != nil {
				return err
			}
		}
		if err := b.addLive(definer); err != nil {
			return err
		}
	}
	sortedDefiners := definers.Slice()
	slices.Sort(sortedDefiners)
	for _, v := range sortedDefiners {
		definer, err := b.r.file(ctx, livePath, v)
		if err != nil {
			return err
		}
		rw := rewriter(ns, pinRewrites(definer.Pins))
		for _, t := range definer.Defined {
			if !t.Synthetic {
				continue
			}
			ct := copyType(t, ns, rw)
			if _, exists := b.table.Get(ct.QualifiedName); exists {
				continue
			}
			if err := b.table.Add(ct); err != nil {
				return err
			}
		}
	}
	return nil
}

// addLive copies the live shapes a version file resolves against, so a frozen surface
// built from it resolves them too. Live types carry no version, so they register under
// their own namespace unchanged. The live path is marked imported so a later import of
// the same schema resolves to these copies instead of redeclaring every type.
func (b *tableBuilder) addLive(f *File) error {
	for i, ns := range f.Live {
		if b.done.Contains(ns) {
			continue
		}
		b.done.Add(ns)
		b.table.MarkImported(f.LivePaths[i])
		for _, t := range f.Table.TypesInNamespace(ns) {
			if _, exists := b.table.Get(t.QualifiedName); exists {
				continue
			}
			if err := b.table.Add(t); err != nil {
				return errors.Wrapf(err, "failed to register live namespace %s", ns)
			}
		}
	}
	return nil
}

// SurfaceInto registers livePath's surface at version n into table under namespace ns,
// plus every transitive pinned dependency surface under its DepNS namespace. Namespaces
// already present in the table are not re-registered.
func (r *Resolver) SurfaceInto(
	ctx context.Context,
	table *resolution.Table,
	livePath string,
	n int,
	ns string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	done := make(set.Set[string])
	for _, t := range table.Types {
		if t.Namespace != "" {
			done.Add(t.Namespace)
		}
	}
	b := &tableBuilder{r: r, table: table, done: done}
	return b.addSurface(ctx, livePath, n, ns)
}

// DepNS is the internal namespace a pinned dependency surface registers under. The
// livePath-based form keeps same-named resources from different domains distinct; "@"
// never appears in schema identifiers, so these names cannot collide with user
// namespaces.
func DepNS(livePath string, n int) string {
	return strings.TrimPrefix(livePath, "schemas/") + "@v" + strconv.Itoa(n)
}

// ParseDepNS inverts DepNS, recovering the live path and version.
func ParseDepNS(ns string) (livePath string, n int, ok bool) {
	base, v, found := strings.Cut(ns, "@v")
	if !found {
		return "", 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return "", 0, false
	}
	return "schemas/" + base, n, true
}

// resourceOf returns the resource qualifier of a live import path.
func resourceOf(livePath string) string { return path.Base(livePath) }
