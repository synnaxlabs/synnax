// filename: main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// main is our entrypoint.
func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatalf("Usage: %s <import/path/or/dir> <output-file>", os.Args[0])
	}
	importPathOrDir := flag.Arg(0)
	outputFile := flag.Arg(1)

	// 1) Figure out the package name of the directory that will contain the output file.
	outDir := filepath.Dir(outputFile)
	pkgName, err := deducePackageNameFromDir(outDir)
	if err != nil {
		log.Fatalf("Could not deduce package name for directory %q: %v", outDir, err)
	}

	// 2) Load package info from the source we want to re-export
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, importPathOrDir)
	if err != nil {
		log.Fatalf("failed to load package %q: %v", importPathOrDir, err)
	}
	if len(pkgs) == 0 {
		log.Fatalf("no package found for path %q", importPathOrDir)
	}

	// We'll just take the first package
	pkg := pkgs[0]

	// Collect items
	exportedTypes, exportedConsts, exportedVars, exportedFuncs := collectExported(pkg)

	// Generate code, but now we pass in the *output* package name we deduce
	code := generateReexports(pkgName, pkg.PkgPath, exportedTypes, exportedConsts, exportedVars, exportedFuncs)

	// Ensure the output dir exists
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory %q: %v", outDir, err)
	}

	// Write file
	if err := os.WriteFile(outputFile, []byte(code), 0o644); err != nil {
		log.Fatalf("failed to write output file %q: %v", outputFile, err)
	}

	log.Printf("Successfully generated re-exports in %s (package %s)", outputFile, pkgName)
}

// deducePackageNameFromDir scans *.go files in dir for a package declaration,
// returning the first package name found. We only parse package clauses.
func deducePackageNameFromDir(dir string) (string, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// We only care about .go files in that directory
		if filepath.Ext(e.Name()) != ".go" {
			continue
		}
		// Parse only the package clause
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return fi.Name() == e.Name()
		}, parser.PackageClauseOnly)
		if err != nil {
			return "", err
		}
		// If there's exactly one package, return it
		for pkgName := range pkgs {
			return pkgName, nil
		}
	}
	return "", errors.New("no Go files or no package clauses found in directory")
}

// -----------------------------------------------------------------------------
// Data Structures for Collected Declarations (+ doc comments)
// -----------------------------------------------------------------------------

type constItem struct {
	Name string
	Doc  string
}

type varItem struct {
	Name string
	Doc  string
}

type typeItem struct {
	Name string
	Doc  string
}

// funcSpec holds info about an exported function (including doc).
type funcSpec struct {
	Name            string
	Doc             string
	TypeParams      []typeParam
	Params          []param
	Results         []param
	HasNamedResults bool
}

// Generics: type parameters for e.g. func Foo[T any, U comparable](...)
type typeParam struct {
	Name       string
	Constraint string
}

// param: represents a function parameter or result
type param struct {
	Name       string
	Type       string
	IsVariadic bool
}

// -----------------------------------------------------------------------------
// Helper: parseDocs
// -----------------------------------------------------------------------------

func parseDocs(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var lines []string
	for _, c := range cg.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

// -----------------------------------------------------------------------------
// Collecting exported declarations from the package AST
// -----------------------------------------------------------------------------

func collectExported(pkg *packages.Package) (
	types []typeItem,
	consts []constItem,
	vars []varItem,
	funcs []funcSpec,
) {
	seenTypes := make(map[string]bool)

	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {

			switch gd := decl.(type) {
			case *ast.GenDecl:
				switch gd.Tok {
				case token.TYPE:
					for _, spec := range gd.Specs {
						ts := spec.(*ast.TypeSpec)
						if ts.Name.IsExported() && !seenTypes[ts.Name.Name] {
							doc := parseDocs(ts.Doc)
							if doc == "" {
								doc = parseDocs(gd.Doc)
							}
							types = append(types, typeItem{
								Name: ts.Name.Name,
								Doc:  doc,
							})
							seenTypes[ts.Name.Name] = true
						}
					}
				case token.CONST:
					for _, spec := range gd.Specs {
						vs := spec.(*ast.ValueSpec)
						specDoc := parseDocs(vs.Doc)
						if specDoc == "" {
							specDoc = parseDocs(gd.Doc)
						}
						for _, name := range vs.Names {
							if name.IsExported() {
								consts = append(consts, constItem{
									Name: name.Name,
									Doc:  specDoc,
								})
							}
						}
					}
				case token.VAR:
					for _, spec := range gd.Specs {
						vs := spec.(*ast.ValueSpec)
						specDoc := parseDocs(vs.Doc)
						if specDoc == "" {
							specDoc = parseDocs(gd.Doc)
						}
						for _, name := range vs.Names {
							if name.IsExported() {
								vars = append(vars, varItem{
									Name: name.Name,
									Doc:  specDoc,
								})
							}
						}
					}
				}
			case *ast.FuncDecl:
				if gd.Recv == nil && gd.Name.IsExported() {
					fn := parseFuncDecl(gd)
					funcs = append(funcs, fn)
				}
			}
		}
	}
	return
}

