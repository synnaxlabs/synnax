// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav, useDebouncedCallback } from "@synnaxlabs/pluto";
import { box, direction, type location, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { type DrawerItem } from "@/nav/item";
import { useSelectBottom, useSelectLeft } from "@/nav/selectors";
import {
  resizeBottom,
  resizeLeft,
  selectBottom,
  selectLeft,
  stopBottomHover,
  stopLeftHover,
} from "@/nav/slice";

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
  activeItem?: DrawerItem;
  hover: boolean;
  onResize: (size: number) => void;
  onCollapse: () => void;
  onStopHover: () => void;
}

const Drawer = ({
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

export interface LeftDrawerProps {
  items: DrawerItem[];
}

export const LeftDrawer = ({ items }: LeftDrawerProps): ReactElement => {
  const { selected, hover, size } = useSelectLeft();
  const dispatch = useDispatch();
  const activeItem = useMemo(() => {
    const item = items.find((i) => i.key === selected);
    if (item == null) return undefined;
    return size != null ? { ...item, initialSize: size } : item;
  }, [items, selected, size]);
  const onResize = useDebouncedCallback(
    (s: number) => dispatch(resizeLeft(s)),
    TimeSpan.milliseconds(100),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(stopLeftHover());
    else if (selected != null) dispatch(selectLeft(selected));
  }, [dispatch, hover, selected]);
  const onStopHover = useCallback(() => dispatch(stopLeftHover()), [dispatch]);
  return (
    <Drawer
      location="left"
      activeItem={activeItem}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    />
  );
};

export interface BottomDrawerProps {
  item: DrawerItem;
}

export const BottomDrawer = ({ item }: BottomDrawerProps): ReactElement => {
  const { visible, hover, size } = useSelectBottom();
  const dispatch = useDispatch();
  const activeItem = useMemo(() => {
    if (!visible) return undefined;
    return size != null ? { ...item, initialSize: size } : item;
  }, [item, visible, size]);
  const onResize = useDebouncedCallback(
    (s: number) => dispatch(resizeBottom(s)),
    TimeSpan.milliseconds(100),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(stopBottomHover());
    else dispatch(selectBottom());
  }, [dispatch, hover]);
  const onStopHover = useCallback(() => dispatch(stopBottomHover()), [dispatch]);
  return (
    <Drawer
      location="bottom"
      activeItem={activeItem}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    />
  );
};
