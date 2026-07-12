// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type direction, xy } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type DragEventHandler,
  type KeyboardEventHandler,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Haul } from "@/haul";
import { useCombinedRefs } from "@/hooks";
import { Triggers } from "@/triggers";

/**
 * The visual variant of a tab selector.
 *
 * - `default`: flat strip with a bottom border and a primary-colored underline on the
 *   selected tab.
 * - `pill`: separated rounded buttons; the selected tab has a filled background and
 *   no underline.
 */
export type Variant = "default" | "pill";

interface ContextValue {
  /** size sets the height of the strip and the typography level of its tabs. */
  size: Component.Size;
  /** variant is the visual variant applied to the strip's tabs. */
  variant: Variant;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tabs.SelectorContext",
  providerName: "Tabs.Selector",
});

export { useContext as useSelectorContext };

const TAB_SELECTOR = "[data-tab-key]";

/**
 * getInsertionIndex returns the index at which a tab dropped at the given cursor
 * position should be inserted into the given selector element. The index ranges
 * from 0 (before the first tab) to the number of tabs (after the last tab).
 */
const getInsertionIndex = (selector: Element, cursor: xy.Crude): number => {
  const pos = xy.construct(cursor);
  const horizontal = selector.getAttribute("aria-orientation") !== "vertical";
  const tabs = selector.querySelectorAll(TAB_SELECTOR);
  let i = 0;
  for (const tab of tabs) {
    const center = box.center(box.construct(tab));
    if (horizontal ? pos.x < center.x : pos.y < center.y) return i;
    i++;
  }
  return tabs.length;
};

/**
 * getIndicatorOffset returns the pixel offset, along the strip's main axis, of the
 * drop indicator for a tab dropped at the given cursor position: the leading edge of
 * the tab at the insertion index, or the trailing edge of the last tab past the end.
 */
const getIndicatorOffset = (
  selector: HTMLElement,
  cursor: xy.Crude,
  horizontal: boolean,
): number => {
  const index = getInsertionIndex(selector, cursor);
  const tabs = selector.querySelectorAll<HTMLElement>(TAB_SELECTOR);
  if (tabs.length === 0) return 0;
  if (index < tabs.length) {
    const tab = tabs[index];
    return horizontal ? tab.offsetLeft : tab.offsetTop;
  }
  const last = tabs[tabs.length - 1];
  return horizontal
    ? last.offsetLeft + last.offsetWidth
    : last.offsetTop + last.offsetHeight;
};

/** The dragging state a strip drop reports, plus the resolved insertion index. */
export interface SelectorOnDropParams extends Haul.OnDropProps {
  /**
   * index is the strip slot the dragged item was dropped at, ranging from 0
   * (before the first tab) to the number of tabs (after the last tab).
   */
  index: number;
}

export interface SelectorProps extends Omit<Flex.BoxProps, "onDrop"> {
  /** size sets the height of the strip and the typography level of its tabs. */
  size?: Component.Size;
  /** variant is the visual variant applied to the strip's tabs. */
  variant?: Variant;
  /**
   * haulType enables drag-and-drop reordering by declaring the Haul item type the
   * strip accepts. When set, dragging an accepted item over the strip renders an
   * insertion indicator and dropping it calls onDrop with the target index. When
   * empty (the default), the strip is passive and registers no drop zone.
   */
  haulType?: string;
  /**
   * canDrop overrides the default acceptance predicate (any item whose type matches
   * haulType). Ignored when haulType is empty.
   */
  canDrop?: Haul.CanDrop;
  /**
   * onDrop fires when an accepted item is dropped on the strip, receiving the
   * dragging state and the resolved insertion index. Return the items the strip
   * consumed, matching the Haul drop contract. Ignored when haulType is empty.
   */
  onDrop?: (params: SelectorOnDropParams) => Haul.Item[];
}

/**
 * Selector is the strip that lays out a Frame's tabs. It renders a tablist with
 * arrow-key roving focus (manual activation: focus moves, Enter or Space selects).
 * Given a haulType it becomes a drag-and-drop target for reordering, owning the
 * insertion geometry and indicator and reporting drops through onDrop.
 */
