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
import { useDispatch, useStore } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

const Internal: Layout.Renderer = ({ visible }) => {
  const key = PLog.useKey();
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const channelKeys = PLog.useSelectChannelKeys();
  const hasChannels = channelKeys.some((k) => !primitive.isZero(k));

  const enableTriggers = useCallback(
    () => Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState()) === key,
    [store, key],
  );

  const handleDoubleClick = useCallback(() => {
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
  }, [dispatch]);

  const handleConfigureChannels = useCallback(() => {
    dispatch(Session.Log.setActiveToolbarTab({ key, tab: "channels" }));
    handleDoubleClick();
  }, [dispatch, key, handleDoubleClick]);

  return (
    <PLog.Log
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
};

export const Log: Layout.Renderer = (props) => (
  <PLog.Suspended logKey={props.layoutKey}>
    <Internal {...props} />
  </PLog.Suspended>
);
Log.useName = Layout.createUseFluxName(PLog.useRename, PLog.useRetrieveObservableName);
