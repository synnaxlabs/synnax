// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent } from "react";

import {
  Location,
  Position,
  swapLoc,
  swapDir,
  locToDir,
  dirToDim,
} from "@synnaxlabs/x";

import { Space, SpaceProps } from "@/core/Space";
import { CSS } from "@/css";

import "./Navbar.css";

export interface NavbarProps extends Omit<SpaceProps, "direction" | "size" | "ref"> {
  location?: Location;
  size?: string | number;
}

const CoreNavbar = ({
  location = "left",
  size = "9rem",
  className,
  style,
  ...props
}: NavbarProps): JSX.Element => {
  const dir = locToDir(location);
  const swappedDir = swapDir(locToDir(location));
  return (
    <Space
      className={CSS(
        CSS.B("navbar"),
        CSS.bordered(swapLoc(location)),
        CSS.dir(swappedDir),
        className
      )}
      direction={swappedDir}
      style={{
        [dirToDim(dir)]: size,
        ...style,
      }}
      align="center"
      empty
      {...props}
    />
  );
};

export interface NavbarContentProps extends Omit<SpaceProps<"div">, "ref"> {
  bordered?: boolean;
  className?: string;
}

const contentFactory =
  (pos: Position | ""): FunctionComponent<NavbarContentProps> =>
  // eslint-disable-next-line react/display-name
  ({ bordered = true, className, ...props }: NavbarContentProps): JSX.Element =>
    (
      <Space
        className={CSS(
          CSS.BE("navbar", "content"),
          CSS.pos(pos),
          CSS.bordered(pos),
          className
        )}
        align="center"
        {...props}
      />
    );

type CoreNavbarType = typeof CoreNavbar;

const NavbarStart = contentFactory("start");
NavbarStart.displayName = "NavbarStart";
const NavbarEnd = contentFactory("end");
NavbarEnd.displayName = "NavbarEnd";
const NavbarCenter = contentFactory("center");
NavbarCenter.displayName = "NavbarCenter";
const NavbarContent = contentFactory("");
NavbarContent.displayName = "NavbarContent";

export interface NavbarType extends CoreNavbarType {
  Start: typeof NavbarStart;
  Center: typeof NavbarCenter;
  End: typeof NavbarEnd;
  Content: typeof NavbarContent;
}

export const Navbar = CoreNavbar as NavbarType;

Navbar.Start = NavbarStart;
Navbar.Center = NavbarCenter;
Navbar.End = NavbarEnd;
Navbar.Content = NavbarContent;
