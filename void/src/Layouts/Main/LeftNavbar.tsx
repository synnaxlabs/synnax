import { Nav, Theme } from "@synnaxlabs/pluto";
import ClustersItem from "../../cluster/ConfigureClusters";
import Logo from "../../lib/Logo/Logo";
import ResourcesItem from "../../resources/ResourcesTree";
import "./LeftNavbar.css";

export default function LeftNavbar() {
  return (
    <Nav.Drawer location="left" size={48} items={[ClustersItem, ResourcesItem]}>
      <Nav.Bar.Start className="sidebar-left__start" bordered>
        <Logo style={{ width: "100%" }} />
      </Nav.Bar.Start>
      <Nav.Bar.End className="sidebar-left__end" bordered>
        <Theme.Switch />
      </Nav.Bar.End>
    </Nav.Drawer>
  );
}
