// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav as PNav } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Bar } from "@/nav/Bar";
import { type NavDrawerItem } from "@/nav/item";
import { Menu } from "@/nav/Menu";
import { useSelectBottom, useSelectLeft } from "@/nav/selectors";
import {
  type LeftItem,
  pinLeft,
  selectBottom,
  selectLeft,
  setBottomVisible,
  startBottomHover,
  startLeftHover,
  stopBottomHover,
  stopLeftHover,
  toggleBottom,
  toggleLeft,
} from "@/nav/slice";

export interface LeftBarProps {
  items: NavDrawerItem[];
  bottomItem: NavDrawerItem;
  enabled?: boolean;
  children?: ReactNode;
}

export const LeftBar = ({
  items,
  bottomItem,
  enabled = true,
  children,
}: LeftBarProps): ReactElement => {
  const dispatch = useDispatch();
  const { selected } = useSelectLeft();
  const { visible } = useSelectBottom();

  const onSelectLeft = useCallback(
    (key: string) => dispatch(selectLeft(key as LeftItem)),
    [dispatch],
  );
  const onToggleLeft = useCallback(
    (key: string) => dispatch(toggleLeft(key as LeftItem)),
    [dispatch],
  );
  const onPinLeft = useCallback(
    (key: string) => dispatch(pinLeft(key as LeftItem)),
    [dispatch],
  );
  const onStartLeftHover = useCallback(
    (key: string) => dispatch(startLeftHover(key as LeftItem)),
    [dispatch],
  );
  const onStopLeftHover = useCallback(() => dispatch(stopLeftHover()), [dispatch]);

  const onSelectBottom = useCallback(() => dispatch(selectBottom()), [dispatch]);
  const onToggleBottom = useCallback(() => dispatch(toggleBottom()), [dispatch]);
  const onPinBottom = useCallback(() => dispatch(setBottomVisible(true)), [dispatch]);
  const onStartBottomHover = useCallback(
    () => dispatch(startBottomHover()),
    [dispatch],
  );
  const onStopBottomHover = useCallback(() => dispatch(stopBottomHover()), [dispatch]);

  return (
    <Bar location="left" size="8rem">
      <PNav.Bar.Content align="center">
        {children}
        <Menu
          items={items}
          activeKey={selected}
          enabled={enabled}
          onSelect={onSelectLeft}
          onToggle={onToggleLeft}
          onPin={onPinLeft}
          onStartHover={onStartLeftHover}
          onStopHover={onStopLeftHover}
        />
      </PNav.Bar.Content>
      <PNav.Bar.End bordered>
        <Menu
          items={[bottomItem]}
          activeKey={visible ? bottomItem.key : undefined}
          enabled={enabled}
          onSelect={onSelectBottom}
          onToggle={onToggleBottom}
          onPin={onPinBottom}
          onStartHover={onStartBottomHover}
          onStopHover={onStopBottomHover}
        />
      </PNav.Bar.End>
    </Bar>
  );
};
