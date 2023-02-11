// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu, useContextMenu } from "./ContextMenu";
import { Menu as CoreMenu, MenuItem } from "./Menu";
export type { ContextMenuProps, UseContextMenuReturn } from "./ContextMenu";
export type { MenuProps, MenuItemProps } from "./Menu";

type CoreMenuType = typeof CoreMenu;

export interface MenuType extends CoreMenuType {
  ContextMenu: typeof ContextMenu;
  useContextMenu: typeof useContextMenu;
  Item: typeof MenuItem;
}

export const Menu = CoreMenu as MenuType;

Menu.ContextMenu = ContextMenu;
Menu.useContextMenu = useContextMenu;
Menu.Item = MenuItem;
