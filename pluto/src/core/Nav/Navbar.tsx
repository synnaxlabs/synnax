// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, FunctionComponent, useContext } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "@/core/Space";
import { Location, Position, swapLoc, swapDir, locToDir, dirToDim } from "@/spatial";

import "./Navbar.css";

export interface NavbarProps
  extends Omit<SpaceProps<HTMLDivElement>, "direction" | "size"> {
  location?: Location;
  size?: string | number;
}

const NavbarContext = createContext<Location>("left");

const CoreNavbar = ({
  location = "left",
  size = 60,
  className,
  style,
  ...props
}: NavbarProps): JSX.Element => {
  const dir = locToDir(location);
  return (
    <NavbarContext.Provider value={location}>
      <Space
        className={clsx(
          "pluto-navbar",
          `pluto-bordered--${swapLoc(location)}`,
          `pluto-navbar--${dir}`,
          className
        )}
        direction={swapDir(dir)}
        style={{
          [dirToDim(dir)]: size,
          ...style,
        }}
        align="center"
        empty
        {...props}
      />
    </NavbarContext.Provider>
  );
};

export interface NavbarContentProps extends SpaceProps<HTMLDivElement> {
  bordered?: boolean;
  className?: string;
}

const contentFactory =
  (pos: Position | ""): FunctionComponent<NavbarContentProps> =>
  // eslint-disable-next-line react/display-name
  ({ bordered = true, className, ...props }: NavbarContentProps): JSX.Element =>
    (
      <Space
        className={clsx(
          "pluto-navbar__content",
          `pluto-navbar__content--${pos}`,
          bordered && "pluto-navbar__content--bordered",
          className
        )}
        direction={swapDir(locToDir(useContext(NavbarContext)))}
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
  Context: typeof NavbarContext;
}

export const Navbar = CoreNavbar as NavbarType;

Navbar.Start = NavbarStart;
Navbar.Center = NavbarCenter;
Navbar.End = NavbarEnd;
Navbar.Content = NavbarContent;
