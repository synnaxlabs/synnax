import { ActiveClusterBadge, ActiveConnectionBadge } from "@/features/cluster";
import { Nav, Divider } from "@synnaxlabs/pluto";
import "./NavBottom.css";

export const NavBottom = () => (
	<Nav.Bar location="bottom" size={32}>
		<Nav.Bar.End className="delta-main-layout__nav-bottom__end">
			<Divider direction="vertical" />
			<ActiveClusterBadge />
			<Divider direction="vertical" />
			<ActiveConnectionBadge />
		</Nav.Bar.End>
	</Nav.Bar>
);
