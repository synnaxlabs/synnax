// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { type Tree, type Haul } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";
import { type RootStore } from "@/store";

export interface SelectionContext {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  selection: ontology.Resource[];
}

export interface TreeContextMenuProps {
  selection: {
    parent: Tree.Node;
    resources: ontology.Resource[];
    nodes: Tree.Node[];
  };
  state: {
    resources: ontology.Resource[];
    nodes: Tree.Node[];
    setNodes: (nodes: Tree.Node[]) => void;
  };
}

export interface Service {
  type: ontology.ResourceType;
  icon: ReactElement;
  hasChildren: boolean;
  allowRename: (res: ontology.Resource) => boolean;
  onSelect: (ctx: SelectionContext) => void;
  canDrop: (hauled: Haul.Item[]) => boolean;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  TreeContextMenu: (props: TreeContextMenuProps) => ReactElement;
}