func parseFuncDecl(fd *ast.FuncDecl) funcSpec {
	var fs funcSpec
	fs.Name = fd.Name.Name
	fs.Doc = parseDocs(fd.Doc)

	// Type parameters (Go 1.18+ generics)
	if fd.Type.TypeParams != nil {
		for _, tparam := range fd.Type.TypeParams.List {
			constraint := exprToString(tparam.Type)
			for _, name := range tparam.Names {
				fs.TypeParams = append(fs.TypeParams, typeParam{
					Name:       name.Name,
					Constraint: constraint,
				})
			}
		}
	}

	// Function parameters
	if fd.Type.Params != nil {
		for i, field := range fd.Type.Params.List {
			isVariadic := false
			if _, ok := field.Type.(*ast.Ellipsis); ok {
				isVariadic = true
			}

			if len(field.Names) > 0 {
				for _, nameIdent := range field.Names {
					fs.Params = append(fs.Params, param{
						Name:       nameIdent.Name,
						Type:       exprToString(field.Type),
						IsVariadic: isVariadic,
					})
				}
			} else {
				fs.Params = append(fs.Params, param{
					Name:       "",
					Type:       exprToString(field.Type),
					IsVariadic: isVariadic,
				})
			}
			_ = i
		}
	}

	// Function results
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		for _, field := range fd.Type.Results.List {
			typeStr := exprToString(field.Type)
			if len(field.Names) == 0 {
				fs.Results = append(fs.Results, param{
					Name: "",
					Type: typeStr,
				})
			} else {
				fs.HasNamedResults = true
				for _, nameIdent := range field.Names {
					fs.Results = append(fs.Results, param{
						Name: nameIdent.Name,
						Type: typeStr,
					})
				}
			}
		}
	}
	return fs
}

// -----------------------------------------------------------------------------
// exprToString
// -----------------------------------------------------------------------------

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[" + exprToString(e.Len) + "]" + exprToString(e.Elt)
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.ChanType:
		switch e.Dir {
		case ast.SEND:
			return "chan<- " + exprToString(e.Value)
		case ast.RECV:
			return "<-chan " + exprToString(e.Value)
		default:
			return "chan " + exprToString(e.Value)
		}
	case *ast.ParenExpr:
		return "(" + exprToString(e.X) + ")"
	case *ast.IndexExpr:
		return exprToString(e.X) + "[" + exprToString(e.Index) + "]"
	case *ast.IndexListExpr:
		indices := make([]string, 0, len(e.Indices))
		for _, idx := range e.Indices {
			indices = append(indices, exprToString(idx))
		}
		return exprToString(e.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.BasicLit:
		return e.Value
	default:
		return fmt.Sprintf("%#v", expr)
	}
}

// -----------------------------------------------------------------------------
// Generating the re-export file
// -----------------------------------------------------------------------------

