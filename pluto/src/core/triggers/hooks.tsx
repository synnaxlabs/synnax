// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, RefObject, useCallback, useEffect, useState } from "react";

import { Box, Compare, XY, unique, OS } from "@synnaxlabs/x";

import { useOS } from "../hooks";

import { useStateRef } from "@/core/hooks/useStateRef";
import { useMemoCompare } from "@/core/memo";
import { diff, filter, purge, Stage, Trigger } from "@/core/triggers/triggers";
import { useTriggerContext } from "@/core/triggers/TriggersContext";

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
  os?: OS;
}

export const useTrigger = ({
  triggers,
  callback: f,
  region,
  loose,
  os: propsOS,
}: UseTriggerProps): void => {
  const { listen } = useTriggerContext();
  const memoTriggers = useMemoCompare(
    () => triggers,
    ([a], [b]) => Compare.primitiveArrays(a.flat(), b.flat()) === 0,
    [triggers]
  );

  const os = useOS({ default: propsOS });

  useEffect(() => {
    return listen((e) => {
      const prevMatches = filter(memoTriggers, e.prev, /* loose */ loose);
      const nextMatches = filter(memoTriggers, e.next, /* loose */ loose);
      let [added, removed] = diff(nextMatches, prevMatches);
      added = filterInRegion(e.target, e.cursor, added, region);
      if (added.length > 0) f?.({ stage: "start", triggers: added, cursor: e.cursor });
      if (removed.length > 0)
        f?.({ stage: "end", triggers: removed, cursor: e.cursor });
    });
  }, [f, memoTriggers, listen, loose]);
};

const filterInRegion = (
  target: HTMLElement,
  cursor: XY,
  added: Trigger[],
  region?: RefObject<HTMLElement>
): Trigger[] => {
  if (region == null) return added;
  if (region.current == null) return [];
  const b = new Box(region.current);
  return added.filter((t) => {
    if (t.some((v) => v.includes("Mouse")))
      return b.contains(cursor) && target === region.current;
    return b.contains(cursor);
  });
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
