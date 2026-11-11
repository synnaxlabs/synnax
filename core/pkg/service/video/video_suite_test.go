// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestVideo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Video Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	groupSvc   *group.Service
	proj       project.Project
	projectSvc *project.Service
	svc        *video.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{DB: db}))
	svc = MustOpen(video.OpenService(ctx, video.ServiceConfig{DB: db}))
	proj.Name = "test-project"
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
})

// dispatch runs Writer.Apply in a transaction of its own.
func dispatch(
	ctx context.Context,
	key video.Key,
	dispatchKey string,
	acts []video.Action,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := svc.NewWriter(tx).
			WithDispatchKey(dispatchKey).
			Apply(ctx, key, acts...)
		return err
	})
}

// create creates v in the test project, in a transaction of its own.
func create(ctx context.Context, v video.Video) video.Video {
	GinkgoHelper()
	v.Project = &proj.Key
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).Create(ctx, &v)
	})).To(Succeed())
	return v
}

func retrieve(ctx context.Context, key video.Key) video.Video {
	GinkgoHelper()
	var res video.Video
	Expect(svc.NewRetrieve().Where(video.MatchKeys(key)).Entry(&res).Exec(ctx, db)).
		To(Succeed())
	return res
}
