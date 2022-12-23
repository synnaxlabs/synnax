import { Nav } from "@synnaxlabs/pluto";

import { NAV_SIZES } from "./constants";

/**
 * NavTop is the top navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavTop = (): JSX.Element => (
  <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top} />
);
