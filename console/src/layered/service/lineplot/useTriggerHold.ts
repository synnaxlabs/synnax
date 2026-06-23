// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/layered/session";
import { Layout } from "@/layout";

export type Config = Triggers.ModeConfig<"toggle">;

export const useTriggerHold = (triggers: Config): void => {
  const { layoutKey: activeTab } = Layout.useSelectActiveMosaicTabState();
  const controlState = Session.LinePlot.useSelectControlStateOptional({
    key: activeTab ?? "",
  });
  const ref = useSyncedRef(controlState?.hold);
  const dispatch = useDispatch();
  const flat = Triggers.useFlattenedMemoConfig(triggers);
  Triggers.use({
    triggers: flat,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (e.stage !== "start" || activeTab == null || ref.current == null) return;
        dispatch(
          Session.LinePlot.setControlState({
            key: activeTab,
            state: { hold: !ref.current },
          }),
        );
      },
      [dispatch, activeTab, flat, ref],
    ),
  });
};
