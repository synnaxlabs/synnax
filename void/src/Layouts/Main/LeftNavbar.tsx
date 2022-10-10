import { NavDrawer, Navbar, ThemeSwitch } from "@synnaxlabs/pluto";
import ClustersItem from "../../cluster/ConfigureClusters";
import Logo from "../../lib/Logo/Logo";
import ResourcesItem from "../../resources/ResourcesTree";
import "./LeftNavbar.css";

export default function LeftNavbar() {
  return (
    <NavDrawer location="left" size={48} items={[ClustersItem, ResourcesItem]}>
      <Navbar.Start className="sidebar-left__start" bordered>
        <Logo style={{ width: "100%" }} />
      </Navbar.Start>
      <Navbar.End className="sidebar-left__end" bordered>
        <ThemeSwitch />
      </Navbar.End>
    </NavDrawer>
  );
}
