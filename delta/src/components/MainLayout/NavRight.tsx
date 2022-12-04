import { Nav } from "@synnaxlabs/pluto";
import { WorkspaceToolBar } from "@/features/workspace";

export const NavRight = () => {
	return <Nav.Drawer location="right" size={48} items={[WorkspaceToolBar]} />;
};
