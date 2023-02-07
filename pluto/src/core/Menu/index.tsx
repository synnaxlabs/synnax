// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as CoreMenu, MenuItem } from "./Menu";
import { MenuContext } from "./MenuContext";
export type { MenuContextProps, MenuContextMenuProps } from "./MenuContext";
export type { MenuProps, MenuItemProps } from "./Menu";

type CoreMenuType = typeof CoreMenu;

export interface MenuType extends CoreMenuType {
  Context: typeof MenuContext;
  Item: typeof MenuItem;
}

export const Menu = CoreMenu as MenuType;

Menu.Context = MenuContext;
Menu.Item = MenuItem;