func generateReexports(
	outputPackage string,
	importPath string,
	types []typeItem,
	consts []constItem,
	vars []varItem,
	funcs []funcSpec,
) string {
	var sb strings.Builder

	// Use the package name we deduce from the output dir
	sb.WriteString("// Code generated by reexport-gen; DO NOT EDIT.\n\n")
	sb.WriteString(fmt.Sprintf("package %s\n\n", outputPackage))

	sb.WriteString(fmt.Sprintf("import original %q\n\n", importPath))

	// Consolidate *all* consts in ONE block
	if len(consts) > 0 {
		sb.WriteString("const (\n")
		for _, c := range consts {
			printDoc(&sb, c.Doc)
			sb.WriteString(fmt.Sprintf("    %s = original.%s\n\n", c.Name, c.Name))
		}
		sb.WriteString(")\n\n")
	}

	// Consolidate *all* vars in ONE block
	if len(vars) > 0 {
		sb.WriteString("var (\n")
		for _, v := range vars {
			printDoc(&sb, v.Doc)
			sb.WriteString(fmt.Sprintf("    %s = original.%s\n\n", v.Name, v.Name))
		}
		sb.WriteString(")\n\n")
	}

	// Types
	for _, t := range types {
		printDoc(&sb, t.Doc)
		sb.WriteString(fmt.Sprintf("type %s = original.%s\n\n", t.Name, t.Name))
	}

	// Functions
	for _, fn := range funcs {
		printDoc(&sb, fn.Doc)
		sb.WriteString(generateFuncWrapper(fn))
		sb.WriteString("\n")
	}

	return sb.String()
}

// printDoc prints doc lines as `// text`.
func printDoc(sb *strings.Builder, doc string) {
	if doc == "" {
		return
	}
	lines := strings.Split(doc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sb.WriteString("// " + line + "\n")
		} else {
			sb.WriteString("//\n")
		}
	}
}

// generateFuncWrapper re-exports a function.
func generateFuncWrapper(fn funcSpec) string {
	var sb strings.Builder

	sb.WriteString("func ")
	sb.WriteString(fn.Name)

	if len(fn.TypeParams) > 0 {
		var tparams []string
		for _, tp := range fn.TypeParams {
			tparams = append(tparams, fmt.Sprintf("%s %s", tp.Name, tp.Constraint))
		}
		sb.WriteString(fmt.Sprintf("[%s]", strings.Join(tparams, ", ")))
	}

	// Build parameters
	sb.WriteString("(")
	var paramList []string
	unnamedCount := 0
	for i, p := range fn.Params {
		paramName := p.Name
		if paramName == "" {
			paramName = fmt.Sprintf("_p%d", unnamedCount)
			unnamedCount++
		}
		if p.IsVariadic {
			baseType := strings.TrimPrefix(p.Type, "...")
			paramList = append(paramList, fmt.Sprintf("%s ...%s", paramName, baseType))
		} else {
			paramList = append(paramList, fmt.Sprintf("%s %s", paramName, p.Type))
		}
		_ = i
	}
	sb.WriteString(strings.Join(paramList, ", "))
	sb.WriteString(")")

	// Build results
	if len(fn.Results) == 1 && !fn.HasNamedResults && fn.Results[0].Name == "" {
		sb.WriteString(" ")
		sb.WriteString(fn.Results[0].Type)
	} else if len(fn.Results) > 0 {
		var results []string
		for _, r := range fn.Results {
			if r.Name == "" {
				results = append(results, r.Type)
			} else {
				results = append(results, fmt.Sprintf("%s %s", r.Name, r.Type))
			}
		}
		if len(results) == 1 {
			sb.WriteString(" (" + results[0] + ")")
		} else {
			sb.WriteString(" (" + strings.Join(results, ", ") + ")")
		}
	}

	sb.WriteString(" {\n")

	// Original call
	call := fmt.Sprintf("original.%s", fn.Name)
	if len(fn.TypeParams) > 0 {
		var tparamNames []string
		for _, tp := range fn.TypeParams {
			tparamNames = append(tparamNames, tp.Name)
		}
		call += "[" + strings.Join(tparamNames, ", ") + "]"
	}

	// Build arguments
	var callArgs []string
	unnamedCount = 0
	for i, p := range fn.Params {
		argName := p.Name
		if argName == "" {
			argName = fmt.Sprintf("_p%d", unnamedCount)
			unnamedCount++
		}
		if p.IsVariadic && i == len(fn.Params)-1 {
			argName += "..."
		}
		callArgs = append(callArgs, argName)
	}

	if len(fn.Results) > 0 {
		sb.WriteString(fmt.Sprintf("    return %s(%s)\n", call, strings.Join(callArgs, ", ")))
	} else {
		sb.WriteString(fmt.Sprintf("    %s(%s)\n", call, strings.Join(callArgs, ", ")))
	}
	sb.WriteString("}\n")

	return sb.String()
}
