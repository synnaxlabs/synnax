// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { selectWindowKey } from "@synnaxlabs/drift";
import { Text, Triggers } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

import { useRemover } from "@/layout/hooks";
import { selectActiveMosaicTabKey, selectFocused } from "@/layout/selectors";
import { setFocus } from "@/layout/slice";
import { useOpenInNewWindow } from "@/layout/useOpenInNewWindow";
import { type RootState } from "@/store";

export const useTriggers = () => {
  const store = useStore<RootState>();
  const remove = useRemover();
  const openInNewWindow = useOpenInNewWindow();
  Triggers.use({
    triggers: [["Control", "L"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      const windowKey = selectWindowKey(state);
      const { focused } = selectFocused(state);
      if (active == null || windowKey == null) return;
      if (focused != null) store.dispatch(setFocus({ key: null, windowKey }));
      else store.dispatch(setFocus({ key: active, windowKey }));
    },
  });
  Triggers.use({
    triggers: [["Control", "W"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      remove(active);
    },
  });
  Triggers.use({
    triggers: [["Control", "O"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      openInNewWindow(active);
    },
  });
  Triggers.use({
    triggers: [["Control", "E"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      Text.edit(`pluto-tab-${active}`);
    },
  });
};
