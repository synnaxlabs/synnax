// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apiproject "github.com/synnaxlabs/synnax/pkg/api/project"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "unretrievable")
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apiproject.RetrieveRequest{
			Where: &project.Where{Key: &filter.Set[project.Key]{Eq: &p.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Export", func() {
	It(
		"Should export the bundle when retrieve is granted on the project and members",
		func(ctx SpecContext) {
			proj := createProject(ctx, "granted")
			l := createLog(ctx, proj.Key, "Metrics")
			grantRetrieveOn(
				ctx,
				author.ResourceID(),
				project.ResourceID(proj.Key),
				log.ResourceID(l.Key),
			)
			Expect(MustSucceed(apiSvc.Export(
				AuthedCtx(ctx, author),
				apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
			))).To(SatisfyAll(HaveKey("manifest.json"), HaveKey("Metrics.json")))
		},
	)
	It("Should reject an unsupported encoding", func(ctx SpecContext) {
		proj := createProject(ctx, "bad-encoding")
		grantRetrieveOn(ctx, author.ResourceID(), project.ResourceID(proj.Key))
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: "YAML"},
		)).Error().To(MatchError(ContainSubstring("unsupported encoding")))
	})
	It("Should reject the request when retrieve is not granted on the project", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "ungranted-project")
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when retrieve is not granted on a member", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "ungranted-member")
		createLog(ctx, proj.Key, "Metrics")
		grantRetrieveOn(ctx, author.ResourceID(), project.ResourceID(proj.Key))
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var (
	projectType = resource.ID{Type: resource.TypeProject}
	logType     = resource.ID{Type: resource.TypeLog}
)

func newBundle(name string) apiproject.ImportRequest {
	return zip.Files{
		"manifest.json": []byte(
			`{"version":1,"type":"project","name":"` + name + `"}`,
		),
		"Metrics.json": []byte(
			`{"version":2,"type":"log","name":"Metrics","channels":[]}`,
		),
	}
}

func importCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := AuthedCtx(ctx, u)
	fctx.Set("params", `{"file_name":"Imported.zip"}`)
	return fctx
}

var _ = Describe("Import", func() {
	It(
		"Should import the bundle with create on every kind and update on projects",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType, logType)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, projectType)
			res := MustSucceed(importBundle(importCtx(ctx, u), newBundle("Imported")))
			Expect(res.Project.Name).To(Equal("Imported"))
		},
	)
	It("Should reject the request without update on the new project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType, logType)
		Expect(importBundle(importCtx(ctx, u), newBundle("Imported"))).Error().
			To(deniedOnType(u, access.ActionUpdate, resource.TypeProject))
	})
	It("Should reject the request when create is not granted on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, logType)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, projectType)
		Expect(importBundle(importCtx(ctx, u), newBundle("Imported"))).Error().
			To(deniedOnType(u, access.ActionCreate, resource.TypeProject))
	})
	It("Should reject the request when create is granted on the project alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, projectType)
		Expect(importBundle(importCtx(ctx, u), newBundle("Imported"))).Error().
			To(deniedOnType(u, access.ActionCreate, resource.TypeLog))
	})
	Context("With a member in a folder", func() {
		groupType := resource.ID{Type: resource.TypeGroup}
		var u user.User
		BeforeEach(func(ctx SpecContext) {
			u = createUser(ctx)
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionCreate,
				projectType,
				logType,
				groupType,
			)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, projectType)
		})
		bundle := func() apiproject.ImportRequest {
			files := newBundle("Foldered")
			files["Folder/Metrics.json"] = files["Metrics.json"]
			delete(files, "Metrics.json")
			return files
		}
		It("Should reject the request without update on the new group", func(
			ctx SpecContext,
		) {
			Expect(importBundle(importCtx(ctx, u), bundle())).Error().
				To(deniedOnType(u, access.ActionUpdate, resource.TypeGroup))
		})
		It("Should import the member into the group with update on groups", func(
			ctx SpecContext,
		) {
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, groupType)
			res := MustSucceed(importBundle(importCtx(ctx, u), bundle()))
			Expect(res.Project.Name).To(Equal("Foldered"))
		})
	})
	It("Should reject a request with no params", func(ctx SpecContext) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType, logType)
		Expect(importBundle(AuthedCtx(ctx, u), newBundle("Imported"))).Error().
			To(MatchError(ContainSubstring("params")))
	})
	It("Should reject an invalid bundle before any access check", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		files := zip.Files{"Metrics.json": []byte("{}")}
		Expect(importBundle(importCtx(ctx, u), files)).Error().
			To(MatchError(ContainSubstring("bundle holds no manifest.json")))
	})
})

