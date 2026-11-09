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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

const (
	// actionsDomain marks a struct or union that takes the standard actions.
	actionsDomain = "actions"
	// patchNamespace holds the path payloads of the standard actions.
	patchNamespace = "patch"
	// createAction is the standard action that only the Core emits.
	createAction = "Create"
)

// pathActions are the standard actions after Create. Each carries the patch struct of
// the same name.
var pathActions = []string{"Set", "Remove", "Insert", "Move"}

// declaresActions reports whether a struct or union of the file is marked @actions,
// which reads the types of the patch schema.
func declaresActions(ast parser.ISchemaContext) bool {
	return lo.SomeBy(ast.AllDefinition(), func(def parser.IDefinitionContext) bool {
		return lo.SomeBy(bodyDomains(def), func(d parser.IDomainContext) bool {
			return d.IDENT().GetText() == actionsDomain
		})
	})
}

func bodyDomains(def parser.IDefinitionContext) []parser.IDomainContext {
	if ud := def.UnionDef(); ud != nil && ud.UnionBody() != nil {
		return ud.UnionBody().AllDomain()
	}
	if full, ok := def.StructDef().(*parser.StructFullContext); ok &&
		full.StructBody() != nil {
		return full.StructBody().AllDomain()
	}
	return nil
}

// definitionName returns the name a struct or union definition declares.
func definitionName(def parser.IDefinitionContext) (string, bool) {
	if ud := def.UnionDef(); ud != nil && len(ud.AllIDENT()) > 0 {
		return ud.AllIDENT()[0].GetText(), true
	}
	if full, ok := def.StructDef().(*parser.StructFullContext); ok {
		return full.IDENT().GetText(), true
	}
	return "", false
}

// synthesizeActions registers the payload types and the Action union of the struct or
// union in the namespace that has actions: the standard ones of @actions, then the ones
// its body declares. A namespace holds one Action union. Version files declare no
// actions, so it does nothing there.
func synthesizeActions(c *analysisCtx) {
	if _, _, inVersionFile := paths.VersionFile(c.filePath); inVersionFile {
		return
	}
	declared := make(map[string][]parser.IActionDefContext)
	for _, def := range c.ast.AllDefinition() {
		if name, ok := definitionName(def); ok {
			declared[c.namespace+"."+name] = actionDefs(def)
		}
	}
	var owner string
	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		mark, marked := typ.Domains[actionsDomain]
		defs := declared[typ.QualifiedName]
		if !marked && len(defs) == 0 {
			continue
		}
		if mark.AST == nil && len(defs) > 0 {
			mark.AST = defs[0]
		}
		if !isRecord(typ) {
			c.report(diagnostics.Errorf(mark.AST,
				"@actions marks a struct or union; %s is neither", typ.Name,
			))
			continue
		}
		if owner != "" {
			c.report(diagnostics.Errorf(mark.AST,
				"%s and %s both have actions; a namespace holds one Action union",
				owner, typ.Name,
			))
			continue
		}
		owner = typ.Name
		synthesizeOwnerActions(c, typ, mark, marked, defs)
	}
}

func isRecord(typ resolution.Type) bool {
	switch typ.Form.(type) {
	case resolution.StructForm, resolution.UnionForm:
		return true
	}
	return false
}

