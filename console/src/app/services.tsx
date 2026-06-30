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

import { Access } from "@/access";
import { ChannelServices } from "@/service/channel/services";
import { GroupServices } from "@/service/group/services";
import { Hardware } from "@/service/hardware";
import { Arc } from "@/service/arc";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";
import { UserServices } from "@/service/user/services";
import { Ontology } from "@/ontology";
import { ProjectServices } from "@/project/services";
import { RangeServices } from "@/range/services";

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
  user: UserServices.ONTOLOGY_SERVICE,
  builtin: createEmptyService("builtin"),
  node: createEmptyService("node", <Icon.Node />),
  group: GroupServices.ONTOLOGY_SERVICE,
  range: RangeServices.ONTOLOGY_SERVICE,
  project: ProjectServices.ONTOLOGY_SERVICE,
  lineplot: LinePlot.ONTOLOGY_SERVICE,
  "range-alias": createEmptyService("range-alias"),
  label: createEmptyService("label", <Icon.Label />),
  rack: Hardware.Rack.ONTOLOGY_SERVICE,
  task: Hardware.Task.ONTOLOGY_SERVICE,
  device: Hardware.Device.ONTOLOGY_SERVICE,
  channel: ChannelServices.ONTOLOGY_SERVICE,
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
