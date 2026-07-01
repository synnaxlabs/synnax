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

import { Items } from "@/app/nav/items";
import { Nav as ComponentNav } from "@/platform/nav";
import { Palette } from "@/platform/palette";
import { Session } from "@/session";

const BottomMenu = () => {
  const dispatch = Session.useDispatch();
  const visible = Session.Nav.useSelectBottomVisible();
  const handleSelect = useCallback(
    () => dispatch(Session.Nav.selectBottom({})),
    [dispatch],
  );
  const handleToggle = useCallback(
    () => dispatch(Session.Nav.toggleBottom({})),
    [dispatch],
  );
  const handlePin = useCallback(() => dispatch(Session.Nav.showBottom({})), [dispatch]);
  const handleStartHover = useCallback(
    () => dispatch(Session.Nav.startBottomHover({})),
    [dispatch],
  );
  const handlestopHover = useCallback(
    () => dispatch(Session.Nav.stopBottomHover({})),
    [dispatch],
  );
  return (
    <ComponentNav.Menu
      items={Items.BOTTOM}
      selected={visible ? Items.BOTTOM.key : undefined}
      onSelect={handleSelect}
      onToggle={handleToggle}
      onPin={handlePin}
      onStartHover={handleStartHover}
      onStopHover={handlestopHover}
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
    <ComponentNav.Menu
      items={Items.LEFT}
      selected={selected}
      onSelect={handleSelect}
      onToggle={handleToggle}
      onPin={handlePin}
      onStartHover={handleStartover}
      onStopHover={handleStopHover}
    />
  );
};

const PALETTE_TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

export const Left = (): ReactElement => (
  <ComponentNav.Bar location="left" size="8rem">
    <Nav.Bar.Start bordered align="center">
      <Palette.Palette commandSymbol=">" triggerConfig={PALETTE_TRIGGER_CONFIG} />
    </Nav.Bar.Start>
    <Nav.Bar.Content align="center">
      <LeftMenu />
    </Nav.Bar.Content>
    <Nav.Bar.End bordered>
      <BottomMenu />
    </Nav.Bar.End>
  </ComponentNav.Bar>
);