// synthesizeOwnerActions registers the payloads and the Action union of owner, and
// sets the actions of owner.
func synthesizeOwnerActions(
	c *analysisCtx,
	owner resolution.Type,
	mark resolution.Domain,
	marked bool,
	defs []parser.IActionDefContext,
) {
	domains := layDomains(c.fileDomains, owner.Domains, "output", "omit")
	var omitted []string
	if _, ok := domains["pb"]; ok {
		omitted = append(omitted, "pb")
	}
	for name, d := range owner.Domains {
		if _, hand := d.Expressions.Find("hand"); hand {
			omitted = append(omitted, name)
		}
	}
	for _, name := range omitted {
		addDomain(domains, resolution.Domain{
			Name: name, Expressions: resolution.Expressions{{Name: "omit"}},
		})
	}
	noun := strings.ReplaceAll(casing.TypeSnake(owner.Name), "_", " ")
	var (
		actions  []resolution.Action
		payloads []resolution.Type
	)
	add := func(
		node antlr.ParserRuleContext, name string, payload resolution.Type,
	) bool {
		payload.Name = name + "Payload"
		payload.Namespace = c.namespace
		payload.QualifiedName = c.namespace + "." + payload.Name
		payload.FilePath = c.filePath
		if lo.ContainsBy(actions, func(a resolution.Action) bool {
			return a.Name == name
		}) {
			c.report(diagnostics.Errorf(node,
				"action %s of %s is declared twice", name, owner.Name,
			))
			return false
		}
		if _, exists := c.table.Get(payload.QualifiedName); exists {
			c.report(diagnostics.Errorf(node,
				"action %s of %s needs the name %s, which the schema declares",
				name, owner.Name, payload.Name,
			))
			return false
		}
		payloads = append(payloads, payload)
		actions = append(actions, resolution.Action{
			AST:     node,
			Name:    name,
			Payload: resolution.TypeRef{Name: payload.QualifiedName},
		})
		return true
	}
	if marked {
		if !standardActionsValid(c, owner, mark) {
			return
		}
		ownerField := requiredField(
			casing.TypeSnake(owner.Name),
			resolution.TypeRef{Name: owner.QualifiedName},
			"is the created "+noun+".",
		)
		if add(mark.AST, createAction, resolution.Type{
			Form: resolution.StructForm{Fields: []resolution.Field{ownerField}},
			Domains: withDoc(
				domains,
				"replaces the state with the created "+noun+". Only the Core "+
					"emits it.",
			),
		}) {
			actions[len(actions)-1].CoreOnly = true
		}
		for _, name := range pathActions {
			base := resolution.TypeRef{Name: patchNamespace + "." + name}
			payloadDomains := cloneDomains(domains)
			if patch, ok := c.table.Get(base.Name); ok {
				if d, ok := patch.Domains["doc"]; ok {
					payloadDomains["doc"] = d
				}
			}
			add(mark.AST, name, resolution.Type{
				Form:    resolution.DistinctForm{Base: base},
				Domains: payloadDomains,
			})
		}
	}
	for _, def := range defs {
		name := def.IDENT().GetText()
		if marked && (name == createAction || lo.Contains(pathActions, name)) {
			c.report(diagnostics.Errorf(def,
				"action %s of %s: @actions already declares %s",
				name, owner.Name, name,
			))
			continue
		}
		decl := collectAction(c, def)
		add(def, name, resolution.Type{
			Form:    decl.form,
			Domains: mergeDomains(domains, decl.domains),
			AST:     def,
		})
	}
	if len(actions) == 0 {
		return
	}
	union := resolution.Type{
		Name:          "Action",
		Namespace:     c.namespace,
		QualifiedName: c.namespace + ".Action",
		FilePath:      c.filePath,
		Form: resolution.UnionForm{
			Discriminator: "type",
			Adjacent:      true,
			Variants: lo.Map(
				actions,
				func(a resolution.Action, _ int) resolution.UnionVariant {
					return resolution.UnionVariant{
						Name:    casing.TypeSnake(a.Name),
						Type:    a.Payload,
						Domains: map[string]resolution.Domain{},
					}
				},
			),
		},
		Domains: withDoc(domains, "holds one "+noun+" mutation."),
	}
	if _, exists := c.table.Get(union.QualifiedName); exists {
		c.report(diagnostics.Errorf(mark.AST,
			"the actions of %s need the name Action, which the schema declares",
			owner.Name,
		))
		return
	}
	for _, p := range payloads {
		lo.Must0(c.table.Add(p))
	}
	lo.Must0(c.table.Add(union))
	switch form := owner.Form.(type) {
	case resolution.StructForm:
		form.Actions = actions
		owner.Form = form
	case resolution.UnionForm:
		form.Actions = actions
		owner.Form = form
	}
	replaceType(c.table, owner)
}

// standardActionsValid reports whether owner can take the standard actions of mark,
// reporting each reason it cannot.
func standardActionsValid(
	c *analysisCtx, owner resolution.Type, mark resolution.Domain,
) bool {
	valid := true
	for _, e := range mark.Expressions {
		c.report(diagnostics.Errorf(e.AST, "unknown @actions option %q", e.Name))
		valid = false
	}
	if !importsNamespace(c.ast, patchNamespace) {
		c.report(diagnostics.Errorf(mark.AST,
			`@actions requires import "schemas/x/patch"`,
		))
		valid = false
	}
	params := resolution.NonDefaultedTypeParams(resolution.TypeParams(owner.Form))
	if len(params) > 0 {
		c.report(diagnostics.Errorf(mark.AST,
			"@actions needs a default for type parameter %s of %s",
			params[0].Name, owner.Name,
		))
		valid = false
	}
	return valid
}

// actionDecl is the payload of an action that a struct or union body declares.
type actionDecl struct {
	form    resolution.StructForm
	domains map[string]resolution.Domain
}

// collectAction collects the payload of an action as a struct declaration.
func collectAction(c *analysisCtx, def parser.IActionDefContext) actionDecl {
	decl := actionDecl{domains: make(map[string]resolution.Domain)}
	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				decl.form.Extends = append(decl.form.Extends, collectTypeRef(tr, nil))
			}
		}
	}
	body := def.ActionBody()
	if body == nil {
		return decl
	}
	for _, f := range body.AllFieldDef() {
		decl.form.Fields = append(decl.form.Fields, collectField(c, f, nil))
	}
	for _, d := range body.AllDomain() {
		addDomain(decl.domains, collectDomain(c, d))
	}
	return decl
}
