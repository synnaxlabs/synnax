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
  type KeyboardEventHandler,
  type ReactElement,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
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

export interface ContextValue {
  /** size sets the height of the strip and the typography level of its tabs. */
  size: Component.Size;
  /** variant is the visual variant applied to the strip's tabs. */
  variant: Variant;
  /** altColor colors the selected tab's text with the primary theme color. */
  altColor: boolean;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tabs.SelectorContext",
  providerName: "Tabs.Selector",
});

export { useContext as useSelectorContext };

const [InsertionIndexProvider, useInsertionIndex] = context.create<number | null>({
  defaultValue: null,
  displayName: "Tabs.InsertionIndexContext",
});

/**
 * InsertionIndexProvider supplies a {@link Selector}'s insertion indicator index
 * from an enclosing drop target (e.g. a mosaic leaf) without threading a prop
 * through the consumer's tree. A Selector's own insertionIndex prop, when set,
 * takes precedence over the provided value.
 */
export { InsertionIndexProvider };

const TAB_SELECTOR = "[data-tab-key]";

/**
 * getInsertionIndex returns the index at which a tab dropped at the given cursor
 * position should be inserted into the given selector element. The index ranges
 * from 0 (before the first tab) to the number of tabs (after the last tab).
 */
export const getInsertionIndex = (selector: Element, cursor: xy.Crude): number => {
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

export interface SelectorProps extends Flex.BoxProps {
  /** size sets the height of the strip and the typography level of its tabs. */
  size?: Component.Size;
  /** variant is the visual variant applied to the strip's tabs. */
  variant?: Variant;
  /** altColor colors the selected tab's text with the primary theme color. */
  altColor?: boolean;
  /**
   * insertionIndex renders a drop indicator line before the tab at the given index
   * (or after the last tab when equal to the number of tabs). Pass null to hide the
   * indicator. Use {@link getInsertionIndex} to derive the index from a drag cursor.
   */
  insertionIndex?: number | null;
}

/**
 * Selector is the strip that lays out a Frame's tabs. It renders a tablist with
 * arrow-key roving focus (manual activation: focus moves, Enter or Space selects)
 * and owns the geometry for drag-and-drop insertion indicators via the
 * insertionIndex prop.
 */
export const Selector = ({
  ref,
  size = "medium",
  variant = "default",
  altColor = false,
  insertionIndex,
  className,
  children,
  direction,
  x,
  y,
  onKeyDown,
  empty,
  gap,
  ...rest
}: SelectorProps): ReactElement => {
  const isDefault = variant === "default";
  const internalRef = useRef<HTMLDivElement | null>(null);
  const combinedRef = useCombinedRefs(ref, internalRef);
  const contextInsertionIndex = useInsertionIndex();
  if (insertionIndex === undefined) insertionIndex = contextInsertionIndex;
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

  const ctx = useMemo<ContextValue>(
    () => ({ size, variant, altColor }),
    [size, variant, altColor],
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
