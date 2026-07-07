// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Panel, Text, TimeSpan, Triggers } from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";

import { Selector } from "@/app/selector";
import { useSelectorVisible } from "@/app/vis/Selector";
import { Session } from "@/session";
import { Modals } from "@/session/modals";

const CLOSE_WINDOW_TIMEOUT = TimeSpan.milliseconds(350);

// TODO(SY-4370): open-in-new-window gesture (formerly Control+O) needs a panel
// equivalent: create a Drift window and select the panel in it.

export const useTriggers = (): void => {
  const sessionDispatch = Session.useDispatch();
  const modals = Modals.useStore("useTriggers");
  const getSelectedPanel = Session.Panel.useGetSelected();
  const getIsOverlaid = Session.Panel.useGetIsOverlaid();
  const getFocusedTab = Session.Panel.useGetFocusedTab();
  const { dispatch } = Panel.useDispatch();
  const closeWindowTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const createComponentEnabled = useSelectorVisible();
  const openSelector = Selector.useOpenTab();
  Triggers.use({
    triggers: [["Control", "L"]],
    loose: true,
    callback: useCallback(({ stage }: Triggers.UseEvent) => {
      if (stage !== "start") return;
      const overlaid = getIsOverlaid();
      if (overlaid) {
        sessionDispatch(Session.Panel.stopOverlaying({}));
        return;
      }
      const focused = getFocusedTab();
      if (focused != null) sessionDispatch(Session.Panel.startOverlaying({}));
    }, []),
  });
  Triggers.use({
    triggers: [["Control", "W"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") {
          if (stage === "end" && closeWindowTimeout.current != null) {
            clearTimeout(closeWindowTimeout.current);
            closeWindowTimeout.current = null;
          }
          return;
        }
        if (modals.isAnyOpen()) return modals.closeTop();
        const panelKey = getSelectedPanel();
        const focused = getFocusedTab();
        if (panelKey != null && focused != null) {
          if (getIsOverlaid()) sessionDispatch(Session.Panel.stopOverlaying({}));
          dispatch({
            key: panelKey,
            actions: [panel.removeTab({ key: focused })],
          });
          return;
        }
        closeWindowTimeout.current = setTimeout(
          () => sessionDispatch(Drift.closeWindow({})),
          CLOSE_WINDOW_TIMEOUT.milliseconds,
        );
      },
      [
        dispatch,
        sessionDispatch,
        getSelectedPanel,
        getFocusedTab,
        getIsOverlaid,
        modals,
      ],
    ),
  });
  Triggers.use({
    triggers: [["Control", "E"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const focused = getFocusedTab();
        if (focused == null) return;
        Text.edit(`pluto-tab-${focused}`);
      },
      [getFocusedTab],
    ),
  });
  Triggers.use({
    triggers: [["Control", "T"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !createComponentEnabled) return;
        openSelector("component");
      },
      [createComponentEnabled, openSelector],
    ),
  });
};
