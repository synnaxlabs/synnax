import { Nav } from "@synnaxlabs/pluto";

import { WorkspaceToolBar } from "@/features/workspace";

export const NavRight = (): JSX.Element => (
  <Nav.Drawer location="right" size={48} items={[WorkspaceToolBar]} />
);
