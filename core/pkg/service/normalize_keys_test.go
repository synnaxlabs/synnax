// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	policyversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions"
	roleversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions"
	arcversions "github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	channelversions "github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	deviceversions "github.com/synnaxlabs/synnax/pkg/service/device/versions"
	groupversions "github.com/synnaxlabs/synnax/pkg/service/group/versions"
	labelversions "github.com/synnaxlabs/synnax/pkg/service/label/versions"
	lineplotversions "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	logversions "github.com/synnaxlabs/synnax/pkg/service/log/versions"
	ontologyversions "github.com/synnaxlabs/synnax/pkg/service/ontology/versions"
	projectversions "github.com/synnaxlabs/synnax/pkg/service/project/versions"
	rackversions "github.com/synnaxlabs/synnax/pkg/service/rack/versions"
	aliasversions "github.com/synnaxlabs/synnax/pkg/service/ranger/alias/versions"
	kvversions "github.com/synnaxlabs/synnax/pkg/service/ranger/kv/versions"
	rangerversions "github.com/synnaxlabs/synnax/pkg/service/ranger/versions"
	symbolversions "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	schematicversions "github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	statusversions "github.com/synnaxlabs/synnax/pkg/service/status/versions"
	tableversions "github.com/synnaxlabs/synnax/pkg/service/table/versions"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/task/versions"
	userversions "github.com/synnaxlabs/synnax/pkg/service/user/versions"
	viewversions "github.com/synnaxlabs/synnax/pkg/service/view/versions"
	"github.com/synnaxlabs/x/migrate"
)

// Every table first stored by v0.53 or earlier holds rows under the old key prefix, and
// only the first migration in its chain moves them. Dropping that step orphans the rows
// with no other spec failing.
var _ = DescribeTable("Pre-v0.54 key normalization",
	func(chain []migrate.Migration) {
		Expect(chain).ToNot(BeEmpty())
		Expect(chain[0].Key()).To(Equal("normalize_keys"))
	},
	Entry("alias", aliasversions.Migrations),
	Entry("arc", arcversions.Migrations),
	Entry("auth", authversions.Migrations),
	Entry("channel", channelversions.Migrations),
	Entry("device", deviceversions.NewMigrations(deviceversions.MigrationsConfig{})),
	Entry("group", groupversions.Migrations),
	Entry("label", labelversions.Migrations),
	Entry("lineplot", lineplotversions.Migrations),
	Entry("log", logversions.Migrations),
	Entry("policy", policyversions.Migrations),
	Entry("project", projectversions.NewMigrations(projectversions.MigrationsConfig{})),
	Entry("rack", rackversions.NewMigrations(rackversions.MigrationsConfig{})),
	Entry("ranger", rangerversions.NewMigrations(rangerversions.MigrationsConfig{})),
	Entry("ranger kv", kvversions.Migrations),
	Entry("relationship", ontologyversions.RelationshipMigrations),
	Entry("resource", ontologyversions.ResourceMigrations),
	Entry("role", roleversions.Migrations),
	Entry("schematic", schematicversions.Migrations),
	Entry("status", statusversions.Migrations),
	Entry("symbol", symbolversions.Migrations),
	Entry("table", tableversions.Migrations),
	Entry("task", taskversions.NewMigrations(taskversions.MigrationsConfig{})),
	Entry("user", userversions.Migrations),
	Entry("view", viewversions.Migrations),
)
