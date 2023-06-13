// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Synnax } from "@synnaxlabs/client";
import type { OntologyResourceType } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Hauled } from "@synnaxlabs/pluto";

import { LayoutPlacer } from "@/layout";
import { WorkspaceState } from "@/workspace";

export interface ResourceContext {
  client: Synnax;
  placer: LayoutPlacer;
  workspace: WorkspaceState;
}

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  hasChildren: boolean;
  acceptsDrop: (hauled: Hauled[]) => boolean;
  onDrop: (ctx: ResourceContext, hauled: Hauled[]) => void;
  contextMenu: (ctx: ResourceContext, hauled: Hauled[]) => ReactElement;
}

export const resourceTypes: Record<string, ResourceType> = {
  builtin: {
    type: "builtin",
    icon: <Icon.Cluster />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  cluster: {
    type: "cluster",
    icon: <Icon.Cluster />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  node: {
    type: "node",
    icon: <Icon.Node />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  channel: {
    type: "channel",
    icon: <Icon.Channel />,
    hasChildren: false,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  group: {
    type: "group",
    hasChildren: true,
    acceptsDrop: () => true,
    onDrop: () => {},
  },
  range: {
    type: "range",
    hasChildren: true,
    acceptsDrop: () => true,
    onDrop: () => {},
  },
};
