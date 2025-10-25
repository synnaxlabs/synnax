// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

// sets the base permissions that need to exist in the server.
func maybeSetBasePermissions(
	ctx context.Context,
	dist *distribution.Layer,
	svc *service.Layer,
) error {
	return dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
		// base policies that need to be created
		basePolicies := map[ontology.Type]access.Action{
			"label":            access.All,
			"cluster":          access.All,
			"channel":          access.All,
			"node":             access.All,
			"group":            access.All,
			"range":            access.All,
			"range-alias":      access.All,
			"workspace":        access.All,
			"log":              access.All,
			"lineplot":         access.All,
			"rack":             access.All,
			"device":           access.All,
			"task":             access.All,
			"table":            access.All,
			"schematic_symbol": access.All,
			"user":             access.Retrieve,
			"schematic":        access.Retrieve,
			"policy":           access.Retrieve,
			"builtin":          access.Retrieve,
			"framer":           access.All,
			"status":           access.All,
		}
		// for migration purposes, some old base policies that need to be deleted
		oldBasePolicies := map[ontology.Type]access.Action{}

		existingPolicies := make([]rbac.Policy, 0, len(basePolicies))
		policiesToDelete := make([]uuid.UUID, 0, len(oldBasePolicies))
		if err := svc.RBAC.NewRetrieve().WhereSubjects(user.OntologyTypeID).
			Entries(&existingPolicies).Exec(ctx, tx); err != nil {
			return err
		}
		for _, p := range existingPolicies {
			if len(p.Subjects) != 1 || len(p.Objects) != 1 || len(p.Actions) != 1 {
				// then this policy is not one of the policies created in maybeSetBasePermissions
				continue
			}
			s := p.Subjects[0]
			o := p.Objects[0]
			a := p.Actions[0]
			if (s != user.OntologyTypeID) || (o.Key != "") {
				// the policy does not apply to the general user ontology type
				continue
			}
			if basePolicies[o.Type] == a {
				delete(basePolicies, o.Type)
			} else if oldBasePolicies[o.Type] == a {
				policiesToDelete = append(policiesToDelete, p.Key)
			}
		}
		for t := range basePolicies {
			if err := svc.RBAC.NewWriter(tx).Create(ctx, &rbac.Policy{
				Subjects: []ontology.ID{user.OntologyTypeID},
				Objects:  []ontology.ID{{Type: t, Key: ""}},
				Actions:  []access.Action{basePolicies[t]},
			}); err != nil {
				return err
			}
		}
		return svc.RBAC.NewWriter(tx).Delete(ctx, policiesToDelete...)
	})
}

func maybeProvisionRootUser(
	ctx context.Context,
	dist *distribution.Layer,
	svc *service.Layer,
) error {
	creds := auth.InsecureCredentials{
		Username: viper.GetString(usernameFlag),
		Password: password.Raw(viper.GetString(passwordFlag)),
	}
	exists, err := svc.User.UsernameExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	if exists {
		// we potentially need to update the root user flag
		// we want to make sure the root user still has the allow_all policy
		return dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
			// For cluster versions before v0.31.0, the root user flag was not set. We
			// need to set it here.
			if err = svc.User.NewWriter(tx).MaybeSetRootUser(ctx, creds.Username); err != nil {
				return err
			}

			var u user.User
			if err = svc.User.NewRetrieve().WhereUsernames(creds.Username).Entry(&u).Exec(ctx, tx); err != nil {
				return err
			}
			if !u.RootUser {
				return nil
			}
			policies := make([]rbac.Policy, 0, 1)
			if err := svc.RBAC.NewRetrieve().
				WhereSubjects(user.OntologyID(u.Key)).
				Entries(&policies).
				Exec(ctx, tx); err != nil {
				return err
			}
			for _, p := range policies {
				if lo.Contains(p.Objects, rbac.AllowAllOntologyID) {
					return nil
				}
			}
			return svc.RBAC.NewWriter(tx).Create(ctx, &rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(u.Key)},
				Objects:  []ontology.ID{rbac.AllowAllOntologyID},
				Actions:  []access.Action{},
			})
		})
	}

	// register the user first, then give them all permissions
	if err = dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
		if err = svc.Auth.NewWriter(tx).Register(ctx, creds); err != nil {
			return err
		}
		userObj := user.User{Username: creds.Username, RootUser: true}
		if err = svc.User.NewWriter(tx).Create(ctx, &userObj); err != nil {
			return err
		}
		return svc.RBAC.NewWriter(tx).Create(
			ctx,
			&rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(userObj.Key)},
				Objects:  []ontology.ID{rbac.AllowAllOntologyID},
				Actions:  []access.Action{},
			},
		)
	}); err != nil {
		return err
	}
	return maybeSetBasePermissions(ctx, dist, svc)
}
