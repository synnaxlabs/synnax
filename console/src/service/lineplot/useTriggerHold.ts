// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";
import { Layout } from "@/layout";

export type Config = Triggers.ModeConfig<"toggle">;

const CONFIG: Triggers.ModeConfig<"toggle"> = {
  defaultMode: "toggle",
  toggle: [["H"]],
};

export const HOLD_TRIGGER: Triggers.Trigger = ["H"];

export const useTriggerHold = (): void => {
  const { layoutKey: activeTab } = Layout.useSelectActiveMosaicTabState();
  const dispatch = Session.useDispatch();
  const flat = Triggers.useFlattenedMemoConfig(CONFIG);
  Triggers.use({
    triggers: flat,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (e.stage === "start" && activeTab != null)
          dispatch(Session.LinePlot.setControlHold({ key: activeTab }));
      },
      [dispatch, activeTab, flat],
    ),
  });
};
