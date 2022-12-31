// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { OntologyID } from "@synnaxlabs/client";
import type { OntologyResourceType } from "@synnaxlabs/client";
import { AiFillDatabase } from "react-icons/ai";
import { MdOutlineDeviceHub, MdSensors } from "react-icons/md";

import { ClusterIcon } from "@/features/cluster";
import { LayoutPlacer } from "@/features/layout";
import { LinePlotVisualization, createVisualization } from "@/features/visualization";
import { WorkspaceState } from "@/features/workspace";

export interface SelectionContext {
  id: OntologyID;
  placer: LayoutPlacer;
  workspace: WorkspaceState;
}

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  onSelect?: (ctx: SelectionContext) => void;
  hasChildren: boolean;
}

export const resourceTypes: Record<string, ResourceType> = {
  builtin: {
    type: "builtin",
    icon: <AiFillDatabase />,
    hasChildren: true,
  },
  cluster: {
    type: "cluster",
    icon: <ClusterIcon />,
    hasChildren: true,
  },
  node: {
    type: "node",
    icon: <MdOutlineDeviceHub />,
    hasChildren: true,
  },
  channel: {
    type: "channel",
    icon: <MdSensors />,
    hasChildren: false,
    onSelect: ({ placer, id, workspace }: SelectionContext) => {
      console.log(workspace.selectedRangeKey);
      placer(
        createVisualization<LinePlotVisualization>({
          channels: [id.key],
          ranges:
            workspace.selectedRangeKey != null ? [workspace.selectedRangeKey] : [],
        })
      );
    },
  },
};
