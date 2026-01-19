// Copyright 2026 Synnax Labs, Inc.
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

export interface GetResource {
  (id: ontology.ID | string): ontology.Resource;
  (ids: (ontology.ID | string)[]): ontology.Resource[];
}

export interface TreeState {
  nodes: Tree.Node[];
  shape: Tree.Shape;
  setNodes: (nodes: Tree.Node[]) => void;
  setResource: (resource: ontology.Resource | ontology.Resource[]) => void;
  getResource: GetResource;
  setSelection: (keys: string[]) => void;
  expand: (key: string) => void;
  contract: (key: string) => void;
  setLoading: (key: string | false) => void;
}

export interface BaseProps {
  client: Synnax;
  store: RootStore;
  placeLayout: Layout.Placer;
  removeLayout: Layout.Remover;
  services: Services;
  addStatus: Status.Adder;
  handleError: Status.ErrorHandler;
}

export interface HandleSelectProps extends BaseProps {
  selection: ontology.Resource[];
}

export interface HandleSelect {
  (props: HandleSelectProps): void;
}

export interface HandleMosaicDropProps extends BaseProps {
  nodeKey: number;
  location: location.Location;
  id: ontology.ID;
}

export interface HandleMosaicDrop {
  (props: HandleMosaicDropProps): void;
}

export interface TreeContextMenuProps extends BaseProps {
  selection: {
    parentID: ontology.ID;
    rootID: ontology.ID;
    ids: ontology.ID[];
  };
  state: TreeState;
}

export interface TreeContextMenu extends FC<TreeContextMenuProps> {}

export interface HandleTreeRenameProps extends BaseProps {
  id: ontology.ID;
  name: string;
  state: TreeState;
}

export interface HandleTreeRename {
  eager?: (props: HandleTreeRenameProps) => void;
  execute: (props: HandleTreeRenameProps) => Promise<void>;
  rollback?: (props: HandleTreeRenameProps, prevName: string) => void;
}

export interface AllowRename {
  (resource: ontology.Resource): boolean;
}

export interface PaletteListItem extends FC<List.ItemProps<string>> {}

export interface TreeItemProps extends Omit<Tree.ItemProps<string>, "id" | "resource"> {
  id: ontology.ID;
  icon?: Icon.ReactElement;
  loading: boolean;
  resource: ontology.Resource;
  onDoubleClick: () => void;
}

export interface Service {
  type: ontology.ResourceType;
  icon?: Icon.ReactElement | ((resource: ontology.Resource) => Icon.ReactElement);
  hasChildren: boolean;
  onSelect?: HandleSelect;
  canDrop: Haul.CanDrop;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  Item?: FC<TreeItemProps>;
  onMosaicDrop?: HandleMosaicDrop;
  TreeContextMenu?: TreeContextMenu;
  PaletteListItem?: PaletteListItem;
  visible?: (resource: ontology.Resource) => boolean;
}

export const NOOP_SERVICE: Omit<Service, "type"> = {
  hasChildren: true,
  canDrop: () => false,
  haulItems: () => [],
};

export interface Services extends Record<ontology.ResourceType, Service> {}