export const Selector = ({
  ref,
  size = "medium",
  variant = "default",
  haulType = "",
  canDrop,
  onDrop,
  className,
  children,
  direction,
  x,
  y,
  onKeyDown,
  onDragLeave,
  empty,
  gap,
  ...rest
}: SelectorProps): ReactElement => {
  const isDefault = variant === "default";
  const internalRef = useRef<HTMLDivElement | null>(null);
  const combinedRef = useCombinedRefs(ref, internalRef);
  const dir: direction.Direction = Flex.parseDirection(direction, x, y) ?? "x";
  const horizontal = dir === "x";

  const handleKeyDown = useCallback<KeyboardEventHandler<HTMLDivElement>>(
    (e) => {
      onKeyDown?.(e);
      const el = internalRef.current;
      if (el == null || e.defaultPrevented) return;
      const key = Triggers.eventKey(e);
      const next = horizontal ? "ArrowRight" : "ArrowDown";
      const prev = horizontal ? "ArrowLeft" : "ArrowUp";
      if (![next, prev, "Home", "End"].includes(key)) return;
      // Only rove when a tab itself is focused: arrow keys pressed inside a tab's
      // children (an editable name, a close button) must keep their own meaning.
      if ((e.target as HTMLElement).getAttribute?.("role") !== "tab") return;
      const tabs = Array.from(el.querySelectorAll<HTMLElement>('[role="tab"]'));
      if (tabs.length === 0) return;
      let target: number;
      if (key === "Home") target = 0;
      else if (key === "End") target = tabs.length - 1;
      else {
        const current = tabs.indexOf(document.activeElement as HTMLElement);
        const delta = key === next ? 1 : -1;
        target = current === -1 ? 0 : (current + delta + tabs.length) % tabs.length;
      }
      e.preventDefault();
      tabs[target].focus();
    },
    [onKeyDown, horizontal],
  );

  const [indicatorOffset, setIndicatorOffset] = useState<number | null>(null);

  const handleCanDrop = useCallback<Haul.CanDrop>(
    (state) => {
      if (haulType === "") return false;
      if (canDrop != null) return canDrop(state);
      return Haul.filterByType(haulType, state.items).length > 0;
    },
    [haulType, canDrop],
  );

  const handleDragOver = useCallback(
    ({ event }: Haul.OnDragOverProps): void => {
      const el = internalRef.current;
      if (event == null || el == null) return;
      const cursor = { x: event.clientX, y: event.clientY };
      setIndicatorOffset(getIndicatorOffset(el, cursor, horizontal));
    },
    [horizontal],
  );

  const handleDrop = useCallback<Haul.OnDrop>(
    (params) => {
      const el = internalRef.current;
      setIndicatorOffset(null);
      if (params.event == null || el == null || onDrop == null) return [];
      const cursor = { x: params.event.clientX, y: params.event.clientY };
      return onDrop({ ...params, index: getInsertionIndex(el, cursor) });
    },
    [onDrop],
  );

  const dropProps = Haul.useDrop({
    type: haulType,
    canDrop: handleCanDrop,
    onDrop: handleDrop,
    onDragOver: handleDragOver,
  });

  const handleDragLeave = useCallback<DragEventHandler<HTMLDivElement>>(
    (e) => {
      onDragLeave?.(e);
      setIndicatorOffset(null);
    },
    [onDragLeave],
  );

  const indicatorStyle: CSSProperties | undefined =
    indicatorOffset == null
      ? undefined
      : { [horizontal ? "left" : "top"]: indicatorOffset };

  const ctx = useMemo<ContextValue>(() => ({ size, variant }), [size, variant]);
  // A passive strip registers no drop zone, so it still forwards the consumer's
  // onDragLeave rather than dropping it.
  const { onDragOver, onDrop: onDropHandler } = dropProps;
  const dropListeners = useMemo(
    () =>
      haulType === ""
        ? { onDragLeave }
        : { onDragOver, onDrop: onDropHandler, onDragLeave: handleDragLeave },
    [haulType, onDragOver, onDropHandler, handleDragLeave, onDragLeave],
  );

  return (
    <Context value={ctx}>
      <Flex.Box
        ref={combinedRef}
        role="tablist"
        aria-orientation={horizontal ? "horizontal" : "vertical"}
        className={CSS(
          CSS.BE("tabs", "selector"),
          CSS.BEM("tabs", "selector", variant),
          className,
        )}
        size={size}
        align="center"
        empty={empty ?? isDefault}
        gap={gap ?? (isDefault ? undefined : "small")}
        direction={dir}
        onKeyDown={handleKeyDown}
        {...rest}
        {...dropListeners}
      >
        {children}
        {indicatorStyle != null && (
          <span
            className={CSS(CSS.BE("tabs", "insertion"), CSS.M("direction", dir))}
            style={indicatorStyle}
          />
        )}
      </Flex.Box>
    </Context>
  );
};
