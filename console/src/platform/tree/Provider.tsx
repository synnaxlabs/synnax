// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Context, type ContextMenus } from "@/platform/tree/context";
import { type Items } from "@/platform/tree/item";
import { CopyContextMenu, FallbackContextMenu } from "@/platform/tree/menu";

const FALLBACK_CONTEXT_MENUS: ContextMenus = {
  root: FallbackContextMenu,
  mixed: CopyContextMenu,
};

export interface ProviderProps extends PropsWithChildren {
  items: Items;
  /**
   * Defaults to a root menu that holds only the reload of the Console, and a mixed menu
   * that adds Copy.
   */
  contextMenus?: ContextMenus;
}

export const Provider = ({
  items,
  contextMenus = FALLBACK_CONTEXT_MENUS,
  ...rest
}: ProviderProps): ReactElement => (
  <Context value={{ items, contextMenus }} {...rest} />
);
