// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("RBAC Enforcement", Ordered, func() {
	var (
		db        *gorp.DB
		otg       *ontology.Ontology
		svc       *rbac.Service
		userSvc   *user.Service
		groupSvc  *group.Service
		tx        gorp.Tx
		testUser  user.User
		adminUser user.User
		userID    ontology.ID
		adminID   ontology.ID
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		groupSvc = MustSucceed(group.OpenService(ctx, group.Config{
			DB:       db,
			Ontology: otg,
		}))
		userSvc = MustSucceed(user.NewService(ctx, user.Config{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
		}))
		svc = MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	BeforeEach(func() {
		tx = db.OpenTx()
		userWriter := userSvc.NewWriter(tx)
		testUser = user.User{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
		}
		Expect(userWriter.Create(ctx, &testUser)).To(Succeed())
		adminUser = user.User{
			Username:  "adminuser",
			FirstName: "Admin",
			LastName:  "User",
		}
		Expect(userWriter.Create(ctx, &adminUser)).To(Succeed())
		userID = user.OntologyID(testUser.Key)
		adminID = user.OntologyID(adminUser.Key)
	})

	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	AfterAll(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(groupSvc.Close()).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	Describe("Default Deny", func() {
		It("Should deny access when no policies exist", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: userID,
				Objects: []ontology.ID{{Type: "channel", Key: "1"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(Equal(access.Denied))
		})

		It("Should deny access when user has no roles", func() {
			// ActionCreate a policy but don't assign it to anyone
			pol := &policy.Policy{
				Name:    "Unused Policy",
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{{Type: "channel", Key: ""}},
				Actions: []access.Action{access.All},
			}
			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, pol)).To(Succeed())

			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: userID,
				Objects: []ontology.ID{{Type: "channel", Key: "1"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})

	Describe("Instance-Level Permissions", func() {
		var (
			channelPolicy policy.Policy
			viewerRole    role.Role
		)

		BeforeEach(func() {
			// ActionCreate a policy for a specific channel
			channelPolicy = policy.Policy{
				Name:   "Specific Channel Read",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: "channel-123"},
				},
				Actions: []access.Action{access.ActionRetrieve},
			}
			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, &channelPolicy)).To(Succeed())

			// ActionCreate a role with this policy
			viewerRole = role.Role{
				Name:        "Channel Viewer",
				Description: "Can view specific channel",
				Policies:    []uuid.UUID{channelPolicy.Key},
			}
			roleWriter := svc.Role.NewWriter(tx)
			Expect(roleWriter.Create(ctx, &viewerRole)).To(Succeed())

			// Assign role to user
			Expect(roleWriter.AssignRole(ctx, userID, viewerRole.Key)).To(Succeed())
		})

		It("Should allow access to the specific channel", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: userID,
				Objects: []ontology.ID{{Type: "channel", Key: "channel-123"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should deny access to a different channel", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: userID,
				Objects: []ontology.ID{{Type: "channel", Key: "channel-456"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(Equal(access.Denied))
		})

		It("Should deny access with wrong action", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: userID,
				Objects: []ontology.ID{{Type: "channel", Key: "channel-123"}},
				Action:  access.ActionDelete,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})

	Describe("Type-Level Permissions", func() {
		var (
			typePolicy policy.Policy
			adminRole  role.Role
		)

		BeforeEach(func() {
			// ActionCreate a policy for all channels (type-level)
			typePolicy = policy.Policy{
				Name:   "All Channels Access",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: ""}, // Empty key = type-level
				},
				Actions: []access.Action{access.All},
			}
			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, &typePolicy)).To(Succeed())

			// ActionCreate admin role with this policy
			adminRole = role.Role{
				Name:        "Channel Admin",
				Description: "Can do anything with channels",
				Policies:    []uuid.UUID{typePolicy.Key},
			}
			roleWriter := svc.Role.NewWriter(tx)
			Expect(roleWriter.Create(ctx, &adminRole)).To(Succeed())

			// Assign role to admin user
			Expect(roleWriter.AssignRole(ctx, adminID, adminRole.Key)).To(Succeed())
		})

		It("Should allow access to any channel with ActionRetrieve", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: adminID,
				Objects: []ontology.ID{{Type: "channel", Key: "any-channel"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow access to any channel with ActionCreate", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: adminID,
				Objects: []ontology.ID{{Type: "channel", Key: "new-channel"}},
				Action:  access.ActionCreate,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow access to any channel with ActionDelete", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: adminID,
				Objects: []ontology.ID{{Type: "channel", Key: uuid.New().String()}},
				Action:  access.ActionDelete,
			})
			Expect(err).To(BeNil())
		})

		It("Should deny access to different resource type", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: adminID,
				Objects: []ontology.ID{{Type: "workspace", Key: "workspace-1"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})

	Describe("Multiple Objects", func() {
		var (
			multiPolicy policy.Policy
			multiRole   role.Role
			testUserID  ontology.ID
		)

		BeforeEach(func() {
			// ActionCreate a real user for this test
			testUserForMulti := user.User{
				Username:  "multiobjectuser",
				FirstName: "Multi",
				LastName:  "User",
			}
			userWriter := userSvc.NewWriter(tx)
			Expect(userWriter.Create(ctx, &testUserForMulti)).To(Succeed())
			testUserID = user.OntologyID(testUserForMulti.Key)

			// ActionCreate a policy with multiple specific objects
			multiPolicy = policy.Policy{
				Name:   "Multi Object Policy",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch-1"},
					{Type: "channel", Key: "ch-2"},
					{Type: "workspace", Key: "ws-1"},
				},
				Actions: []access.Action{access.ActionRetrieve},
			}
			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, &multiPolicy)).To(Succeed())

			multiRole = role.Role{
				Name:     "Multi Object Role",
				Policies: []uuid.UUID{multiPolicy.Key},
			}
			roleWriter := svc.Role.NewWriter(tx)
			Expect(roleWriter.Create(ctx, &multiRole)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, testUserID, multiRole.Key)).To(Succeed())
		})

		It("Should allow access when requesting one allowed object", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: testUserID,
				Objects: []ontology.ID{{Type: "channel", Key: "ch-1"}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow access when requesting multiple allowed objects", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: testUserID,
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch-1"},
					{Type: "channel", Key: "ch-2"},
				},
				Action: access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow access to mixed types if all are allowed", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: testUserID,
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch-1"},
					{Type: "workspace", Key: "ws-1"},
				},
				Action: access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should deny access if even one object is not allowed", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: testUserID,
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch-1"},
					{Type: "channel", Key: "ch-3"}, // Not in policy
				},
				Action: access.ActionRetrieve,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})

	Describe("Multiple Roles and Policies", func() {
		var (
			readPolicy  policy.Policy
			writePolicy policy.Policy
			readRole    role.Role
			writeRole   role.Role
			complexUser ontology.ID
		)

		BeforeEach(func() {
			// ActionCreate a real user for this test
			complexUserEntity := user.User{
				Username:  "complexuser",
				FirstName: "Complex",
				LastName:  "User",
			}
			userWriter := userSvc.NewWriter(tx)
			Expect(userWriter.Create(ctx, &complexUserEntity)).To(Succeed())
			complexUser = user.OntologyID(complexUserEntity.Key)

			// ActionCreate read policy
			readPolicy = policy.Policy{
				Name:   "Read All Channels",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: ""},
				},
				Actions: []access.Action{access.ActionRetrieve},
			}

			// ActionCreate write policy
			writePolicy = policy.Policy{
				Name:   "Write Specific Workspace",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "workspace", Key: "ws-special"},
				},
				Actions: []access.Action{access.ActionCreate, access.ActionUpdate},
			}

			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, &readPolicy)).To(Succeed())
			Expect(policyWriter.Create(ctx, &writePolicy)).To(Succeed())

			// ActionCreate two separate roles
			readRole = role.Role{
				Name:     "Reader",
				Policies: []uuid.UUID{readPolicy.Key},
			}
			writeRole = role.Role{
				Name:     "Writer",
				Policies: []uuid.UUID{writePolicy.Key},
			}

			roleWriter := svc.Role.NewWriter(tx)
			Expect(roleWriter.Create(ctx, &readRole)).To(Succeed())
			Expect(roleWriter.Create(ctx, &writeRole)).To(Succeed())

			// Assign both roles to user
			Expect(roleWriter.AssignRole(ctx, complexUser, readRole.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, complexUser, writeRole.Key)).To(Succeed())
		})

		It("Should allow reading any channel from first role", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: complexUser,
				Objects: []ontology.ID{{Type: "channel", Key: uuid.New().String()}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow writing to specific workspace from second role", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: complexUser,
				Objects: []ontology.ID{{Type: "workspace", Key: "ws-special"}},
				Action:  access.ActionUpdate,
			})
			Expect(err).To(BeNil())
		})

		It("Should deny writing to channels (no permission)", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: complexUser,
				Objects: []ontology.ID{{Type: "channel", Key: "ch-1"}},
				Action:  access.ActionCreate,
			})
			Expect(err).To(Equal(access.Denied))
		})

		It("Should deny writing to different workspace", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: complexUser,
				Objects: []ontology.ID{{Type: "workspace", Key: "ws-other"}},
				Action:  access.ActionUpdate,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})

	Describe("Combined Type and Instance Permissions", func() {
		var (
			combinedUser   ontology.ID
			typeReadPolicy policy.Policy
			instancePolicy policy.Policy
			combinedRole   role.Role
		)

		BeforeEach(func() {
			// ActionCreate a real user for this test
			combinedUserEntity := user.User{
				Username:  "combineduser",
				FirstName: "Combined",
				LastName:  "User",
			}
			userWriter := userSvc.NewWriter(tx)
			Expect(userWriter.Create(ctx, &combinedUserEntity)).To(Succeed())
			combinedUser = user.OntologyID(combinedUserEntity.Key)

			// Type-level read for channels
			typeReadPolicy = policy.Policy{
				Name:   "Read All Channels Type",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: ""},
				},
				Actions: []access.Action{access.ActionRetrieve},
			}

			// Instance-level write for specific channel
			instancePolicy = policy.Policy{
				Name:   "Write Specific Channel",
				Effect: policy.EffectAllow,
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch-editable"},
				},
				Actions: []access.Action{access.ActionUpdate},
			}

			policyWriter := svc.Policy.NewWriter(tx)
			Expect(policyWriter.Create(ctx, &typeReadPolicy)).To(Succeed())
			Expect(policyWriter.Create(ctx, &instancePolicy)).To(Succeed())

			// Single role with both policies
			combinedRole = role.Role{
				Name: "Combined Permissions",
				Policies: []uuid.UUID{
					typeReadPolicy.Key,
					instancePolicy.Key,
				},
			}

			roleWriter := svc.Role.NewWriter(tx)
			Expect(roleWriter.Create(ctx, &combinedRole)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, combinedUser, combinedRole.Key)).To(Succeed())
		})

		It("Should allow reading any channel via type-level permission", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: combinedUser,
				Objects: []ontology.ID{{Type: "channel", Key: uuid.New().String()}},
				Action:  access.ActionRetrieve,
			})
			Expect(err).To(BeNil())
		})

		It("Should allow updating specific channel via instance permission", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: combinedUser,
				Objects: []ontology.ID{{Type: "channel", Key: "ch-editable"}},
				Action:  access.ActionUpdate,
			})
			Expect(err).To(BeNil())
		})

		It("Should deny updating other channels (only have read type permission)", func() {
			err := svc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: combinedUser,
				Objects: []ontology.ID{{Type: "channel", Key: "ch-readonly"}},
				Action:  access.ActionUpdate,
			})
			Expect(err).To(Equal(access.Denied))
		})
	})
})
