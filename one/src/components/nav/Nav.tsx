// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as PMenu, Nav } from "@synnaxlabs/pluto";
import { CSS as PCSS } from "@synnaxlabs/pluto";
import { Text } from "@synnaxlabs/pluto/text";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { Vis } from "@/vis";

export const NAV_DRAWERS: Layout.NavDrawerItem[] = [
  Ontology.Toolbar,
  Range.Toolbar,
  Vis.Toolbar,
  Task.Toolbar,
];

interface NavMenuProps extends Omit<PMenu.MenuProps, "children"> {
  children: Layout.NavMenuItem[];
  activeItem?: Layout.NavDrawerItem;
}

export const NavMenu = ({
  children,
  activeItem,
  ...props
}: NavMenuProps): ReactElement => (
  <PMenu.Menu {...props}>
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

export interface NavDrawerProps {
  location: Layout.NavDrawerLocation;
}

export const NavDrawer = ({ location: l, ...props }: NavDrawerProps): ReactElement => {
  const { activeItem, onResize, onSelect } = Layout.useNavDrawer(l, NAV_DRAWERS);
  return (
    <Nav.Drawer
      location={l}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", location.direction(l)),
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
      {...props}
    />
  );
};
