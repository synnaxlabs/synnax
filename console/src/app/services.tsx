// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { Access } from "@/service/access";
import { Arc } from "@/service/arc";
import { Channel } from "@/service/channel";
import { Device } from "@/service/device";
import { Group } from "@/service/group";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Ontology } from "@/service/ontology";
import { Project } from "@/service/project";
import { Rack } from "@/service/rack";
import { Range } from "@/service/range";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";
import { Task } from "@/service/task";
import { User } from "@/service/user";

const createEmptyService = (
  type: ontology.ResourceType,
  icon?: Icon.ReactElement,
): Ontology.Service => {
  const service = { ...Ontology.NOOP_SERVICE, type };
  if (icon != null) service.icon = icon;
  return service;
};

export const SERVICES: Ontology.Services = {
  schematic: Schematic.ONTOLOGY_SERVICE,
  schematic_symbol: createEmptyService("schematic_symbol", <Icon.Schematic />),
  user: User.ONTOLOGY_SERVICE,
  builtin: createEmptyService("builtin"),
  node: createEmptyService("node", <Icon.Node />),
  group: Group.ONTOLOGY_SERVICE,
  range: Range.ONTOLOGY_SERVICE,
  project: Project.ONTOLOGY_SERVICE,
  lineplot: LinePlot.ONTOLOGY_SERVICE,
  "range-alias": createEmptyService("range-alias"),
  label: createEmptyService("label", <Icon.Label />),
  rack: Rack.ONTOLOGY_SERVICE,
  task: Task.ONTOLOGY_SERVICE,
  device: Device.ONTOLOGY_SERVICE,
  channel: Channel.ONTOLOGY_SERVICE,
  framer: createEmptyService("framer"),
  policy: Access.Policy.ONTOLOGY_SERVICE,
  log: Log.ONTOLOGY_SERVICE,
  table: Table.ONTOLOGY_SERVICE,
  panel: createEmptyService("panel"),
  status: createEmptyService("status", <Icon.Status />),
  arc: Arc.ONTOLOGY_SERVICE,
  view: createEmptyService("view"),
  role: Access.Role.ONTOLOGY_SERVICE,
};
