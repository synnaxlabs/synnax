// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, compare, unique, type xy } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect, useRef, useState } from "react";

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

/** The event {@link use} hands its callback. */
export interface UseEvent {
  target: HTMLElement;
  prevTriggers: Trigger[];
  /** The matched triggers this stage applies to. */
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

/** Props for {@link use}. */
export interface UseProps extends MatchOptions {
  triggers?: Trigger | Trigger[];
  /** Fires only while the cursor sits inside this element. */
  region?: RefObject<HTMLElement | null>;
  callback?: (e: UseEvent) => void;
  /** Whether the event target must be the region itself, not a descendant. */
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

/**
 * Binds a keyboard or mouse shortcut for as long as the caller is mounted. The callback
 * fires on press and again on release, and only while the enclosing {@link Scope} is
 * the active one.
 *
 * @example
 * Triggers.use({ triggers: [["Control", "S"]], callback: ({ stage }) => save(stage) });
 */
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
  const startedRef = useRef<Set<string>>(new Set());
  let baseTriggers: Trigger[];
  if (triggers != null && triggers?.length > 0 && typeof triggers[0] === "string")
    baseTriggers = [triggers as Trigger];
  else baseTriggers = triggers as Trigger[];
  const memoTriggers = useMemoCompare<Trigger[] | undefined, [Trigger[] | undefined]>(
    () => baseTriggers,
    ([a], [b]) => {
      // Load-bearing: lets call sites that hoist their triggers skip the compare.
      if (a === b) return true;
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
      const [added, removed] = res;
      if (added.length === 0 && removed.length === 0) return;
      const inRegion = filterInRegion(
        e.target,
        e.cursor,
        added,
        region,
        regionMustBeElement,
      );
      const base = {
        target: e.target,
        cursor: e.cursor,
        stopPropagation: e.stopPropagation,
      };
      if (activeRef.current()) {
        added.forEach((t) => startedRef.current.add(t.join("+")));
        if (inRegion.length > 0)
          f?.({ ...base, stage: "start", triggers: inRegion, prevTriggers: e.prev });
      }
      // A release lands only for a press seen while active: a key held when the
      // subscriber went inactive cannot stick, and a press a scope withheld stays
      // withheld through its release.
      const ended = removed.filter((t) => startedRef.current.delete(t.join("+")));
      if (ended.length > 0)
        f?.({ ...base, stage: "end", triggers: ended, prevTriggers: e.prev });
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

/** Props for {@link useUndoRedo}. */
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

/** Which of the watched triggers are down right now. */
export interface UseHeldReturn {
  triggers: Trigger[];
  held: boolean;
}

/** Props for {@link useHeld} and {@link useHeldRef}. */
export interface UseHeldProps {
  triggers: Trigger[];
  /** Whether a superset of a trigger still counts as held. */
  loose?: boolean;
}

/**
 * Tracks which of the given triggers are down, in a ref rather than state. Use it in an
 * event handler that reads the modifier keys, where a re-render per keypress is waste.
 */
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

/**
 * Tracks which of the given triggers are down and re-renders the caller on every
 * change. Prefer {@link useHeldRef} when only an event handler reads the result.
 */
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
