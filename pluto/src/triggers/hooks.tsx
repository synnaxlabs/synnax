// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, RefObject, useEffect, useRef, useState } from "react";

import { Box, Compare } from "@synnaxlabs/x";

import { useMemoCompare } from "..";

import { Trigger, TriggerCallback } from "./triggers";
import { useTriggerContext } from "./TriggersContext";

export const useTrigger = (
  triggers: Trigger[],
  callback?: TriggerCallback,
  region?: RefObject<HTMLElement>
): MutableRefObject<UseTriggerHeldReturn> => {
  const { listen } = useTriggerContext();
  const ref = useRef<UseTriggerHeldReturn>({ triggers: [], held: false });
  const memoTriggers = useMemoCompare(
    () => triggers,
    ([a], [b]) => Compare.primitiveArrays(a.flat(), b.flat()) === 0,
    [triggers]
  );
  useEffect(() => {
    return listen((e) => {
      if (region != null) {
        if (region.current == null) return;
        const box = new Box(region.current);
        if (
          (e.stage === "start" || !ref.current.held) &&
          !box.contains(e.cursor) &&
          e.target !== region.current
        )
          return;
      }
      ref.current = {
        triggers: e.stage === "end" ? [] : triggers,
        held: e.stage === "start",
      };
      callback?.(e);
    }, triggers);
  }, [callback, memoTriggers, listen]);
  return ref;
};

export interface UseTriggerHeldReturn {
  triggers: Trigger[];
  held: boolean;
}

export const useTriggerHeld = (triggers: Trigger[]): UseTriggerHeldReturn => {
  const [held, setHeld] = useState<UseTriggerHeldReturn>({
    triggers: [],
    held: false,
  });
  useTrigger(triggers, ({ triggers, stage }) =>
    setHeld({ triggers: stage === "end" ? [] : triggers, held: stage === "start" })
  );
  return held;
};
