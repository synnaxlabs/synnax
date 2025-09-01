// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";

import { ChannelServices } from "@/channel/services";
import { Node } from "@/cluster/node";
import { ClusterServices } from "@/cluster/services";
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

const createEmptyService = (type: ontology.ResourceType): Ontology.Service => ({
  ...Ontology.NOOP_SERVICE,
  type,
});

export const SERVICES: Ontology.Services = {
  schematic: SchematicServices.ONTOLOGY_SERVICE,
  schematic_symbol: createEmptyService("schematic_symbol"),
  cluster: ClusterServices.ONTOLOGY_SERVICE,
  user: UserServices.ONTOLOGY_SERVICE,
  builtin: createEmptyService(ontology.BUILTIN_TYPE),
  node: Node.ONTOLOGY_SERVICE,
  group: GroupServices.ONTOLOGY_SERVICE,
  range: RangeServices.ONTOLOGY_SERVICE,
  workspace: WorkspaceServices.ONTOLOGY_SERVICE,
  lineplot: LinePlotServices.ONTOLOGY_SERVICE,
  "range-alias": createEmptyService("range-alias"),
  label: createEmptyService("label"),
  rack: Hardware.Rack.ONTOLOGY_SERVICE,
  task: Hardware.Task.ONTOLOGY_SERVICE,
  device: Hardware.Device.ONTOLOGY_SERVICE,
  channel: ChannelServices.ONTOLOGY_SERVICE,
  framer: createEmptyService("framer"),
  policy: createEmptyService("policy"),
  allow_all: createEmptyService("allow_all"),
  log: LogServices.ONTOLOGY_SERVICE,
  table: TableServices.ONTOLOGY_SERVICE,
};
