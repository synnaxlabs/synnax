// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { type FC } from "react";

import { type Items } from "@/platform/tree/item";
import { type ContextMenu, type RootContextMenuProps } from "@/platform/tree/types";

export interface ContextMenus {
  /** The menu for a right-click on empty space. */
  root: FC<RootContextMenuProps>;
  /** The menu for a selection of more than one resource type. */
  mixed: ContextMenu;
}

interface ContextValue {
  items: Items;
  contextMenus: ContextMenus;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.Context",
  providerName: "Tree.Provider",
});

export { Context };

export const useItems = (): Items => useContext("Tree.useItems").items;

export const useContextMenus = (): ContextMenus =>
  useContext("Tree.useContextMenus").contextMenus;
