// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelServices } from "@/channel/services";
import { Node } from "@/cluster/node";
import { ClusterServices } from "@/cluster/services";
import { Group } from "@/group";
import { Device } from "@/hardware/device";
import { Task } from "@/hardware/task";
import { LinePlotServices } from "@/lineplot/services";
import { Ontology } from "@/ontology";
import { Builtin } from "@/ontology/builtin";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { User } from "@/user";
import { Workspace } from "@/workspace";

export const EMPTY_ONTOLOGY_SERVICE: Ontology.Service = {
  type: "rack",
  icon: <></>,
  hasChildren: true,
  canDrop: () => false,
  onMosaicDrop: () => {},
  TreeContextMenu: () => <></>,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => false,
  onRename: undefined,
};

export const SERVICES: Ontology.Services = {
  schematic: SchematicServices.ONTOLOGY_SERVICE,
  cluster: ClusterServices.ONTOLOGY_SERVICE,
  user: User.ONTOLOGY_SERVICE,
  builtin: Builtin.ONTOLOGY_SERVICE,
  node: Node.ONTOLOGY_SERVICE,
  group: Group.ONTOLOGY_SERVICE,
  range: RangeServices.ONTOLOGY_SERVICE,
  workspace: Workspace.ONTOLOGY_SERVICE,
  lineplot: LinePlotServices.ONTOLOGY_SERVICE,
  "range-alias": EMPTY_ONTOLOGY_SERVICE,
  label: EMPTY_ONTOLOGY_SERVICE,
  rack: EMPTY_ONTOLOGY_SERVICE,
  task: Task.ONTOLOGY_SERVICE,
  device: Device.ONTOLOGY_SERVICE,
  channel: ChannelServices.ONTOLOGY_SERVICE,
  policy: EMPTY_ONTOLOGY_SERVICE,
};
