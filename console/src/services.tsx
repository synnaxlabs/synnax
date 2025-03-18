// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  device,
  framer,
  group,
  label,
  linePlot,
  log,
  ontology,
  policy,
  rack,
  ranger,
  schematic,
  table,
  task,
  user,
  workspace,
} from "@synnaxlabs/client";

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
  [schematic.ONTOLOGY_TYPE]: SchematicServices.ONTOLOGY_SERVICE,
  [ontology.CLUSTER_TYPE]: ClusterServices.ONTOLOGY_SERVICE,
  [user.ONTOLOGY_TYPE]: UserServices.ONTOLOGY_SERVICE,
  [ontology.BUILTIN_TYPE]: createEmptyService(ontology.BUILTIN_TYPE),
  [ontology.NODE_TYPE]: Node.ONTOLOGY_SERVICE,
  [group.ONTOLOGY_TYPE]: GroupServices.ONTOLOGY_SERVICE,
  [ranger.ONTOLOGY_TYPE]: RangeServices.ONTOLOGY_SERVICE,
  [workspace.ONTOLOGY_TYPE]: WorkspaceServices.ONTOLOGY_SERVICE,
  [linePlot.ONTOLOGY_TYPE]: LinePlotServices.ONTOLOGY_SERVICE,
  [ranger.ALIAS_ONTOLOGY_TYPE]: createEmptyService(ranger.ALIAS_ONTOLOGY_TYPE),
  [label.ONTOLOGY_TYPE]: createEmptyService(label.ONTOLOGY_TYPE),
  [rack.ONTOLOGY_TYPE]: Hardware.Rack.ONTOLOGY_SERVICE,
  [task.ONTOLOGY_TYPE]: Hardware.Task.ONTOLOGY_SERVICE,
  [device.ONTOLOGY_TYPE]: Hardware.Device.ONTOLOGY_SERVICE,
  [channel.ONTOLOGY_TYPE]: ChannelServices.ONTOLOGY_SERVICE,
  [framer.ONTOLOGY_TYPE]: createEmptyService(framer.ONTOLOGY_TYPE),
  [policy.ONTOLOGY_TYPE]: createEmptyService(policy.ONTOLOGY_TYPE),
  [policy.ALLOW_ALL_ONTOLOGY_TYPE]: createEmptyService(policy.ALLOW_ALL_ONTOLOGY_TYPE),
  [log.ONTOLOGY_TYPE]: LogServices.ONTOLOGY_SERVICE,
  [table.ONTOLOGY_TYPE]: TableServices.ONTOLOGY_SERVICE,
};
