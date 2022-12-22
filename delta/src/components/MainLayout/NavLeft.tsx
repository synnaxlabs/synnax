import { Nav, Theming } from "@synnaxlabs/pluto";

import { Logo } from "@/components";
import "./NavLeft.css";
import { ClusterToolbar } from "@/features/cluster";
import { ResourcesToolbar } from "@/features/resources";

export const NavLeft = (): JSX.Element => (
  <Nav.Drawer location="left" size={48} items={[ClusterToolbar, ResourcesToolbar]}>
    <Nav.Bar.Start className="delta-main-layout__nav-left__start" bordered>
      <Logo style={{ width: "100%" }} color="auto" />
    </Nav.Bar.Start>
    <Nav.Bar.End className="delta-main-layout__nav-left__end" bordered>
      <Theming.Switch />
    </Nav.Bar.End>
  </Nav.Drawer>
);
