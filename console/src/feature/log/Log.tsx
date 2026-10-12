// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type LinePlot, Log as Base, Panel as PPanel } from "@synnaxlabs/pluto";
import { primitive, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Controls } from "@/feature/log/Controls";
import { ContextMenu } from "@/platform/context-menu";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Session } from "@/session";

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

const useResolvedRange = (): LinePlot.ResolvedRange | undefined => {
  const key = Base.useRange();
  const custom = Base.useCustomRange();
  const resolved = Range.useResolve(key);
  return useMemo(() => {
    if (key === Range.CUSTOM_KEY) {
      if (custom == null) return undefined;
      return custom.variant === "dynamic"
        ? { variant: "dynamic", span: new TimeSpan(custom.span) }
        : {
            variant: "static",
            timeRange: new TimeRange(BigInt(custom.start), BigInt(custom.end)),
          };
    }
    if (resolved == null) return undefined;
    return resolved.variant === "dynamic"
      ? { variant: "dynamic", span: new TimeSpan(resolved.span) }
      : { variant: "static", timeRange: new TimeRange(resolved.timeRange) };
  }, [key, custom, resolved]);
};

const Internal: Panel.Content = () => {
  const key = Base.useKey();
  const dispatch = Session.useDispatch();
  const visible = Session.Panel.useSelectIsTabVisible();
  const channelKeys = Base.useChannelKeys();
  const hasChannels = channelKeys.some((k) => !primitive.isZero(k));
  const hold = Session.Log.useSelectHold();
  const resolvedRange = useResolvedRange();

  const handleDoubleClick = useCallback(() => {
    dispatch(Session.Nav.showBottom({}));
  }, [dispatch]);

  const handleConfigureChannels = useCallback(() => {
    dispatch(Session.Log.setSelectedToolbarTab({ key, tab: "channels" }));
    handleDoubleClick();
  }, [dispatch, key, handleDoubleClick]);

  const handleHold = useCallback(
    (hold: boolean) => dispatch(Session.Log.setHold({ key, hold })),
    [dispatch, key],
  );

  return (
    <Base.Log
      onDoubleClick={handleDoubleClick}
      hold={hold}
      onHold={handleHold}
      extraContextMenuItems={EXTRA_CONTEXT_MENU_ITEMS}
      emptyContent={
        <Empty.Action
          message={hasChannels ? "No data received yet" : "No channels configured"}
          action={hasChannels ? undefined : "Configure channels"}
          onClick={hasChannels ? handleDoubleClick : handleConfigureChannels}
        />
      }
      visible={visible}
      resolvedRange={resolvedRange}
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
