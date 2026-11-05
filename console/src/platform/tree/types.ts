// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, type Synnax } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/lyra/status";
import { type Tree } from "@synnaxlabs/lyra/tree";
import { type destructor, type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { type Panel } from "@/platform/panel";
import { type Session } from "@/session";

/** A record that a tree shows: its resource ID and name, and the record itself. */
export interface Resource<T extends record.Unknown = record.Unknown>
  extends resource.Resource {
  data?: T | null;
}

export interface GetResource {
  (id: resource.ID | string): Resource;
  (ids: (resource.ID | string)[]): Resource[];
}

export interface TreeState {
  nodes: Tree.Node[];
  shape: Tree.Shape;
  setNodes: (nodes: Tree.Node[]) => void;
  setResource: (resource: Resource | Resource[]) => void;
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
    parentID: resource.ID;
    rootID: resource.ID;
    ids: resource.ID[];
  };
  state: TreeState;
}

export interface ContextMenu extends FC<ContextMenuProps> {}

export interface AllowRename {
  (resource: Resource): boolean;
}
