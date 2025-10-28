// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { ArcServices } from "@/arc/services";
import { ChannelServices } from "@/channel/services";
import { GroupServices } from "@/group/services";
import { Hardware } from "@/hardware";
import { LinePlotServices } from "@/lineplot/services";
import { LogServices } from "@/log/services";
import { Ontology } from "@/ontology";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { TableServices } from "@/table/services";
import { UserServices } from "@/user/services";
import { WorkspaceServices } from "@/workspace/services";

const createEmptyService = (
  type: ontology.ResourceType,
  icon?: Icon.ReactElement,
): Ontology.Service => {
  const service = { ...Ontology.NOOP_SERVICE, type };
  if (icon != null) service.icon = icon;
  return service;
};

export const SERVICES: Ontology.Services = {
  schematic: SchematicServices.ONTOLOGY_SERVICE,
  schematic_symbol: createEmptyService("schematic_symbol", <Icon.Schematic />),
  cluster: createEmptyService("cluster", <Icon.Cluster />),
  user: UserServices.ONTOLOGY_SERVICE,
  builtin: createEmptyService("builtin"),
  node: createEmptyService("node", <Icon.Node />),
  group: GroupServices.ONTOLOGY_SERVICE,
  range: RangeServices.ONTOLOGY_SERVICE,
  workspace: WorkspaceServices.ONTOLOGY_SERVICE,
  lineplot: LinePlotServices.ONTOLOGY_SERVICE,
  "range-alias": createEmptyService("range-alias"),
  label: createEmptyService("label", <Icon.Label />),
  rack: Hardware.Rack.ONTOLOGY_SERVICE,
  task: Hardware.Task.ONTOLOGY_SERVICE,
  device: Hardware.Device.ONTOLOGY_SERVICE,
  channel: ChannelServices.ONTOLOGY_SERVICE,
  framer: createEmptyService("framer"),
  policy: createEmptyService("policy", <Icon.Access />),
  allow_all: createEmptyService("allow_all"),
  log: LogServices.ONTOLOGY_SERVICE,
  table: TableServices.ONTOLOGY_SERVICE,
  status: createEmptyService("status", <Icon.Status />),
  arc: ArcServices.ONTOLOGY_SERVICE,
  view: createEmptyService("view"),
};
