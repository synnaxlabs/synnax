import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface BarProps extends Nav.BarProps {}

export const Bar = ({ className, ...rest }: BarProps): ReactElement => (
  <Nav.Bar className={CSS(CSS.BE("nav", "bar"), className)} {...rest} />
);
