// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent, ReactElement } from "react";

import { Location, LooseLocationT, CrudePosition } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";

import "@/nav/Bar.css";

export interface BarProps extends Omit<Align.SpaceProps, "direction" | "size" | "ref"> {
  location?: LooseLocationT;
  size?: string | number;
}

const CoreBar = ({
  location: location_ = "left",
  size = "9rem",
  className,
  style,
  ...props
}: BarProps): ReactElement => {
  const location = new Location(location_);
  return (
    <Align.Space
      className={CSS(
        CSS.B("navbar"),
        CSS.bordered(location.inverse.crude),
        CSS.dir(location.direction.inverse),
        CSS.loc(location),
        className
      )}
      direction={location.direction.inverse}
      style={{
        [location.direction.dimension]: size,
        ...style,
      }}
      align="center"
      empty
      {...props}
    />
  );
};

export interface BarContentProps extends Omit<Align.SpaceProps<"div">, "ref"> {
  bordered?: boolean;
  className?: string;
}

const contentFactory =
  (pos: CrudePosition | ""): FunctionComponent<BarContentProps> =>
  // eslint-disable-next-line react/display-name
  ({ bordered = false, className, ...props }: BarContentProps): ReactElement =>
    (
      <Align.Space
        className={CSS(
          CSS.BE("navbar", "content"),
          CSS.pos(pos),
          pos !== "" && bordered && CSS.bordered(pos),
          className
        )}
        align="center"
        {...props}
      />
    );

type CoreNavbarType = typeof CoreBar;

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

export const Navbar = CoreBar as NavbarType;

Navbar.Start = NavbarStart;
Navbar.Center = NavbarCenter;
Navbar.End = NavbarEnd;
Navbar.Content = NavbarContent;
