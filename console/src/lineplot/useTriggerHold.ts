// Copyright 2024 Synnax Labs, Inc.
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

import { useSelectControlState } from "@/lineplot/selectors";
import { setControlState } from "@/lineplot/slice";

export type Config = Triggers.ModeConfig<"toggle">;

export const useTriggerHold = (triggers: Config): void => {
  const { hold } = useSelectControlState();
  const ref = useSyncedRef(hold);
  const dispatch = useDispatch();
  const flat = Triggers.useFlattenedMemoConfig(triggers);
  Triggers.use({
    triggers: flat,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (e.stage !== "start") return;
        dispatch(setControlState({ state: { hold: !ref.current } }));
      },
      [dispatch],
    ),
  });
};
