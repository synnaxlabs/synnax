// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";

import { ONTOLOGY_SERVICE as CHANNEL_ONTOLOGY_SERVICE } from "@/channel/ontology";
import { Cluster } from "@/cluster";
import { Node } from "@/cluster/node";
import { Group } from "@/group";
import { LinePlot } from "@/lineplot";
import { Builtin } from "@/ontology/builtin";
import { type Service } from "@/ontology/service";
import { PID } from "@/pid";
import { ONTOLOGY_SERVICE as RANGE_ONTOLOGY_SERVICE } from "@/range/ontology";
import { User } from "@/user";
import { Workspace } from "@/workspace";

export const EMPTY_ONTOLOGY_SERVICE: Service = {
  type: "rack",
  icon: <></>,
  hasChildren: true,
  canDrop: () => false,
  onMosaicDrop: () => {},
  TreeContextMenu: () => <></>,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => false,
  onRename: () => {},
};

export const SERVICES: Record<ontology.ResourceType, Service> = {
  pid: PID.ONTOLOGY_SERVICE,
  channel: CHANNEL_ONTOLOGY_SERVICE,
  cluster: Cluster.ONTOLOGY_SERVICE,
  user: User.ONTOLOGY_SERVICE,
  builtin: Builtin.ONTOLOGY_SERVICE,
  node: Node.ONTOLOGY_SERVICE,
  group: Group.ONTOLOGY_SERVICE,
  range: RANGE_ONTOLOGY_SERVICE,
  workspace: Workspace.ONTOLOGY_SERVICE,
  lineplot: LinePlot.ONTOLOGY_SERVICE,
  "range-alias": EMPTY_ONTOLOGY_SERVICE,
  label: EMPTY_ONTOLOGY_SERVICE,
  rack: EMPTY_ONTOLOGY_SERVICE,
};
