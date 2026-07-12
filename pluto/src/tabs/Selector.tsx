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
  useLayoutEffect,
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

/** The dragging state a strip drop reports, plus the resolved insertion index. */
export interface SelectorDropProps extends Haul.OnDropProps {
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
  onDrop?: (props: SelectorDropProps) => Haul.Item[];
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

  const handleKeyDown: KeyboardEventHandler<HTMLDivElement> = (e) => {
    onKeyDown?.(e);
    const el = internalRef.current;
    if (el == null || e.defaultPrevented) return;
    const next = horizontal ? "ArrowRight" : "ArrowDown";
    const prev = horizontal ? "ArrowLeft" : "ArrowUp";
    if (![next, prev, "Home", "End"].includes(e.key)) return;
    // Only rove when a tab itself is focused: arrow keys pressed inside a tab's
    // children (an editable name, a close button) must keep their own meaning.
    if ((e.target as HTMLElement).getAttribute?.("role") !== "tab") return;
    const tabs = Array.from(el.querySelectorAll<HTMLElement>('[role="tab"]'));
    if (tabs.length === 0) return;
    let target: number;
    if (e.key === "Home") target = 0;
    else if (e.key === "End") target = tabs.length - 1;
    else {
      const current = tabs.indexOf(document.activeElement as HTMLElement);
      const delta = e.key === next ? 1 : -1;
      target = current === -1 ? 0 : (current + delta + tabs.length) % tabs.length;
    }
    e.preventDefault();
    tabs[target].focus();
  };

  const [insertionIndex, setInsertionIndex] = useState<number | null>(null);

  const handleCanDrop = useCallback<Haul.CanDrop>(
    (state) => {
      if (haulType === "") return false;
      if (canDrop != null) return canDrop(state);
      return Haul.filterByType(haulType, state.items).length > 0;
    },
    [haulType, canDrop],
  );

  const handleDragOver = useCallback(({ event }: Haul.OnDragOverProps): void => {
    const el = internalRef.current;
    if (event == null || el == null) return;
    setInsertionIndex(getInsertionIndex(el, { x: event.clientX, y: event.clientY }));
  }, []);

  const handleDrop = useCallback<Haul.OnDrop>(
    (props) => {
      const el = internalRef.current;
      setInsertionIndex(null);
      if (props.event == null || el == null || onDrop == null) return [];
      const index = getInsertionIndex(el, {
        x: props.event.clientX,
        y: props.event.clientY,
      });
      return onDrop({ ...props, index });
    },
    [onDrop],
  );

  const dropProps = Haul.useDrop({
    type: haulType,
    canDrop: handleCanDrop,
    onDrop: handleDrop,
    onDragOver: handleDragOver,
  });

  const handleDragLeave: DragEventHandler<HTMLDivElement> = (e) => {
    onDragLeave?.(e);
    setInsertionIndex(null);
  };

  const [indicatorOffset, setIndicatorOffset] = useState<number | null>(null);
  useLayoutEffect(() => {
    const el = internalRef.current;
    if (insertionIndex == null || el == null) {
      setIndicatorOffset(null);
      return;
    }
    const tabs = el.querySelectorAll<HTMLElement>(TAB_SELECTOR);
    if (tabs.length === 0) setIndicatorOffset(0);
    else if (insertionIndex < tabs.length) {
      const tab = tabs[Math.max(insertionIndex, 0)];
      setIndicatorOffset(horizontal ? tab.offsetLeft : tab.offsetTop);
    } else {
      const last = tabs[tabs.length - 1];
      setIndicatorOffset(
        horizontal
          ? last.offsetLeft + last.offsetWidth
          : last.offsetTop + last.offsetHeight,
      );
    }
  }, [insertionIndex, horizontal]);
  const indicatorStyle: CSSProperties | undefined =
    indicatorOffset == null
      ? undefined
      : { [horizontal ? "left" : "top"]: indicatorOffset };

  const ctx = useMemo<ContextValue>(() => ({ size, variant }), [size, variant]);
  const active = haulType !== "";

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
        {...(active ? { ...dropProps, onDragLeave: handleDragLeave } : {})}
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
