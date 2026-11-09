// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Auth Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db *gorp.DB
	// users stores the users that hold the credentials and sessions of the specs.
	users *gorp.Table[user.Key, user.User]
)

var _ = BeforeSuite(func() { ShouldNotLeakGoroutines() })

// Every spec gets its own db so sweeps and clocks never cross.
var _ = BeforeEach(func(ctx SpecContext) {
	db = DeferClose(gorp.Wrap(memkv.New()))
	users = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[user.Key, user.User]{
		DB:   db,
		Name: string(resource.TypeUser),
	}))
})

// subject creates a user and returns its resource ID.
func subject() resource.ID {
	GinkgoHelper()
	u := user.User{Key: uuid.New(), Name: "subject"}
	Expect(users.NewCreate().Entry(&u).Exec(context.Background(), db)).To(Succeed())
	return u.ResourceID()
}

// deleteSubject deletes the user that id names.
func deleteSubject(ctx context.Context, id resource.ID) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(id))
	Expect(users.NewDelete().
		Where(gorp.MatchKeys[user.Key, user.User](key)).
		Exec(ctx, db)).To(Succeed())
}

// hash hashes plaintext with svc.
func hash(svc *password.Service, plaintext string) password.Hash {
	GinkgoHelper()
	return MustSucceed(svc.Hash(plaintext))
}
