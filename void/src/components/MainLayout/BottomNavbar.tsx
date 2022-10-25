import {
  ActiveClusterBadge,
  ActiveConnectionBadge,
} from "@/features/cluster/components/ConnectionStatus/ConnectionStatus";
import { Nav, Divider } from "@synnaxlabs/pluto";

export default function BottomNavbar() {
  return (
    <Nav.Bar location="bottom" size={32}>
      <Nav.Bar.End className="void-main-layout__bottom-navbar">
        <Divider direction="vertical" />
        <ActiveClusterBadge />
        <Divider direction="vertical" />
        <ActiveConnectionBadge />
      </Nav.Bar.End>
    </Nav.Bar>
  );
}
