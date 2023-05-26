// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, RefObject, useCallback, useEffect, useState } from "react";

import { Box, Compare, XY, unique } from "@synnaxlabs/x";

import { useStateRef } from "@/hooks/useStateRef";
import { useMemoCompare } from "@/memo";
import { diff, filter, purge, Stage, Trigger } from "@/triggers/triggers";
import { useTriggerContext } from "@/triggers/TriggersContext";

export interface UseTriggerEvent {
  triggers: Trigger[];
  stage: Stage;
  cursor: XY;
}

export interface UseTriggerProps {
  triggers: Trigger[];
  callback?: (e: UseTriggerEvent) => void;
  region?: RefObject<HTMLElement>;
  loose?: boolean;
}

export const useTrigger = ({
  triggers,
  callback: f,
  region,
  loose,
}: UseTriggerProps): void => {
  const { listen } = useTriggerContext();
  const memoTriggers = useMemoCompare(
    () => triggers,
    ([a], [b]) => Compare.primitiveArrays(a.flat(), b.flat()) === 0,
    [triggers]
  );
  useEffect(() => {
    return listen((e) => {
      const prevMatches = filter(memoTriggers, e.prev, /* loose */ loose);
      const nextMatches = filter(memoTriggers, e.next, /* loose */ loose);
      const [added, removed] = diff(nextMatches, prevMatches);
      if (
        added.length > 0 &&
        (region == null ||
          (region.current != null && new Box(region.current).contains(e.cursor)) ||
          e.target === region.current)
      )
        f?.({ stage: "start", triggers: added, cursor: e.cursor });
      if (removed.length > 0)
        f?.({ stage: "end", triggers: removed, cursor: e.cursor });
    });
  }, [f, memoTriggers, listen, loose]);
};

export interface UseTriggerHeldReturn {
  triggers: Trigger[];
  held: boolean;
}

export interface UseTriggerHeldProps {
  triggers: Trigger[];
  loose?: boolean;
}

export const useTriggerHeldRef = ({
  triggers,
  loose,
}: UseTriggerHeldProps): MutableRefObject<UseTriggerHeldReturn> => {
  const [ref, setRef] = useStateRef<UseTriggerHeldReturn>({
    triggers: [],
    held: false,
  });
  useTrigger({
    triggers,
    callback: (e) => {
      setRef((prev) => {
        let next: Trigger[] = [];
        if (e.stage === "start") {
          next = unique([...prev.triggers, ...e.triggers]);
        } else {
          next = purge(prev.triggers, e.triggers);
        }
        return { triggers: next, held: next.length > 0 };
      });
    },
    loose,
  });
  return ref;
};

export const useTriggerHeld = ({
  triggers,
  loose,
}: UseTriggerHeldProps): UseTriggerHeldReturn => {
  const [held, setHeld] = useState<UseTriggerHeldReturn>({
    triggers: [],
    held: false,
  });
  useTrigger({
    triggers,
    callback: useCallback((e: UseTriggerEvent) => {
      setHeld((prev) => {
        let next: Trigger[] = [];
        if (e.stage === "start") {
          next = unique([...prev.triggers, ...e.triggers]);
        } else {
          next = purge(prev.triggers, e.triggers);
        }
        return { triggers: next, held: next.length > 0 };
      });
    }, []),
    loose,
  });
  return held;
};
