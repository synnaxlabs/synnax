import { Navbar } from "./Navbar";
import { NavDrawer } from "./Navdrawer";
export type { NavbarProps, NavbarContextValue } from "./Navbar";
export type { NavDrawerProps, NavDrawerItem } from "./Navdrawer";

export const Nav = {
  Bar: Navbar,
  Drawer: NavDrawer,
};
