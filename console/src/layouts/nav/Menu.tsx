// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { CSS as PCSS, Menu as PMenu, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

export interface MenuProps extends Omit<PMenu.MenuProps, "children"> {
  children: Layout.NavMenuItem[];
  activeItem?: Layout.NavDrawerItem;
}

export const Menu = ({ children, activeItem, ...rest }: MenuProps) => (
  <PMenu.Menu {...rest}>
    {children.map(({ key, tooltip, icon }) => (
      <PMenu.Item.Icon
        className={CSS(
          CSS.BE("main-nav", "item"),
          PCSS.selected(activeItem?.key === key),
        )}
        key={key}
        itemKey={key}
        size="large"
        tooltip={<Text.Text level="small">{tooltip}</Text.Text>}
      >
        {icon}
      </PMenu.Item.Icon>
    ))}
  </PMenu.Menu>
);
