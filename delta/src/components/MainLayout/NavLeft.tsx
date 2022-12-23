import { Nav, Theming } from "@synnaxlabs/pluto";

import { NAV_SIZES } from "./constants";

import { Logo } from "@/components";
import { ClusterToolbar } from "@/features/cluster";
import { ResourcesToolbar } from "@/features/resources";

import "./NavLeft.css";

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): JSX.Element => (
  <Nav.Drawer
    location="left"
    size={NAV_SIZES.side}
    items={[ClusterToolbar, ResourcesToolbar]}
  >
    <Nav.Bar.Start className="delta-main-nav-left__start" bordered>
      <Logo className="delta-main-nav-left__logo" />
    </Nav.Bar.Start>
    <Nav.Bar.End className="delta-main-nav-left__end" bordered>
      <Theming.Switch />
    </Nav.Bar.End>
  </Nav.Drawer>
);
