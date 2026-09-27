// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type location, type record, TimeSpan } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef, useState } from "react";

import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";

/** Props for {@link useHover}. */
export interface UseHoverProps<K extends record.Key> {
  /** Position in the order hovered when the dialog opens. Defaults to none. */
  initialHover?: number;
  /**
   * Returns every option in the order the arrow keys walk. A new function means the
   * order may have changed.
   */
  getOrder: () => K[];
  /** Calls the listener when the order changes but getOrder does not. */
  subscribe: (listener: () => void) => destructor.Destructor;
  /** Clicks the option with the given key. Enter clicks the hovered option. */
  click: (key: K) => void;
  /** Brings the given option into view after the hover moves onto it. */
  scrollTo?: (key: K, direction: location.Y) => void;
  /**
   * When to answer keyboard triggers. Defaults to the enclosing dialog's visibility.
   */
  enableTriggers?: Triggers.Condition;
}

const UP_TRIGGER: Triggers.Trigger = ["ArrowUp"];
const DOWN_TRIGGER: Triggers.Trigger = ["ArrowDown"];
const SELECT_TRIGGER: Triggers.Trigger = ["Enter"];
const TRIGGERS: Triggers.Trigger[] = [UP_TRIGGER, DOWN_TRIGGER, SELECT_TRIGGER];

const INITIAL_HOVER_DELAY = TimeSpan.milliseconds(200).milliseconds;
const HOVER_INTERVAL = TimeSpan.milliseconds(100).milliseconds;

/** Return value for {@link useHover}. */
export interface UseHoverReturn<K extends record.Key> {
  /** The key the arrow keys currently rest on. */
  hover: K | undefined;
}

const resolveHover = <K extends record.Key>(
  order: K[],
  hover: K | undefined,
  initialHover: number,
): K | undefined => {
  if (hover != null && order.includes(hover)) return hover;
  if (initialHover < 0 || order.length === 0) return undefined;
  return order[initialHover >= order.length ? 0 : initialHover];
};

/**
 * Moves a hover cursor through the options with the arrow keys and clicks the hovered
 * option with Enter, scrolling the hovered option into view. Holding an arrow key
 * repeats. The hover is a key, so it survives options appearing or disappearing around
 * it.
 */
export const useHover = <K extends record.Key>({
  getOrder,
  subscribe,
  initialHover = -1,
  click,
  scrollTo,
  enableTriggers,
}: UseHoverProps<K>): UseHoverReturn<K> => {
  // The key the arrow keys last moved to, which may since have left the order.
  const movedRef = useRef<K | undefined>(undefined);
  const [hover, setHover] = useState<K | undefined>(undefined);
  const getOrderRef = useSyncedRef(getOrder);
  const resolve = useCallback(
    () => setHover(resolveHover(getOrder(), movedRef.current, initialHover)),
    [getOrder, initialHover],
  );
  useEffect(() => {
    resolve();
    return subscribe(resolve);
  }, [subscribe, resolve]);
  const scrollToRef = useSyncedRef(scrollTo);
  const { visible } = Dialog.useContext();
  const enabledRef = useSyncedRef<Triggers.Condition>(enableTriggers ?? visible);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (!Triggers.resolveCondition(enabledRef.current)) return;
      if (intervalRef.current != null) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      if (stage !== "start") return;

      if (Triggers.match(triggers, [SELECT_TRIGGER])) {
        const current = resolveHover(
          getOrderRef.current(),
          movedRef.current,
          initialHover,
        );
        if (current != null) click(current);
        return;
      }
      const move = () => {
        const order = getOrderRef.current();
        if (order.length === 0) return;
        const current = resolveHover(order, movedRef.current, initialHover);
        const pos = current == null ? -1 : order.indexOf(current);
        let next: number;
        if (Triggers.match(triggers, [UP_TRIGGER], { loose: true }))
          next = pos <= 0 ? order.length - 1 : pos - 1;
        else if (Triggers.match(triggers, [DOWN_TRIGGER], { loose: true }))
          next = pos >= order.length - 1 ? 0 : pos + 1;
        else return;
        movedRef.current = order[next];
        setHover(order[next]);
        scrollToRef.current?.(order[next], pos > next ? "bottom" : "top");
      };
      move();
      intervalRef.current = setTimeout(() => {
        intervalRef.current = setInterval(move, HOVER_INTERVAL);
      }, INITIAL_HOVER_DELAY);
    },
    [click, initialHover],
  );
  Triggers.use({
    triggers: TRIGGERS,
    callback: handleTrigger,
    loose: true,
  });
  return { hover };
};
