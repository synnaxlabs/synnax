// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Log as PLog } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { createEnsureState } from "@/hooks/useEnsureState";
import { Layout } from "@/layout";
import { useSelectExists, useSelectPendingUpload } from "@/log/selectors";
import { clearPendingUpload, internalCreate, setActiveToolbarTab } from "@/log/slice";
import { createUseAutoUpload } from "@/vis/useAutoUpload";

export { create, LAYOUT_TYPE, type LayoutType } from "@/log/layout";

const useAutoUpload = createUseAutoUpload({
  useSelectPendingUpload,
  useCreate: PLog.useCreate,
  clearPendingUpload,
});

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

const Loaded: Layout.Renderer = ({ layoutKey, visible, active }) => {
  PLog.useEnsureRetrieved({ key: layoutKey });
  const dispatch = useDispatch();
  const channelKeys = PLog.useSelectChannelKeys({ key: layoutKey });
  const hasChannels = channelKeys.some((k) => !primitive.isZero(k));

  const enableTriggers = useCallback(() => active, [active]);

  const handleDoubleClick = useCallback(() => {
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
  }, [dispatch]);

  const handleConfigureChannels = useCallback(() => {
    dispatch(setActiveToolbarTab({ key: layoutKey, tab: "channels" }));
    handleDoubleClick();
  }, [dispatch, layoutKey, handleDoubleClick]);

  return (
    <PLog.Log
      resourceKey={layoutKey}
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

const useEnsureState = createEnsureState({
  useExists: useSelectExists,
  create: (key) => internalCreate({ key }),
});

export const Log: Layout.Renderer = (props) => {
  const exists = useEnsureState(props.layoutKey);
  const uploaded = useAutoUpload(props.layoutKey);
  if (!exists || !uploaded) return null;
  return <Loaded {...props} />;
};

Log.useName = Layout.createUseFluxName(PLog.useRename, PLog.useRetrieveObservableName);
Log.icon = <Icon.Log />;
