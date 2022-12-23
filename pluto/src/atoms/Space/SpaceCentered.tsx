import clsx from "clsx";

import { Space, SpaceProps } from "./Space";

import "./SpaceCentered.css";

export interface SpaceCenteredProps extends Omit<SpaceProps, "justify" | "align"> {}

export const SpaceCentered = ({ className, ...props }: SpaceProps): JSX.Element => (
  <Space
    {...props}
    justify="center"
    align="center"
    className={clsx("pluto-space-centered", className)}
  />
);
