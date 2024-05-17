// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactElement } from "react";

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { type Tree, type Haul } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { type RootStore } from "@/store";

export interface BaseProps {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  removeLayout: Layout.Remover;
  services: Services;
}

export interface HandleSelectProps extends BaseProps {
  selection: ontology.Resource[];
}

export type HandleSelect = (props: HandleSelectProps) => void;

export interface HandleMosaicDropProps {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  nodeKey: number;
  location: location.Location;
  id: ontology.ID;
}

export type HandleMosaicDrop = (props: HandleMosaicDropProps) => void;

export interface TreeContextMenuProps extends Omit<HandleSelectProps, "selection"> {
  selection: {
    parent: Tree.Node;
    resources: ontology.Resource[];
    nodes: Tree.NodeWithPosition[];
  };
  state: {
    resources: ontology.Resource[];
    nodes: Tree.Node[];
    setNodes: (nodes: Tree.Node[]) => void;
    setResources: (resources: ontology.Resource[]) => void;
    setSelection: (keys: string[]) => void;
  };
}

export type TreeContextMenu = FC<TreeContextMenuProps>;

export interface HandleTreeRenameProps extends BaseProps {
  id: ontology.ID;
  name: string;
  state: {
    resources: ontology.Resource[];
    nodes: Tree.Node[];
    setNodes: (nodes: Tree.Node[]) => void;
    setResources: (resources: ontology.Resource[]) => void;
  };
}

export type HandleTreeRename = (props: HandleTreeRenameProps) => void;

export interface NodeAdapterProps extends BaseProps {
  node: Tree.FlattenedNode;
}

export interface Service {
  type: ontology.ResourceType;
  icon: ReactElement;
  hasChildren: boolean;
  onSelect: HandleSelect;
  canDrop: Haul.CanDrop;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  allowRename: (res: ontology.Resource) => boolean;
  Item?: Tree.Item;
  onRename?: (ctx: HandleTreeRenameProps) => void;
  onMosaicDrop?: HandleMosaicDrop;
  TreeContextMenu?: TreeContextMenu;
}

export type Services = Record<ontology.ResourceType, Service>;
