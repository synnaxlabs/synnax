import { Navbar } from "./Navbar";
import { Navdrawer, useNavDrawer } from "./Navdrawer";
import { NavMenu } from "./NavMenu";
export type { NavbarProps, NavbarContextValue } from "./Navbar";
export type {
  NavDrawerProps,
  NavDrawerContent,
  UseNavDrawerReturn,
  UseNavDrawerProps,
  NavDrawerItem,
} from "./Navdrawer";
export type { NavMenuItem } from "./NavMenu";

export const Nav = {
  Bar: Navbar,
  Drawer: Navdrawer,
  useDrawer: useNavDrawer,
  Menu: NavMenu,
};
