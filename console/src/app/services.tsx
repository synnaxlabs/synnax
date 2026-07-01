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

import { Access } from "@/feature/access";
import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Rack } from "@/feature/rack";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { User } from "@/feature/user";
import { Group } from "@/platform/group";
import { Ontology } from "@/platform/ontology";

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
