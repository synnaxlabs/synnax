// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift, selectWindowKey } from "@synnaxlabs/drift";
import { Text, TimeSpan, Triggers } from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";

import { Selector } from "@/app/selector";
import { Layout } from "@/platform/layout";
import { Session } from "@/session";
import { Modals } from "@/session/modals";

const PREVENT_DEFAULT_ON: Triggers.Trigger[] = [
  ["Control", "P"],
  ["Control", "Shift", "P"],
  ["Control", "MouseLeft"],
  ["Control", "W"],
];

export const PROVIDER_PROPS: Triggers.ProviderProps = {
  preventDefaultOn: PREVENT_DEFAULT_ON,
  preventDefaultOptions: { double: true },
};

const CLOSE_WINDOW_TIMEOUT = TimeSpan.milliseconds(350);

export const use = (): void => {
  const store = Session.useStore();
  const modals = Modals.useStore("Layout.useTriggers");
  const remove = Layout.useRemover();
  const openInNewWindow = Layout.useOpenInNewWindow();
  const placeLayout = Layout.usePlacer();
  const closeWindowTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const createTabEnabled = Selector.useVisible();
  Triggers.use({
    triggers: [["Control", "L"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const state = store.getState();
        const { layoutKey: active } = Session.Layout.selectActiveMosaicTabState(state);
        const windowKey = selectWindowKey(state);
        const { focused } = Session.Layout.selectFocused(state);
        if (active == null || windowKey == null) return;
        if (focused != null)
          store.dispatch(Session.Layout.setFocus({ key: null, windowKey }));
        else store.dispatch(Session.Layout.setFocus({ key: active, windowKey }));
      },
      [store],
    ),
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
        const state = store.getState();
        const { layoutKey: active } = Session.Layout.selectActiveMosaicTabState(state);
        if (active != null) return remove(active);
        closeWindowTimeout.current = setTimeout(
          () => store.dispatch(Drift.closeWindow({})),
          CLOSE_WINDOW_TIMEOUT.milliseconds,
        );
      },
      [store, remove, openInNewWindow, modals],
    ),
  });
  Triggers.use({
    triggers: [["Control", "O"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (Session.Runtime.ENGINE !== "tauri") return;
        const state = store.getState();
        const { layoutKey: active } = Session.Layout.selectActiveMosaicTabState(state);
        if (active == null) return;
        openInNewWindow(active);
      },
      [store, openInNewWindow],
    ),
  });
  Triggers.use({
    triggers: [["Control", "E"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const state = store.getState();
        const { layoutKey: active } = Session.Layout.selectActiveMosaicTabState(state);
        if (active == null) return;
        Text.edit(`pluto-tab-${active}`);
      },
      [store],
    ),
  });
  Triggers.use({
    triggers: [["Control", "T"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !createTabEnabled) return;
        placeLayout(Selector.create({ tab: { location: "center" } }));
      },
      [createTabEnabled, placeLayout],
    ),
  });
};
