// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Items } from "@/layered/app/nav/items";
import { Session } from "@/layered/session";
import { View } from "@/layered/view";

export const Left = (): ReactElement => {
  const dispatch = useDispatch();
  const { selected } = Session.Nav.useSelectLeft();
  const { visible } = Session.Nav.useSelectBottom();
  const onSelectLeft = useCallback(
    (key: string) => dispatch(Session.Nav.selectLeft({ key })),
    [dispatch],
  );
  const onToggleLeft = useCallback(
    (key: string) => dispatch(Session.Nav.toggleLeft({ key })),
    [dispatch],
  );
  const onPinLeft = useCallback(
    (key: string) => dispatch(Session.Nav.pinLeft({ key })),
    [dispatch],
  );
  const onStartLeftHover = useCallback(
    (key: string) => dispatch(Session.Nav.startLeftHover({ key })),
    [dispatch],
  );
  const onStopLeftHover = useCallback(
    () => dispatch(Session.Nav.stopLeftHover({})),
    [dispatch],
  );

  const onSelectBottom = useCallback(
    () => dispatch(Session.Nav.selectBottom({})),
    [dispatch],
  );
  const onToggleBottom = useCallback(
    () => dispatch(Session.Nav.toggleBottom({})),
    [dispatch],
  );
  const onPinBottom = useCallback(
    () => dispatch(Session.Nav.setBottomVisible({ visible: true })),
    [dispatch],
  );
  const onStartBottomHover = useCallback(
    () => dispatch(Session.Nav.startBottomHover({})),
    [dispatch],
  );
  const onStopBottomHover = useCallback(
    () => dispatch(Session.Nav.stopBottomHover({})),
    [dispatch],
  );

  return (
    <View.Nav.Bar location="left" size="8rem">
      <Nav.Bar.Content align="center">
        <View.Nav.Menu
          items={Items.LEFT}
          activeKey={selected}
          onSelect={onSelectLeft}
          onToggle={onToggleLeft}
          onPin={onPinLeft}
          onStartHover={onStartLeftHover}
          onStopHover={onStopLeftHover}
        />
      </Nav.Bar.Content>
      <Nav.Bar.End bordered>
        <View.Nav.Menu
          items={Items.BOTTOM}
          activeKey={visible ? Items.BOTTOM.key : undefined}
          onSelect={onSelectBottom}
          onToggle={onToggleBottom}
          onPin={onPinBottom}
          onStartHover={onStartBottomHover}
          onStopHover={onStopBottomHover}
        />
      </Nav.Bar.End>
    </View.Nav.Bar>
  );
};
