import { Nav } from "@synnaxlabs/pluto";

import { WorkspaceToolbar } from "@/features/workspace";

export const NavRight = (): JSX.Element => (
  <Nav.Drawer location="right" size={48} items={[WorkspaceToolbar]} />
);
