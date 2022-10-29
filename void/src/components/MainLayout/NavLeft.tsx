import { Nav, Theming } from "@synnaxlabs/pluto";
import { Logo } from "@/components";
import "./NavLeft.css";
import { ClusterToolBar } from "@/features/cluster";
import { ResourcesToolBar } from "@/features/resources";

export const NavLeft = () => (
  <Nav.Drawer
    location="left"
    size={48}
    items={[ClusterToolBar, ResourcesToolBar]}
  >
    <Nav.Bar.Start className="void-main-layout__nav-left__start" bordered>
      <Logo style={{ width: "100%" }} color="auto" />
    </Nav.Bar.Start>
    <Nav.Bar.End className="void-main-layout__nav-left__end" bordered>
      <Theming.Switch />
    </Nav.Bar.End>
  </Nav.Drawer>
);
