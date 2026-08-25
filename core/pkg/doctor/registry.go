// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	roleversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcversions "github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelversions "github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	deviceversions "github.com/synnaxlabs/synnax/pkg/service/device/versions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupversions "github.com/synnaxlabs/synnax/pkg/service/group/versions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	labelversions "github.com/synnaxlabs/synnax/pkg/service/label/versions"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	lineplotversions "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	logversions "github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	ontologyversions "github.com/synnaxlabs/synnax/pkg/service/ontology/versions"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	panelversions "github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	projectversions "github.com/synnaxlabs/synnax/pkg/service/project/versions"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackversions "github.com/synnaxlabs/synnax/pkg/service/rack/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	aliasversions "github.com/synnaxlabs/synnax/pkg/service/ranger/alias/versions"
	rangekv "github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	rangekvversions "github.com/synnaxlabs/synnax/pkg/service/ranger/kv/versions"
	rangerversions "github.com/synnaxlabs/synnax/pkg/service/ranger/versions"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	symbolversions "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	schematicversions "github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	statusversions "github.com/synnaxlabs/synnax/pkg/service/status/versions"
	xtable "github.com/synnaxlabs/synnax/pkg/service/table"
	tableversions "github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/task/versions"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	userversions "github.com/synnaxlabs/synnax/pkg/service/user/versions"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	viewversions "github.com/synnaxlabs/synnax/pkg/service/view/versions"
)

// whitelist holds the ontology types no table backs: they are synthesized during
// traversal rather than stored.
var whitelist = []ontology.ResourceType{
	ontology.ResourceTypeBuiltin,
	ontology.ResourceTypeFramer,
	ontology.ResourceTypeNode,
}

// newRegistry builds the doctor's table registry. Every gorp table the Core opens is
// listed here; a table left out reports its keys as an unknown prefix.
func newRegistry() []table {
	return []table{
		newTable(tableConfig[channel.Key, channel.Channel]{
			migrations: channelversions.Migrations,
			ontologyID: channel.Channel.OntologyID,
			collect: func(s *state, c channel.Channel) {
				s.channels[c.Key()] = c
			},
			check: checkChannel,
		}),
		newTable(tableConfig[ranger.Key, ranger.Range]{
			migrations: rangerversions.NewMigrations(rangerversions.MigrationsConfig{}),
			ontologyID: ranger.Range.OntologyID,
			collect:    func(s *state, r ranger.Range) { s.ranges.Add(r.Key) },
		}),
		newTable(tableConfig[string, alias.Alias]{
			migrations: aliasversions.Migrations,
			ontologyID: func(a alias.Alias) ontology.ID {
				return alias.OntologyID(a.Range, a.Channel)
			},
			check: checkAlias,
		}),
		newTable(tableConfig[string, rangekv.Pair]{
			migrations: rangekvversions.Migrations,
			check:      checkRangeKV,
		}),
		newTable(tableConfig[rack.Key, rack.Rack]{
			migrations: rackversions.NewMigrations(rackversions.MigrationsConfig{}),
			ontologyID: rack.Rack.OntologyID,
			collect:    func(s *state, r rack.Rack) { s.racks.Add(r.Key) },
		}),
		newTable(tableConfig[task.Key, task.Task]{
			migrations: taskversions.NewMigrations(taskversions.MigrationsConfig{}),
			ontologyID: task.Task.OntologyID,
			collect:    func(s *state, t task.Task) { s.tasks.Add(t.Key) },
			check:      checkTask,
		}),
		newTable(tableConfig[device.Key, device.Device]{
			migrations: deviceversions.NewMigrations(deviceversions.MigrationsConfig{}),
			ontologyID: device.Device.OntologyID,
			check:      checkDevice,
		}),
		newTable(tableConfig[user.Key, user.User]{
			migrations: userversions.Migrations,
			ontologyID: user.User.OntologyID,
			collect:    func(s *state, u user.User) { s.usernames.Add(u.Username) },
		}),
		newTable(tableConfig[string, auth.SecureCredentials]{
			migrations: authversions.Migrations,
			check:      checkCredentials,
		}),
		newTable(tableConfig[policy.Key, policy.Policy]{
			migrations: policyversions.Migrations,
			ontologyID: policy.Policy.OntologyID,
			check:      checkPolicy,
		}),
		newTable(tableConfig[role.Key, role.Role]{
			migrations: roleversions.Migrations,
			ontologyID: role.Role.OntologyID,
		}),
		newTable(tableConfig[panel.Key, panel.Panel]{
			migrations: panelversions.Migrations,
			ontologyID: panel.Panel.OntologyID,
			check:      checkPanel,
		}),
		newTable(tableConfig[string, ontology.Resource]{
			migrations: ontologyversions.ResourceMigrations,
			collect:    func(s *state, r ontology.Resource) { s.resources.Add(r.ID.String()) },
			check:      checkResource,
		}),
		newTable(tableConfig[string, ontology.Relationship]{
			migrations: ontologyversions.RelationshipMigrations,
			checkKey:   checkRelationshipKey,
			check:      checkRelationship,
		}),
		newTable(tableConfig[group.Key, group.Group]{
			migrations: groupversions.Migrations,
			ontologyID: group.Group.OntologyID,
		}),
		newTable(tableConfig[label.Key, label.Label]{
			migrations: labelversions.Migrations,
			ontologyID: label.Label.OntologyID,
		}),
		newTable(tableConfig[project.Key, project.Project]{
			migrations: projectversions.NewMigrations(
				projectversions.MigrationsConfig{},
			),
			ontologyID: project.Project.OntologyID,
		}),
		newTable(tableConfig[schematic.Key, schematic.Schematic]{
			migrations: schematicversions.Migrations,
			ontologyID: schematic.Schematic.OntologyID,
		}),
		newTable(tableConfig[symbol.Key, symbol.Symbol]{
			migrations: symbolversions.Migrations,
			ontologyID: symbol.Symbol.OntologyID,
		}),
		newTable(tableConfig[xtable.Key, xtable.Table]{
			migrations: tableversions.Migrations,
			ontologyID: xtable.Table.OntologyID,
		}),
		newTable(tableConfig[lineplot.Key, lineplot.LinePlot]{
			migrations: lineplotversions.Migrations,
			ontologyID: lineplot.LinePlot.OntologyID,
		}),
		newTable(tableConfig[log.Key, log.Log]{
			migrations: logversions.Migrations,
			ontologyID: log.Log.OntologyID,
		}),
		newTable(tableConfig[view.Key, view.View]{
			migrations: viewversions.Migrations,
			ontologyID: view.View.OntologyID,
		}),
		newTable(tableConfig[arc.Key, arc.Arc]{
			migrations: arcversions.Migrations,
			ontologyID: arc.Arc.OntologyID,
		}),
		newTable(tableConfig[status.Key, status.Status[any]]{
			migrations: statusversions.Migrations,
			ontologyID: status.Status[any].OntologyID,
		}),
	}
}
