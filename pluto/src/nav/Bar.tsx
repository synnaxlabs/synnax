// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/nav/Bar.css";

import { direction, location, type spatial } from "@synnaxlabs/x";
import { type FunctionComponent, type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";

export interface BarProps extends Omit<Align.SpaceProps, "direction" | "size" | "ref"> {
  location?: location.Crude;
  size?: string | number;
}

const CoreBar = ({
  location: location_ = "left",
  size = "9rem",
  className,
  style,
  ...props
}: BarProps): ReactElement => {
  const loc = location.construct(location_);
  const dir = location.direction(loc);
  const oppositeDir = direction.swap(dir);
  return (
    <Align.Space
      className={CSS(
        CSS.B("navbar"),
        CSS.bordered(location.swap(loc)),
        CSS.dir(oppositeDir),
        CSS.loc(loc),
        className,
      )}
      direction={oppositeDir}
      style={{
        [direction.dimension(dir)]: size,
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
  (
    pos: spatial.Alignment | "" | "absolute-center",
  ): FunctionComponent<BarContentProps> =>
  // eslint-disable-next-line react/display-name
  ({ bordered = false, className, ...props }: BarContentProps): ReactElement => (
    <Align.Space
      className={CSS(
        CSS.BE("navbar", "content"),
        pos === "absolute-center" ? CSS.M(pos) : CSS.align(pos),
        pos !== "" && bordered && CSS.bordered(pos),
        className,
      )}
      align="center"
      {...props}
    />
  );

type CoreBarType = typeof CoreBar;

const Start = contentFactory("start");
Start.displayName = "NavbarStart";
const End = contentFactory("end");
End.displayName = "NavbarEnd";
const Center = contentFactory("center");
Center.displayName = "NavbarCenter";
const Content = contentFactory("");
Content.displayName = "NavbarContent";
const AbsoluteCenter = contentFactory("absolute-center");
AbsoluteCenter.displayName = "NavbarAbsoluteCenter";

export interface BarType extends CoreBarType {
  Start: typeof Start;
  Center: typeof Center;
  End: typeof End;
  AbsoluteCenter: typeof AbsoluteCenter;
  Content: typeof Content;
}

export const Bar = CoreBar as BarType;

Bar.Start = Start;
Bar.Center = Center;
Bar.End = End;
Bar.AbsoluteCenter = AbsoluteCenter;
Bar.Content = Content;
