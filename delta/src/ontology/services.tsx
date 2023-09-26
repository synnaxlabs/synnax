// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Node } from "@/cluster/node";
import { Group } from "@/group";
import { Builtin } from "@/ontology/builtin";
import { type Service } from "@/ontology/service";
import { PID } from "@/pid";
import { Range } from "@/range";
import { User } from "@/user";
import { Workspace } from "@/workspace";

export const SERVICES: Record<ontology.ResourceType, Service> = {
  pid: PID.RESOURCE_SERVICE,
  channel: Channel.ONTOLOGY_SERVICE,
  cluster: Cluster.ONTOLOGY_SERVICE,
  user: User.RESOURCE_SERVICE,
  builtin: Builtin.ONTOLOGY_SERVICE,
  node: Node.ONTOLOGY_SERVICE,
  group: Group.ONTOLOGY_SERVICE,
  range: Range.ONTOLOGY_SERVICE,
  workspace: Workspace.ONTOLOGY_SERVICE,
};
