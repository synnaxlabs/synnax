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
  useEffect,
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
import { KEY_ATTRIBUTE, KEY_SELECTOR } from "@/tabs/Frame";
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

/** LIST_ROLE is the ARIA role a Selector renders on the tab strip. */
export const LIST_ROLE = "tablist";

/** LIST_SELECTOR matches the tab strip rendered by Selector via {@link LIST_ROLE}. */
export const LIST_SELECTOR = `[role="${LIST_ROLE}"]`;

export { useContext as useSelectorContext };

/**
 * getInsertionIndex returns the index at which a tab dropped at the given cursor
 * position should be inserted into the given selector element. The index ranges
 * from 0 (before the first tab) to the number of tabs (after the last tab).
 */
const getInsertionIndex = (selector: Element, cursor: xy.Crude): number => {
  const pos = xy.construct(cursor);
  const horizontal = selector.getAttribute("aria-orientation") !== "vertical";
  const tabs = selector.querySelectorAll(KEY_SELECTOR);
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
 * drop indicator for the given insertion index: the leading edge of the tab at the
 * index, or the trailing edge of the last tab past the end.
 */
const getIndicatorOffset = (
  selector: HTMLElement,
  index: number,
  horizontal: boolean,
): number => {
  const tabs = selector.querySelectorAll<HTMLElement>(KEY_SELECTOR);
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

const mainAxisSize = (el: HTMLElement, horizontal: boolean): number =>
  horizontal ? el.offsetWidth : el.offsetHeight;

/** Class marking the source tab whose slot is lifted out during a reorder drag. */
const HAULED_CLASS = CSS.BEM("tabs", "tab", "hauled");

/**
 * applyReorderPreview shifts the strip's tabs to open a gap for a same-strip drag,
 * mirroring the reorder a drop at index would produce, and lifts the source tab out.
 * Returns false when draggedKey names no tab here, so the caller can fall back to the
 * insertion indicator.
 */
const applyReorderPreview = (
  selector: HTMLElement,
  index: number,
  draggedKey: string,
  horizontal: boolean,
): boolean => {
  const tabs = Array.from(selector.querySelectorAll<HTMLElement>(KEY_SELECTOR));
  const dragged = tabs.findIndex((t) => t.getAttribute(KEY_ATTRIBUTE) === draggedKey);
  if (dragged === -1) return false;
  const gap = mainAxisSize(tabs[dragged], horizontal);
  const axis = horizontal ? "X" : "Y";
  tabs.forEach((tab, i) => {
    if (i === dragged) {
      tab.classList.add(HAULED_CLASS);
      tab.style.transform = "";
      return;
    }
    let shift = 0;
    if (dragged < i && i < index) shift = -gap;
    else if (index <= i && i < dragged) shift = gap;
    tab.style.transform = shift === 0 ? "" : `translate${axis}(${shift}px)`;
  });
  return true;
};

/**
 * resetTabs clears the transforms applyReorderPreview set. snap suppresses the
 * transition so a dropped reorder lands without the tabs sliding back from their old
 * layout; without it the reset animates, closing the gap for a cancelled drag.
 */
const resetTabs = (selector: HTMLElement, snap: boolean): void => {
  const tabs = Array.from(selector.querySelectorAll<HTMLElement>(KEY_SELECTOR));
  if (snap) tabs.forEach((tab) => (tab.style.transition = "none"));
  tabs.forEach((tab) => {
    tab.classList.remove(HAULED_CLASS);
    tab.style.transform = "";
  });
  if (!snap) return;
  void selector.offsetHeight;
  tabs.forEach((tab) => (tab.style.transition = ""));
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
  // The applied shift (insertion index + dragged key), or null when none. Doubles as
  // an early-out so a dragover that stays in slot skips rewriting the transforms.
  const previewRef = useRef<{ index: number; key: string } | null>(null);
  // Set on a drop over a live preview so the layout effect snaps the transforms away.
  const committingRef = useRef(false);

  const handleCanDrop = useCallback<Haul.CanDrop>(
    (state) => {
      if (haulType === "") return false;
      if (canDrop != null) return canDrop(state);
      return Haul.filterByType(haulType, state.items).length > 0;
    },
    [haulType, canDrop],
  );

  const handleDragOver = useCallback(
    ({ event, items }: Haul.OnDragOverProps): void => {
      const el = internalRef.current;
      if (event == null || el == null) return;
      const index = getInsertionIndex(el, { x: event.clientX, y: event.clientY });
      const [dragged] = Haul.filterByType(haulType, items);
      const key = dragged == null ? null : String(dragged.key);
      const prev = previewRef.current;
      if (prev != null && key != null && prev.index === index && prev.key === key)
        return;
      if (key != null && applyReorderPreview(el, index, key, horizontal)) {
        previewRef.current = { index, key };
        setIndicatorOffset(null);
      } else {
        previewRef.current = null;
        setIndicatorOffset(getIndicatorOffset(el, index, horizontal));
      }
    },
    [haulType, horizontal],
  );

  const handleDrop = useCallback<Haul.OnDrop>(
    (params) => {
      const el = internalRef.current;
      setIndicatorOffset(null);
      committingRef.current = previewRef.current != null;
      previewRef.current = null;
      if (params.event == null || el == null || onDrop == null) {
        committingRef.current = false;
        if (el != null) resetTabs(el, false);
        return [];
      }
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
      const el = internalRef.current;
      // dragleave also fires crossing onto a child tab; reset only on a true leave so
      // the preview doesn't flicker on every crossing.
      if (el != null && e.relatedTarget instanceof Node && el.contains(e.relatedTarget))
        return;
      setIndicatorOffset(null);
      if (el != null && previewRef.current != null) {
        resetTabs(el, false);
        previewRef.current = null;
      }
    },
    [onDragLeave],
  );

  // Snap the transforms away the frame a dropped reorder commits, before paint, so the
  // new order with stale transforms never shows.
  useLayoutEffect(() => {
    const el = internalRef.current;
    if (el == null || !committingRef.current) return;
    committingRef.current = false;
    resetTabs(el, true);
  });

  // Cleanup for a drag that ends without dropping here (Escape, dropped outside). The
  // drop path handles its own preview, so skip while a commit is pending.
  const dragging = Haul.useDraggingState();
  const wasDragging = useRef(false);
  useEffect(() => {
    const active = dragging.items.length > 0;
    const ended = wasDragging.current && !active;
    wasDragging.current = active;
    if (!ended) return;
    setIndicatorOffset(null);
    const el = internalRef.current;
    if (el != null && previewRef.current != null && !committingRef.current) {
      resetTabs(el, false);
      previewRef.current = null;
    }
  }, [dragging]);

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
        role={LIST_ROLE}
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
