// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type MutableRefObject,
  type RefObject,
  useCallback,
  useEffect,
  useState,
} from "react";

import { box, Compare, runtime, unique, xy } from "@synnaxlabs/x";

import { useStateRef } from "@/hooks/useStateRef";
import { useMemoCompare } from "@/memo";
import { useContext } from "@/triggers/Context";
import { diff, filter, purge, type Stage, type Trigger } from "@/triggers/triggers";

export interface UseEvent {
  triggers: Trigger[];
  stage: Stage;
  cursor: xy.XY;
}

export interface UseProps {
  triggers: Trigger[];
  callback?: (e: UseEvent) => void;
  region?: RefObject<HTMLElement>;
  loose?: boolean;
  os?: runtime.OS;
}

export const use = ({
  triggers,
  callback: f,
  region,
  loose,
  os: propsOS,
}: UseProps): void => {
  const { listen } = useContext();
  const memoTriggers = useMemoCompare(
    () => triggers,
    ([a], [b]) => Compare.primitiveArrays(a.flat(), b.flat()) === Compare.EQUAL,
    [triggers],
  );

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
  cursor: xy.XY,
  added: Trigger[],
  region?: RefObject<HTMLElement>,
): Trigger[] => {
  if (region == null) return added;
  if (region.current == null) return [];
  const b = box.construct(region.current);
  return added.filter((t) => {
    if (t.some((v) => v.includes("Mouse")))
      return box.contains(b, cursor) && target === region.current;
    return box.contains(b, cursor);
  });
};

export interface UseHeldReturn {
  triggers: Trigger[];
  held: boolean;
}

export interface UseHeldProps {
  triggers: Trigger[];
  loose?: boolean;
}

export const useHeldRef = ({
  triggers,
  loose,
}: UseHeldProps): MutableRefObject<UseHeldReturn> => {
  const [ref, setRef] = useStateRef<UseHeldReturn>({
    triggers: [],
    held: false,
  });
  use({
    triggers,
    callback: useCallback((e: UseEvent) => {
      setRef((prev) => {
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
  return ref;
};

export const useHeld = ({ triggers, loose }: UseHeldProps): UseHeldReturn => {
  const [held, setHeld] = useState<UseHeldReturn>({
    triggers: [],
    held: false,
  });
  use({
    triggers,
    callback: useCallback((e: UseEvent) => {
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
