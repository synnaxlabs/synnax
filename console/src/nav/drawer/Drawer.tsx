// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav } from "@synnaxlabs/pluto";
import { box, direction, type location, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Item } from "@/nav/item";

const mouseLeaveBy =
  (threshold: xy.XY, onLeave: (e: MouseEvent) => void) => (e: React.MouseEvent) => {
    const content = (e.target as HTMLElement).closest(".pluto-nav-drawer");
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
        onLeave(e);
      } else if (e.buttons != 0) window.removeEventListener("mousemove", lis);
    };
    window.addEventListener("mousemove", lis);
  };

const LONG_AXIS_THRESHOLD = 36;
const SHORT_AXIS_THRESHOLD = 24;

const X_THRESHOLD = xy.construct(LONG_AXIS_THRESHOLD, SHORT_AXIS_THRESHOLD);

interface DrawerProps {
  location: location.Location;
  activeItem?: Item.Item;
  hover: boolean;
  onResize: (size: number) => void;
  onCollapse: () => void;
  onStopHover: () => void;
}

export const Drawer = ({
  location: loc,
  activeItem,
  hover,
  onResize,
  onCollapse,
  onStopHover,
}: DrawerProps): ReactElement => (
  <Nav.Drawer
    location={loc}
    className={CSS(CSS.BE("nav", "drawer"), hover && CSS.M("hover"))}
    activeItem={activeItem}
    onResize={onResize}
    onMouseLeave={mouseLeaveBy(
      direction.construct(loc) === "y" ? xy.swap(X_THRESHOLD) : X_THRESHOLD,
      onStopHover,
    )}
    eraseEnabled={activeItem != null && !hover}
    onCollapse={onCollapse}
    background={0}
    rounded={1}
    bordered
    borderColor={5}
  />
);
