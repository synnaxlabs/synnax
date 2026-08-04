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

import { useBottomActions } from "@/app/nav/bar/bottom";
import { Palette } from "@/app/palette";
import { Toolbars } from "@/app/toolbars";
import { Project } from "@/feature/project";
import { Nav as PlatformNav } from "@/platform/nav";
import { Session } from "@/session";

const BottomMenu = () => {
  const visible = Session.Nav.useSelectBottomVisible();
  const { select, toggle, pin, startHover, stopHover } = useBottomActions();
  return (
    <PlatformNav.Menu
      items={Toolbars.BOTTOM}
      selected={visible ? Toolbars.BOTTOM.key : undefined}
      onSelect={select}
      onToggle={toggle}
      onPin={pin}
      onStartHover={startHover}
      onStopHover={stopHover}
    />
  );
};

const LeftMenu = () => {
  const dispatch = Session.useDispatch();
  const selected = Session.Nav.useSelectLeftSelected();
  const handleSelect = useCallback(
    (key: string) => dispatch(Session.Nav.selectLeft({ key })),
    [dispatch],
  );
  const handleToggle = useCallback(
    (key: string) => dispatch(Session.Nav.toggleLeft({ key })),
    [dispatch],
  );
  const handlePin = useCallback(
    (key: string) => dispatch(Session.Nav.pinLeft({ key })),
    [dispatch],
  );
  const handleStartover = useCallback(
    (key: string) => dispatch(Session.Nav.startLeftHover({ key })),
    [dispatch],
  );
  const handleStopHover = useCallback(
    () => dispatch(Session.Nav.stopLeftHover({})),
    [dispatch],
  );
  return (
    <PlatformNav.Menu
      items={Toolbars.LEFT}
      selected={selected}
      onSelect={handleSelect}
      onToggle={handleToggle}
      onPin={handlePin}
      onStartHover={handleStartover}
      onStopHover={handleStopHover}
    />
  );
};

export const Left = (): ReactElement => (
  <PlatformNav.Bar location="left" size="7.5rem">
    <Nav.Bar.Start bordered align="center">
      <Project.Selector />
    </Nav.Bar.Start>
    <Nav.Bar.Content align="center">
      <Palette.Palette />
      <LeftMenu />
    </Nav.Bar.Content>
    <Nav.Bar.End bordered>
      <BottomMenu />
    </Nav.Bar.End>
  </PlatformNav.Bar>
);
