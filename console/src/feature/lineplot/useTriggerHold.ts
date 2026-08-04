// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { Triggers } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { HOLD_TRIGGER } from "@/feature/lineplot/Controls";
import { Session } from "@/session";

export interface UseTriggerHoldProps {
  key: lineplot.Key;
  enabled: Triggers.Condition;
}

export const useTriggerHold = ({ key, enabled }: UseTriggerHoldProps): void => {
  const dispatch = Session.useDispatch();
  Triggers.use({
    triggers: HOLD_TRIGGER,
    loose: true,
    enabled,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        dispatch(Session.LinePlot.setControlHold({ key }));
      },
      [dispatch, key],
    ),
  });
};
