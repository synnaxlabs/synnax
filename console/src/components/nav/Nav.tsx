// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS as PCSS, Menu as PMenu, Nav, Text } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { ChannelServices } from "@/channel/services";
import { CSS } from "@/css";
import { Hardware } from "@/hardware";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { UserServices } from "@/user/services";
import { Vis } from "@/vis";
import { WorkspaceServices } from "@/workspace/services";

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [
  Ontology.Toolbar,
  Range.TOOLBAR,
  Vis.Toolbar,
  ChannelServices.TOOLBAR,
  Hardware.TOOLBAR,
  UserServices.TOOLBAR,
  WorkspaceServices.TOOLBAR,
  ...Hardware.NAV_DRAWER_ITEMS,
];

interface NavMenuProps extends Omit<PMenu.MenuProps, "children"> {
  children: Layout.NavMenuItem[];
  activeItem?: Layout.NavDrawerItem;
}

export const NavMenu = ({
  children,
  activeItem,
  ...rest
}: NavMenuProps): ReactElement => (
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

export interface NavDrawerProps {
  location: Layout.NavDrawerLocation;
}

export const NavDrawer = ({ location: l }: NavDrawerProps): ReactElement => {
  const { activeItem, onResize, onSelect } = Layout.useNavDrawer(l, NAV_DRAWER_ITEMS);
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
    />
  );
};
