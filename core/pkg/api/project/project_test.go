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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Retrieve", func() {
	It("Should leave out the projects that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createProject(ctx, "leave-out-granted")
		createProject(ctx, "leave-out-denied")
		grantRetrieveOn(ctx, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiproject.RetrieveRequest{}))
		Expect(res.Projects).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "unretrievable")
		Expect(retrieve(AuthedCtx(ctx, u), apiproject.RetrieveRequest{
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
			Expect(MustSucceed(export(
				AuthedCtx(ctx, author),
				apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
			))).To(SatisfyAll(HaveKey("manifest.json"), HaveKey("Metrics.json")))
		},
	)
	It("Should reject an unsupported encoding", func(ctx SpecContext) {
		proj := createProject(ctx, "bad-encoding")
		grantRetrieveOn(ctx, author.ResourceID(), project.ResourceID(proj.Key))
		Expect(export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: "YAML"},
		)).Error().To(MatchError(ContainSubstring("unsupported encoding")))
	})
	It("Should reject the request when retrieve is not granted on the project", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "ungranted-project")
		Expect(export(
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
		Expect(export(
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
	It("Should import the bundle with create on every kind alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType, logType)
		res := MustSucceed(importBundle(importCtx(ctx, u), newBundle("Imported")))
		Expect(res.Project.Name).To(Equal("Imported"))
	})
	It("Should reject the request when create is not granted on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, logType)
		Expect(importBundle(importCtx(ctx, u), newBundle("Imported"))).Error().
			To(deniedOnType(u, access.ActionCreate, resource.TypeProject))
	})
	It("Should reject the request when create is granted on the project alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, projectType)
		Expect(importBundle(importCtx(ctx, u), newBundle("Imported"))).Error().
			To(deniedOnType(u, access.ActionCreate, resource.TypeLog))
	})
	It("Should import a member into a new folder with create on every kind", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		groupType := resource.ID{Type: resource.TypeGroup}
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionCreate,
			projectType,
			logType,
			groupType,
		)
		files := newBundle("Foldered")
		files["Folder/Metrics.json"] = files["Metrics.json"]
		delete(files, "Metrics.json")
		res := MustSucceed(importBundle(importCtx(ctx, u), files))
		Expect(res.Project.Name).To(Equal("Foldered"))
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
	It("Should decode the bundle before it waits for the write lock", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		files := zip.Files{"Metrics.json": []byte("{}")}
		DeferCleanup(db.OpenTx().Close)
		errs := make(chan error, 1)
		go func() {
			_, err := importBundle(importCtx(ctx, u), files)
			errs <- err
		}()
		Eventually(errs).Should(Receive(MatchError(validate.ErrValidation)))
	})
	It("Should migrate each member before it waits for the write lock", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		files := newBundle("Too new")
		files["Metrics.json"] = []byte(
			`{"version":99,"type":"log","name":"Metrics","channels":[]}`,
		)
		DeferCleanup(db.OpenTx().Close)
		errs := make(chan error, 1)
		go func() {
			_, err := importBundle(importCtx(ctx, u), files)
			errs <- err
		}()
		Eventually(errs).Should(Receive(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("newer than this Core supports")),
		)))
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

var _ = Describe("Dispatch", func() {
	setGroup := func(g group.Key) []project.Action {
		return []project.Action{project.NewSetAction(project.SetPayload{
			Path:  []string{"group"},
			Value: g,
		})}
	}
	It("Should refuse the dispatch without update on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "Dispatch denied")
		g := createProjectGroup(ctx, "Dispatch denied")
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, g.ResourceID())
		Expect(dispatch(AuthedCtx(ctx, u), apiproject.DispatchRequest{
			Key:     p.Key,
			Actions: setGroup(g.Key),
		})).Error().To(denied(u, access.ActionUpdate, p.ResourceID()))
		Expect(retrieveProject(ctx, p.Key).Group).To(BeNil())
	})
	It("Should refuse a group change without update on the group", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "Dispatch group denied")
		g := createProjectGroup(ctx, "Dispatch group denied")
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, p.ResourceID())
		Expect(dispatch(AuthedCtx(ctx, u), apiproject.DispatchRequest{
			Key:     p.Key,
			Actions: setGroup(g.Key),
		})).Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
		Expect(retrieveProject(ctx, p.Key).Group).To(BeNil())
	})
	It("Should apply the actions with update on the project and the group", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := createProject(ctx, "Dispatch granted")
		g := createProjectGroup(ctx, "Dispatch granted")
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionUpdate,
			p.ResourceID(),
			g.ResourceID(),
		)
		MustSucceed(dispatch(AuthedCtx(ctx, u), apiproject.DispatchRequest{
			Key:     p.Key,
			Actions: setGroup(g.Key),
		}))
		Expect(retrieveProject(ctx, p.Key).Group).To(Equal(&g.Key))
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
		Expect(deleteAs(ctx)).Error().
			To(denied(u, access.ActionDelete, p.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeTrue())
	})
	It("Should refuse the delete without delete on a log it cascades to", func(
		ctx SpecContext,
	) {
		grantOn(ctx, u.ResourceID(), access.ActionDelete, p.ResourceID())
		Expect(deleteAs(ctx)).Error().
			To(denied(u, access.ActionDelete, l.ResourceID()))
		Expect(projectExists(ctx, p.Key)).To(BeTrue())
	})
	It(
		"Should delete the project and its log with delete on both alone",
		func(ctx SpecContext) {
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionDelete,
				p.ResourceID(),
				l.ResourceID(),
			)
			MustSucceed(deleteAs(ctx))
			Expect(projectExists(ctx, p.Key)).To(BeFalse())
			Expect(logSvc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Exists(ctx, db)).To(BeFalse())
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
		Exists(ctx, db))
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
		Exec(ctx, db)).To(Succeed())
	return p
}

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
