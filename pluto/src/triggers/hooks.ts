// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, compare, unique, type xy } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect, useState } from "react";

import { useStateRef } from "@/hooks/ref";
import { useMemoCompare } from "@/memo";
import { useContext } from "@/triggers/Provider";
import {
  diff,
  filter,
  type MatchOptions,
  purge,
  type Stage,
  type Trigger,
} from "@/triggers/triggers";

export interface UseEvent {
  target: HTMLElement;
  prevTriggers: Trigger[];
  triggers: Trigger[];
  stage: Stage;
  cursor: xy.XY;
}

export interface UseProps extends MatchOptions {
  triggers?: Trigger | Trigger[];
  region?: RefObject<HTMLElement | null>;
  callback?: (e: UseEvent) => void;
  regionMustBeElement?: boolean;
}

export const use = ({
  triggers,
  callback: f,
  region,
  loose,
  double,
  regionMustBeElement,
}: UseProps): void => {
  const { listen } = useContext();
  let baseTriggers: Trigger[];
  if (triggers != null && triggers?.length > 0 && typeof triggers[0] === "string")
    baseTriggers = [triggers as Trigger];
  else baseTriggers = triggers as Trigger[];
  const memoTriggers = useMemoCompare<Trigger[] | undefined, [Trigger[] | undefined]>(
    () => baseTriggers,
    ([a], [b]) => {
      if (a == null && b == null) return true;
      if (a == null || b == null) return false;
      return compare.primitiveArrays(a.flat(), b.flat()) === compare.EQUAL;
    },
    [baseTriggers],
  );

  useEffect(() => {
    if (memoTriggers == null || memoTriggers.length === 0) return;
    return listen((e) => {
      const prevMatches = filter(memoTriggers, e.prev, { loose, double });
      const nextMatches = filter(memoTriggers, e.next, { loose, double });
      const res = diff(nextMatches, prevMatches);
      let added = res[0];
      const removed = res[1];
      if (added.length === 0 && removed.length === 0) return;
      added = filterInRegion(e.target, e.cursor, added, region, regionMustBeElement);
      const base = { target: e.target, cursor: e.cursor };
      if (added.length > 0)
        f?.({ ...base, stage: "start", triggers: added, prevTriggers: e.prev });
      if (removed.length > 0)
        f?.({ ...base, stage: "end", triggers: removed, prevTriggers: e.prev });
    });
  }, [f, memoTriggers, listen, loose, region, double, regionMustBeElement]);
};

const filterInRegion = (
  target: HTMLElement,
  cursor: xy.XY,
  added: Trigger[],
  region?: RefObject<HTMLElement | null>,
  regionMustBeElement?: boolean,
): Trigger[] => {
  if (region == null) return added;
  if (region.current == null) return [];
  const b = box.construct(region.current);
  return added.filter((t) => {
    const rg = regionMustBeElement ?? t.some((v) => v.includes("Mouse"));
    if (rg) return box.contains(b, cursor) && target === region.current;
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
}: UseHeldProps): RefObject<UseHeldReturn> => {
  const [ref, setRef] = useStateRef<UseHeldReturn>({
    triggers: [],
    held: false,
  });
  use({
    triggers,
    callback: useCallback((e: UseEvent) => {
      setRef((prev) => {
        let next: Trigger[];
        if (e.stage === "start")
          next = unique.unique([...prev.triggers, ...e.triggers]);
        else next = purge(prev.triggers, e.triggers);
        return { triggers: next, held: next.length > 0 };
      });
    }, []),
    loose,
  });
  return ref;
};

export const useHeld = ({ triggers, loose }: UseHeldProps): UseHeldReturn => {
  const [held, setHeld] = useState<UseHeldReturn>({ triggers: [], held: false });
  use({
    triggers,
    callback: useCallback((e: UseEvent) => {
      setHeld((prev) => {
        let next: Trigger[];
        if (e.stage === "start")
          next = unique.unique([...prev.triggers, ...e.triggers]);
        else next = purge(prev.triggers, e.triggers);
        return { triggers: next, held: next.length > 0 };
      });
    }, []),
    loose,
  });
  return held;
};