var _ = Describe("Create", func() {
	It("Should refuse the create without create on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := project.Project{Key: uuid.New(), Name: "Denied"}
		Expect(create(AuthedCtx(ctx, u), apiproject.CreateRequest{
			Projects: []project.Project{p},
		})).Error().To(denied(u, access.ActionCreate, p.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeFalse())
	})
	It("Should create the project with create on it", func(ctx SpecContext) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType)
		res := MustSucceed(create(AuthedCtx(ctx, u), apiproject.CreateRequest{
			Projects: []project.Project{{Name: "Granted"}},
		}))
		Expect(retrieveProject(ctx, res.Projects[0].Key).Name).To(Equal("Granted"))
	})
})

var _ = Describe("Rename", func() {
	It("Should refuse the rename without update on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "Rename denied")
		Expect(rename(AuthedCtx(ctx, u), apiproject.RenameRequest{
			Key:  p.Key,
			Name: "Renamed",
		})).Error().To(denied(u, access.ActionUpdate, p.ResourceID()))
		Expect(retrieveProject(ctx, p.Key).Name).To(Equal("Rename denied"))
	})
	It("Should rename the project with update on it", func(ctx SpecContext) {
		u := createUser(ctx)
		p := createProject(ctx, "Rename granted")
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
		MustSucceed(rename(AuthedCtx(ctx, u), apiproject.RenameRequest{
			Key:  p.Key,
			Name: "Renamed",
		}))
		Expect(retrieveProject(ctx, p.Key).Name).To(Equal("Renamed"))
	})
})

var _ = Describe("Delete", func() {
	var (
		u user.User
		p project.Project
		l log.Log
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		p = createProject(ctx, "Deleted")
		l = createLog(ctx, p.Key, "Cascaded")
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apiproject.DeleteRequest{
			Keys: []project.Key{p.Key},
		})
	}
	It("Should refuse the delete without delete on the project", func(
		ctx SpecContext,
	) {
		grantOn(ctx, u.ResourceID(), access.ActionDelete, l.ResourceID())
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
		Expect(deleteAs(ctx)).Error().
			To(denied(u, access.ActionDelete, p.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeTrue())
	})
	It("Should refuse the delete without delete on a log it cascades to", func(
		ctx SpecContext,
	) {
		grantOn(ctx, u.ResourceID(), access.ActionDelete, p.ResourceID())
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
		Expect(deleteAs(ctx)).Error().
			To(denied(u, access.ActionDelete, l.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeTrue())
	})
	It("Should refuse the delete without update on the project it empties", func(
		ctx SpecContext,
	) {
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionDelete,
			p.ResourceID(),
			l.ResourceID(),
		)
		Expect(deleteAs(ctx)).Error().
			To(denied(u, access.ActionUpdate, p.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeTrue())
	})
	It(
		"Should delete the project and its log with delete on both",
		func(ctx SpecContext) {
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionDelete,
				p.ResourceID(),
				l.ResourceID(),
			)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
			MustSucceed(deleteAs(ctx))
			Expect(projectExists(ctx, p.Key)).To(BeFalse())
			Expect(logSvc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Exists(ctx, nil)).To(BeFalse())
		},
	)
})

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// deniedOnType matches an error that denies u the action on a record of type t.
func deniedOnType(
	u user.User,
	action access.Action,
	t resource.Type,
) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring("%s cannot %s %s:", u.ResourceID(), action, t)),
	)
}

func projectExists(ctx SpecContext, key project.Key) bool {
	GinkgoHelper()
	return MustSucceed(projectSvc.NewRetrieve().
		Where(project.MatchKeys(key)).
		Exists(ctx, nil))
}

// createProjectGroup creates a project group with no top parent.
func createProjectGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeProject}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

func retrieveProject(ctx SpecContext, key project.Key) project.Project {
	GinkgoHelper()
	var p project.Project
	Expect(projectSvc.NewRetrieve().
		Where(project.MatchKeys(key)).
		Entry(&p).
		Exec(ctx, nil)).To(Succeed())
	return p
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			p := createProject(ctx, "Set group denied")
			g := createProjectGroup(ctx, "Set group denied")
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apiproject.SetGroupRequest{Group: &g.Key, Keys: []project.Key{p.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveProject(ctx, p.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the project with update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			p := createProject(ctx, "Set group granted")
			g := createProjectGroup(ctx, "Set group granted")
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionUpdate,
				p.ResourceID(),
				g.ResourceID(),
			)
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apiproject.SetGroupRequest{Group: &g.Key, Keys: []project.Key{p.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveProject(ctx, p.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	projectType := resource.ID{Type: resource.TypeProject}
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createProjectGroup(ctx, "Create denied")
			grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType)
			Expect(create(
				AuthedCtx(ctx, u),
				apiproject.CreateRequest{
					Projects: []project.Project{{Name: "Denied", Group: &g.Key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the project with update on the group", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createProjectGroup(ctx, "Create granted")
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, g.ResourceID())
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apiproject.CreateRequest{
				Projects: []project.Project{{Name: "Granted", Group: &g.Key}},
			},
		))
		Expect(retrieveProject(ctx, res.Projects[0].Key).Group).To(Equal(&g.Key))
	})
})
