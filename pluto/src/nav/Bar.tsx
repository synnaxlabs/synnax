// Copyright 2026 Synnax Labs, Inc.
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

import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface BarProps extends Omit<Flex.BoxProps, "direction" | "size" | "ref"> {
  location?: location.Crude;
  size?: string | number;
  bordered?: boolean;
}

const BaseBar = ({
  location: location_ = "left",
  size = "9rem",
  className,
  style,
  bordered = false,
  ...rest
}: BarProps): ReactElement => {
  const loc = location.construct(location_);
  const dir = location.direction(loc);
  const oppositeDir = direction.swap(dir);
  return (
    <Flex.Box
      className={CSS(
        CSS.B("navbar"),
        bordered && CSS.bordered(location.swap(loc)),
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
      {...rest}
    />
  );
};

export interface BarContentProps extends Omit<Flex.BoxProps<"div">, "ref"> {
  bordered?: boolean;
  className?: string;
}

const contentFactory =
  (
    pos: spatial.Alignment | "" | "absolute-center",
  ): FunctionComponent<BarContentProps> =>
  // eslint-disable-next-line react/display-name
  ({ bordered = false, className, ...rest }: BarContentProps): ReactElement => (
    <Flex.Box
      className={CSS(
        CSS.BE("navbar", "content"),
        pos === "absolute-center" ? CSS.M(pos) : CSS.align(pos),
        pos !== "" && bordered && CSS.bordered(pos),
        className,
      )}
      align="center"
      {...rest}
    />
  );

type BaseBarType = typeof BaseBar;

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

export interface BarType extends BaseBarType {
  Start: typeof Start;
  Center: typeof Center;
  End: typeof End;
  AbsoluteCenter: typeof AbsoluteCenter;
  Content: typeof Content;
}

export const Bar = BaseBar as BarType;

Bar.Start = Start;
Bar.Center = Center;
Bar.End = End;
Bar.AbsoluteCenter = AbsoluteCenter;
Bar.Content = Content;
