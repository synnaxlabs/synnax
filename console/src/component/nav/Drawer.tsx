// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/component/nav/Nav.css";

import { Eraser, Errors, Nav, type Resize } from "@synnaxlabs/pluto";
import { box, direction, type location, xy } from "@synnaxlabs/x";
import {
  type MouseEvent as ReactMouseEvent,
  type ReactElement,
  useCallback,
} from "react";

import { CSS } from "@/component/css";

const X_THRESHOLD = { x: 36, y: 24 };
const Y_THRESHOLD = xy.swap(X_THRESHOLD);

interface DrawerProps extends Omit<Nav.DrawerProps, "onResize"> {
  location: location.Location;
  hover: boolean;
  onStopHover: () => void;
}

const CLASS = CSS.BE("nav", "drawer");

export const Drawer = ({
  location: loc,
  hover,
  collapsed,
  onStopHover,
  children,
  ...rest
}: DrawerProps): ReactElement => {
  const { erase } = Eraser.use({ enabled: !hover });
  const handleResize = useCallback(
    (_: number, { box }: Resize.HandlerExtra) => erase(box),
    [erase],
  );
  const handleMouseLeave = useCallback(
    (e: ReactMouseEvent) => {
      const threshold = direction.isY(loc) ? Y_THRESHOLD : X_THRESHOLD;
      const content = (e.target as HTMLElement).closest(`.${CLASS}`);
      if (content == null) return;
      let b = box.construct(content);
      b = box.translate(b, xy.scale(threshold, -1));
      b = box.resize(b, {
        width: box.width(b) + threshold.x * 2,
        height: box.height(b) + threshold.y * 2,
      });
      const lis = (e: MouseEvent) => {
        const pos = xy.construct(e);
        if (!box.contains(b, pos)) {
          window.removeEventListener("mousemove", lis);
          onStopHover();
        } else if (e.buttons != 0) window.removeEventListener("mousemove", lis);
      };
      window.addEventListener("mousemove", lis);
    },
    [loc, onStopHover],
  );
  return (
    <Nav.Drawer
      location={loc}
      className={CSS(CLASS, hover && CSS.M("hover"))}
      collapsed={collapsed}
      onMouseLeave={handleMouseLeave}
      onResize={handleResize}
      background={0}
      rounded={1}
      bordered
      borderColor={5}
      {...rest}
    >
      <Errors.Boundary>{collapsed ? null : children}</Errors.Boundary>
    </Nav.Drawer>
  );
};
