import { Menu as PMenu, Nav } from "@synnaxlabs/pluto";
import { Text } from "@synnaxlabs/pluto/text";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { Vis } from "@/vis";

export const NAV_DRAWERS: Layout.NavDrawerItem[] = [
  Ontology.Toolbar,
  Range.Toolbar,
  Vis.Toolbar,
];

export const NavMenu = ({
  children,
  ...props
}: {
  children: Layout.NavMenuItem[];
} & Omit<PMenu.MenuProps, "children">): ReactElement => (
  <PMenu.Menu {...props}>
    {children.map(({ key, tooltip, icon }) => (
      <PMenu.Item.Icon
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
  location: Layout.NavdrawerLocation;
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
