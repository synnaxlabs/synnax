// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav } from "@synnaxlabs/pluto";
import { box, location, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { NAV_DRAWER_ITEMS } from "@/layouts/nav/drawerItems";

export interface DrawerProps {
  location: Layout.NavDrawerLocation;
}

const mouseLeaveBy =
  (threshold: xy.XY, onLeave: (e: MouseEvent) => void) => (e: React.MouseEvent) => {
    const content = (e.target as HTMLElement).closest(".pluto-navdrawer__content");
    if (content == null) return;
    let b = box.construct(content);
    b = box.translate(b, xy.scale(threshold, -1));
    b = box.resize(b, {
      width: box.width(b) + threshold.x * 2,
      height: box.height(b) + threshold.y * 2,
    });
    const lis = (e: MouseEvent) => {
      const pos = xy.construct(e);
      if (e.buttons != 0) window.removeEventListener("mousemove", lis);
      else if (!box.contains(b, pos)) {
        window.removeEventListener("mousemove", lis);
        onLeave(e);
      }
    };
    window.addEventListener("mousemove", lis);
  };

export const Drawer = ({ location: loc }: DrawerProps): ReactElement => {
  const { activeItem, onResize, onSelect, hover, onStopHover } = Layout.useNavDrawer(
    loc,
    NAV_DRAWER_ITEMS,
  );
  return (
    <Nav.Drawer
      location={loc}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", location.direction(loc)),
        CSS.BM("main-nav-drawer", loc),
        hover && CSS.M("hover"),
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
      onMouseLeave={mouseLeaveBy(xy.construct(120, 20), onStopHover)}
    />
  );
};
