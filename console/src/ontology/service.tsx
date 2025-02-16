// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import {
  type Haul,
  type Icon,
  type List,
  type Status,
  type Tree,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Layout } from "@/layout";
import { type RootStore } from "@/store";

export interface BaseProps {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  removeLayout: Layout.Remover;
  services: Services;
  addStatus: Status.Adder;
  handleException: Status.ExceptionHandler;
}

export interface HandleSelectProps extends BaseProps {
  selection: ontology.Resource[];
}

export interface HandleSelect {
  (props: HandleSelectProps): void;
}

export interface HandleMosaicDropProps {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  nodeKey: number;
  location: location.Location;
  addStatus: Status.Adder;
  handleException: Status.ExceptionHandler;
  id: ontology.ID;
}

export interface HandleMosaicDrop {
  (props: HandleMosaicDropProps): void;
}

export interface TreeContextMenuProps extends BaseProps {
  selection: {
    parentID: ontology.ID;
    resources: ontology.Resource[];
    nodes: Tree.NodeWithPosition[];
  };
  state: {
    resources: ontology.Resource[];
    nodes: Tree.Node[];
    setNodes: (nodes: Tree.Node[]) => void;
    setResources: (resources: ontology.Resource[]) => void;
    setSelection: (keys: string[]) => void;
    expand: (key: string) => void;
    contract: (key: string) => void;
    setLoading: (key: string | false) => void;
  };
}

export interface TreeContextMenu extends FC<TreeContextMenuProps> {}

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

export interface HandleTreeRename {
  eager?: (props: HandleTreeRenameProps) => void;
  execute: (props: HandleTreeRenameProps) => Promise<void>;
  rollback?: (props: HandleTreeRenameProps, prevName: string) => void;
}

export interface NodeAdapterProps extends BaseProps {
  node: Tree.FlattenedNode;
}

export interface AllowRename {
  (res: ontology.Resource): boolean;
}

export interface PaletteListItem
  extends FC<List.ItemProps<string, ontology.Resource>> {}

export interface Service {
  type: ontology.ResourceType;
  icon: Icon.Element | ((resource: ontology.Resource) => Icon.Element);
  hasChildren: boolean;
  onSelect?: HandleSelect;
  canDrop: Haul.CanDrop;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  allowRename: AllowRename;
  Item?: Tree.Item;
  onRename?: HandleTreeRename;
  onMosaicDrop?: HandleMosaicDrop;
  TreeContextMenu?: TreeContextMenu;
  PaletteListItem?: PaletteListItem;
}

export const NOOP_SERVICE: Omit<Service, "type"> = {
  icon: <></>,
  hasChildren: true,
  onSelect: () => {},
  canDrop: () => false,
  haulItems: () => [],
  allowRename: () => false,
};

export interface Services extends Record<ontology.ResourceType, Service> {}
