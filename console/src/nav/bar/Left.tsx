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
import { type ReactElement, type ReactNode, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Bar } from "@/nav/bar/Bar";
import { Menu } from "@/nav/bar/Menu";
import { Item } from "@/nav/item";
import { Session } from "@/nav/session";

export interface Left {
  enabled?: boolean;
  children?: ReactNode;
}

export const Left = ({ enabled = true, children }: Left): ReactElement => {
  const dispatch = useDispatch();
  const { selected } = Session.useSelectLeft();
  const { visible } = Session.useSelectBottom();
  const bottomItem = Item.useBottom();
  const bottomItems = useMemo(() => [bottomItem], [bottomItem]);
  const leftItems = Item.useLeft();

  const onSelectLeft = useCallback(
    (key: string) => dispatch(Session.selectLeft({ key })),
    [dispatch],
  );
  const onToggleLeft = useCallback(
    (key: string) => dispatch(Session.toggleLeft({ key })),
    [dispatch],
  );
  const onPinLeft = useCallback(
    (key: string) => dispatch(Session.pinLeft({ key })),
    [dispatch],
  );
  const onStartLeftHover = useCallback(
    (key: string) => dispatch(Session.startLeftHover({ key })),
    [dispatch],
  );
  const onStopLeftHover = useCallback(
    () => dispatch(Session.stopLeftHover({})),
    [dispatch],
  );

  const onSelectBottom = useCallback(
    () => dispatch(Session.selectBottom({})),
    [dispatch],
  );
  const onToggleBottom = useCallback(
    () => dispatch(Session.toggleBottom({})),
    [dispatch],
  );
  const onPinBottom = useCallback(
    () => dispatch(Session.setBottomVisible({ visible: true })),
    [dispatch],
  );
  const onStartBottomHover = useCallback(
    () => dispatch(Session.startBottomHover({})),
    [dispatch],
  );
  const onStopBottomHover = useCallback(
    () => dispatch(Session.stopBottomHover({})),
    [dispatch],
  );

  return (
    <Bar location="left" size="8rem">
      <PNav.Bar.Content align="center">
        {children}
        <Menu
          items={leftItems}
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
          items={bottomItems}
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
