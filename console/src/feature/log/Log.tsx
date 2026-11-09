// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log as Base, Panel as PPanel } from "@synnaxlabs/pluto";
import { TimeRange } from "@synnaxlabs/x";

import { Controls } from "@/feature/log/Controls";
import { ContextMenu } from "@/platform/context-menu";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

const Internal: Panel.Content = () => {
  const key = Base.useKey();
  const dispatch = Session.useDispatch();
  const visible = Session.Panel.useSelectIsTabVisible();
  const hasChannels = Base.useChannelKeys().length > 0;
  const hold = Session.Log.useSelectHold();
  const position = Session.Log.useSelectPosition();
  const range = Session.Log.useSelectRange();

  const handleDoubleClick = () => {
    dispatch(Session.Nav.showBottom({}));
  };

  const handleConfigureChannels = () => {
    dispatch(Session.Log.setSelectedToolbarTab({ key, tab: "channels" }));
    handleDoubleClick();
  };

  const handleHold = (hold: boolean) => dispatch(Session.Log.setHold({ key, hold }));
  const handleLeave = (position: string) =>
    dispatch(Session.Log.setPosition({ key, position }));
  const handleDropRange = ({ start, end }: TimeRange) =>
    dispatch(
      Session.Log.setRange({
        key,
        range: { start: start.valueOf().toString(), end: end.valueOf().toString() },
      }),
    );

  return (
    <Base.Log
      onDoubleClick={handleDoubleClick}
      hold={hold}
      onHold={handleHold}
      position={position}
      onLeave={handleLeave}
      range={
        range == null
          ? undefined
          : new TimeRange(BigInt(range.start), BigInt(range.end))
      }
      onDropRange={handleDropRange}
      extraContextMenuItems={EXTRA_CONTEXT_MENU_ITEMS}
      emptyContent={
        <Empty.Action
          message={hasChannels ? "No data received yet" : "No channels configured"}
          action={hasChannels ? undefined : "Configure channels"}
          onClick={hasChannels ? handleDoubleClick : handleConfigureChannels}
        />
      }
      visible={visible}
    >
      <Controls />
    </Base.Log>
  );
};

export const Log: Panel.Content = () => {
  const { key } = PPanel.useTabResource();
  return (
    <Base.Suspended logKey={key}>
      <Internal />
    </Base.Suspended>
  );
};
