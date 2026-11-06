// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/lyra/status";
import { type Tree } from "@synnaxlabs/lyra/tree";
import { type destructor } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Panel } from "@/platform/panel";
import { type Session } from "@/session";

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
  /** Calls the handler when the resource with the given key changes. */
  subscribeResource: (handler: () => void, key: string) => destructor.Destructor;
  setSelection: (keys: string[]) => void;
  expand: (key: string) => void;
  contract: (key: string) => void;
}

export interface BaseProps {
  client: Synnax;
  store: Session.Store;
  openTab: Panel.OpenTab;
  addStatus: Status.Adder;
  handleError: Status.ErrorHandler;
}

export interface ContextMenuProps extends BaseProps {
  selection: {
    /** Null when the top selected items sit under more than one parent. */
    parentID: ontology.ID | null;
    rootID: ontology.ID;
    ids: ontology.ID[];
    /** The shallowest selected items, without any inside another selected item. */
    top: ontology.ID[];
  };
  state: TreeState;
}

export interface ContextMenu extends FC<ContextMenuProps> {}

export interface AllowRename {
  (resource: ontology.Resource): boolean;
}
