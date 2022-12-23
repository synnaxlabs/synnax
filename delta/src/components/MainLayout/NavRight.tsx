import { Nav } from "@synnaxlabs/pluto";

import { NAV_SIZES } from "./constants";

import { WorkspaceToolbar } from "@/features/workspace";

/**
 * NavRight is the right navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavRight = (): JSX.Element => (
  <Nav.Drawer location="right" size={NAV_SIZES.side} items={[WorkspaceToolbar]} />
);
