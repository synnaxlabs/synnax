// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log as Base } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";

import { ContextMenu } from "@/component/context-menu";
import { Empty } from "@/component/empty";
import { Session } from "@/session";

const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

export interface LogProps {
  visible: boolean;
}

export const Log = ({ visible }: LogProps) => {
  const key = Base.useKey();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const channelKeys = Base.useSelectChannelKeys();
  const hasChannels = channelKeys.some((k) => !primitive.isZero(k));

  const modals = Session.Modals.useStore("Log");
  const enableTriggers = useCallback(
    () =>
      Session.Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState(), modals) ===
      key,
    [store, key, modals],
  );

  const handleDoubleClick = useCallback(() => {
    dispatch(Session.Nav.showBottom({}));
  }, [dispatch]);

  const handleConfigureChannels = useCallback(() => {
    dispatch(Session.Log.setSelectedToolbarTab({ key, tab: "channels" }));
    handleDoubleClick();
  }, [dispatch, key, handleDoubleClick]);

  return (
    <Base.Log
      onDoubleClick={handleDoubleClick}
      enableTriggers={enableTriggers}
      extraContextMenuItems={EXTRA_CONTEXT_MENU_ITEMS}
      emptyContent={
        <Empty.Action
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
