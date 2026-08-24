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
import { Panel as PPanel, TimeSpan, Triggers } from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";

import { Palette } from "@/app/palette";
import { useSelectorVisible } from "@/app/vis/Selector";
import { Panel } from "@/feature/panel";
import { Panel as PlatformPanel } from "@/platform/panel";
import { Selector } from "@/platform/selector";
import { Session } from "@/session";
import { Modals } from "@/session/modals";

const PREVENT_DEFAULT_ON: Triggers.Trigger[] = [
  Palette.SEARCH_TRIGGER,
  Palette.COMMAND_TRIGGER,
  ["Control", "MouseLeft"],
  PPanel.CLOSE_TRIGGER,
];

export const PROVIDER_PROPS: Triggers.ProviderProps = {
  preventDefaultOn: PREVENT_DEFAULT_ON,
  preventDefaultOptions: { double: true },
};

const OVERLAY_TRIGGERS: Triggers.Trigger[] = [PPanel.OVERLAY_TRIGGER];
const ESCAPE_TRIGGERS: Triggers.Trigger[] = [Triggers.ESCAPE];
const CLOSE_TRIGGERS: Triggers.Trigger[] = [PPanel.CLOSE_TRIGGER];
const RENAME_TRIGGERS: Triggers.Trigger[] = [PlatformPanel.RENAME_TRIGGER];
const CREATE_TAB_TRIGGERS: Triggers.Trigger[] = [PPanel.CREATE_TAB_TRIGGER];
const OPEN_WINDOW_TRIGGERS: Triggers.Trigger[] = [Panel.OPEN_WINDOW_TRIGGER];

const CLOSE_WINDOW_TIMEOUT = TimeSpan.milliseconds(350);

export const use = (): void => {
  const sessionDispatch = Session.useDispatch();
  const modals = Modals.useStore("useTriggers");
  const getSelectedPanel = Session.Panel.useGetSelected();
  const getIsOverlaid = Session.Panel.useGetIsOverlaid();
  const getFocusedTab = Session.Panel.useGetFocusedTab();
  const { dispatch } = PPanel.useDispatch();
  const openWindow = Panel.useOpenWindow();
  const closeWindowTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const createTabEnabled = useSelectorVisible();
  const canEditPanel = PlatformPanel.useCanEditActive();
  const openSelector = Selector.useOpenTab();
  Triggers.use({
    triggers: OVERLAY_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (getIsOverlaid()) {
          sessionDispatch(Session.Panel.stopOverlaying({}));
          return;
        }
        const focused = getFocusedTab();
        if (focused != null) sessionDispatch(Session.Panel.startOverlaying({}));
      },
      [getIsOverlaid, getFocusedTab, sessionDispatch],
    ),
  });
  Triggers.use({
    triggers: ESCAPE_TRIGGERS,
    double: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !getIsOverlaid()) return;
        sessionDispatch(Session.Panel.stopOverlaying({}));
      },
      [getIsOverlaid, sessionDispatch],
    ),
  });
  Triggers.use({
    triggers: CLOSE_TRIGGERS,
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
        if (canEditPanel && panelKey != null && focused != null) {
          if (getIsOverlaid()) sessionDispatch(Session.Panel.stopOverlaying({}));
          dispatch({
            key: panelKey,
            actions: [panel.removeTab({ key: focused })],
          });
          return;
        }
        // A page cannot close the tab that holds it.
        if (Session.Runtime.ENGINE !== "tauri") return;
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
        canEditPanel,
      ],
    ),
  });
  Triggers.use({
    triggers: RENAME_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const focused = getFocusedTab();
        if (focused == null) return;
        PlatformPanel.editTabName(focused);
      },
      [getFocusedTab],
    ),
  });
  Triggers.use({
    triggers: OPEN_WINDOW_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || Session.Runtime.ENGINE !== "tauri") return;
        openWindow(getSelectedPanel());
      },
      [openWindow, getSelectedPanel],
    ),
  });
  Triggers.use({
    triggers: CREATE_TAB_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !createTabEnabled) return;
        openSelector();
      },
      [createTabEnabled, openSelector],
    ),
  });
};
