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
import { useStore } from "react-redux";

import {
  selectActiveMosaicTabState,
  selectFocused,
  selectModals,
} from "@/layout/selectors";
import { setFocus } from "@/layout/slice";
import { useOpenInNewWindow } from "@/layout/useOpenInNewWindow";
import { usePlacer } from "@/layout/usePlacer";
import { useRemover } from "@/layout/useRemover";
import { createSelectorLayout, useSelectorVisible } from "@/layouts/Selector";
import { Runtime } from "@/runtime";
import { type RootState } from "@/store";

const CLOSE_WINDOW_TIMEOUT = TimeSpan.milliseconds(350);

export const useTriggers = (): void => {
  const store = useStore<RootState>();
  const remove = useRemover();
  const openInNewWindow = useOpenInNewWindow();
  const placeLayout = usePlacer();
  const closeWindowTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const createComponentEnabled = useSelectorVisible();
  Triggers.use({
    triggers: [["Control", "L"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const state = store.getState();
        const { layoutKey: active } = selectActiveMosaicTabState(state);
        const windowKey = selectWindowKey(state);
        const { focused } = selectFocused(state);
        if (active == null || windowKey == null) return;
        if (focused != null) store.dispatch(setFocus({ key: null, windowKey }));
        else store.dispatch(setFocus({ key: active, windowKey }));
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
        const state = store.getState();
        const modals = selectModals(state);
        if (modals.length !== 0) return remove(modals[0].key);
        const { layoutKey: active } = selectActiveMosaicTabState(state);
        if (active != null) return remove(active);
        closeWindowTimeout.current = setTimeout(
          () => store.dispatch(Drift.closeWindow({})),
          CLOSE_WINDOW_TIMEOUT.milliseconds,
        );
      },
      [store, remove, openInNewWindow],
    ),
  });
  Triggers.use({
    triggers: [["Control", "O"]],
    loose: true,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (Runtime.ENGINE !== "tauri") return;
        const state = store.getState();
        const { layoutKey: active } = selectActiveMosaicTabState(state);
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
        const { layoutKey: active } = selectActiveMosaicTabState(state);
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
        if (stage !== "start" || !createComponentEnabled) return;
        placeLayout(createSelectorLayout({ tab: { location: "center" } }));
      },
      [createComponentEnabled, placeLayout],
    ),
  });
};
