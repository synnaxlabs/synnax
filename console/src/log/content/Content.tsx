// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log as PLog } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { Session } from "@/log/session";
import { Tab } from "@/log/tab";
import { Nav } from "@/nav";
import { Panel } from "@/panel";

export interface ContentProps {
  visible?: boolean;
}

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

export const Content = Tab.createSuspended<ContentProps>(({ visible = true }) => {
  const key = PLog.useKey();
  const dispatch = useDispatch();
  const channelKeys = PLog.useSelectChannelKeys({});
  const hasChannels = channelKeys.some((k) => !primitive.isZero(k));
  const focused = Panel.useSelectIsFocused();
  const setActiveToolbarTab = Session.useSetActiveToolbarTab();

  const enableTriggers = useCallback(() => focused, [focused]);

  const handleDoubleClick = useCallback(
    () => dispatch(Nav.setBottomVisible(true)),
    [dispatch],
  );

  const handleConfigureChannels = useCallback(() => {
    setActiveToolbarTab("channels");
    dispatch(Nav.setBottomVisible(true));
  }, [setActiveToolbarTab, dispatch]);

  return (
    <PLog.Log
      resourceKey={key}
      onDoubleClick={handleDoubleClick}
      enableTriggers={enableTriggers}
      extraContextMenuItems={EXTRA_CONTEXT_MENU_ITEMS}
      emptyContent={
        <EmptyAction
          message={
            hasChannels
              ? "No data received yet."
              : "No channels configured for this log."
          }
          action={hasChannels ? "" : "Configure channels"}
          onClick={hasChannels ? handleDoubleClick : handleConfigureChannels}
        />
      }
      visible={visible}
    />
  );
});
