// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { type Status, type Tree } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { type Layout } from "@/platform/layout";
import { type Session } from "@/session";

// Entry is the identity and display name of an item passed to a resource type's select
// handler. The name is resolved by the type's own Item from its flux store, not carried
// on the ontology resource.
export interface Entry {
  // key is the string form of id, matching the tree's node and list keys.
  key: string;
  id: ontology.ID;
  name: string;
}

export interface TreeState {
  nodes: Tree.Node[];
  shape: Tree.Shape;
  setNodes: (nodes: Tree.Node[]) => void;
  setSelection: (keys: string[]) => void;
  expand: (key: string) => void;
  contract: (key: string) => void;
  setLoading: (key: string | false) => void;
}

export interface BaseProps {
  client: Synnax;
  store: Session.Store;
  placeLayout: Layout.Placer;
  removeLayout: Layout.Remover;
  addStatus: Status.Adder;
  handleError: Status.ErrorHandler;
}

export interface ContextMenuProps extends BaseProps {
  selection: {
    parentID: ontology.ID;
    rootID: ontology.ID;
    ids: ontology.ID[];
  };
  state: TreeState;
}

export interface ContextMenu extends FC<ContextMenuProps> {}

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
