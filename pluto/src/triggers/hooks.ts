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

import { useStateRef, useSyncedRef } from "@/hooks/ref";
import { useMemoCompare } from "@/memo";
import { useContext } from "@/triggers/Provider";
import { type Condition, resolveCondition, useScope } from "@/triggers/Scope";
import {
  determineMode,
  diff,
  filter,
  flattenConfig,
  type MatchOptions,
  type ModeConfig,
  purge,
  REDO,
  type Stage,
  type Trigger,
  UNDO,
} from "@/triggers/triggers";

export interface UseEvent {
  target: HTMLElement;
  prevTriggers: Trigger[];
  triggers: Trigger[];
  stage: Stage;
  cursor: xy.XY;
  /**
   * Prevents the event from being dispatched to any remaining Triggers.use
   * subscribers with a lower priority than the current one. Subscribers at the
   * same priority that have not yet been notified still receive the event.
   */
  stopPropagation: () => void;
}

export interface UseProps extends MatchOptions {
  triggers?: Trigger | Trigger[];
  region?: RefObject<HTMLElement | null>;
  callback?: (e: UseEvent) => void;
  regionMustBeElement?: boolean;
  /**
   * Withholds events from this subscriber while it resolves false. Use it for conditions
   * the subscriber itself owns, such as whether its content is editable. Whether the
   * surrounding view is the one the user is working in belongs in a {@link Scope}.
   */
  enabled?: Condition;
  /**
   * Priority of this subscriber. Higher-priority subscribers receive events
   * before lower-priority ones and may call stopPropagation on the event to
   * prevent lower-priority subscribers from receiving it. Defaults to 0.
   */
  priority?: number;
}

export const use = ({
  triggers,
  callback: f,
  region,
  loose,
  double,
  regionMustBeElement,
  priority,
  enabled = true,
}: UseProps): void => {
  const { listen } = useContext();
  const scope = useScope();
  const activeRef = useSyncedRef(() => scope() && resolveCondition(enabled));
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
      const base = {
        target: e.target,
        cursor: e.cursor,
        stopPropagation: e.stopPropagation,
      };
      if (added.length > 0 && activeRef.current())
        f?.({ ...base, stage: "start", triggers: added, prevTriggers: e.prev });
      // A release lands even while inactive, so a key held when the subscriber went
      // inactive can never stick.
      if (removed.length > 0)
        f?.({ ...base, stage: "end", triggers: removed, prevTriggers: e.prev });
    }, priority);
  }, [f, memoTriggers, listen, loose, region, double, regionMustBeElement, priority]);
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

const UNDO_REDO_CONFIG: ModeConfig<"undo" | "redo" | "default"> = {
  undo: [UNDO],
  redo: [REDO],
  default: [],
  defaultMode: "default",
};
const UNDO_REDO_TRIGGERS = flattenConfig(UNDO_REDO_CONFIG);

export interface UseUndoRedoProps {
  undo: () => void;
  redo: () => void;
  enabled?: Condition;
}

/** useUndoRedo binds the standard undo and redo shortcuts to the given handlers. */
export const useUndoRedo = ({ undo, redo, enabled }: UseUndoRedoProps): void => {
  use({
    triggers: UNDO_REDO_TRIGGERS,
    loose: true,
    enabled,
    callback: useCallback(
      ({ triggers, stage }: UseEvent) => {
        if (stage !== "start") return;
        const mode = determineMode(UNDO_REDO_CONFIG, triggers);
        if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [undo, redo],
    ),
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
