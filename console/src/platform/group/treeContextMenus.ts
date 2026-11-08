// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RootContextMenu } from "@/platform/group/RootContextMenu";
import { SelectionContextMenu } from "@/platform/group/SelectionContextMenu";
import { type Tree } from "@/platform/tree";

/** The tree menus that create groups. */
export const TREE_CONTEXT_MENUS: Tree.ContextMenus = {
  root: RootContextMenu,
  mixed: SelectionContextMenu,
};
