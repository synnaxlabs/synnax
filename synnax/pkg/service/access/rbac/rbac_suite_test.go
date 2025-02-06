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
	"context"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

var (
	ctx                  = context.Background()
	userID               = user.OntologyID(uuid.New())
	changePasswordPolicy = rbac.Policy{
		Subjects: []ontology.ID{userID},
		Objects:  []ontology.ID{userID},
		Actions:  []access.Action{"changePassword"},
	}
)

func TestRBAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Suite")
}
