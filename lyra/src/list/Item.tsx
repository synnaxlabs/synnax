// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Item.css";

import { type optional, type record } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type MouseEventHandler,
  type PointerEventHandler,
  type ReactElement,
  useLayoutEffect,
  useRef,
} from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { useItemsContext } from "@/list/scope";
import { Menu } from "@/menu";

// aria-selected is only valid on these roles. A generic row, or a plain listitem,
// carries its selection in a nested control instead.
const SELECTABLE_ROLES = new Set([
  "columnheader",
  "gridcell",
  "option",
  "row",
  "rowheader",
  "tab",
  "treeitem",
]);

export interface ItemRenderProps<K extends record.Key = record.Key> {
  index: number;
  key: K;
  itemKey: K;
  className?: string;
  translate?: number;
}

export type ItemProps<
  K extends record.Key,
  E extends Button.ElementType = "div",
> = Omit<
  Button.ButtonProps<E>,
  "key" | "onSelect" | "translate" | "onClick" | "onPointerMove"
> &
  optional.Optional<ItemRenderProps<K>, "index" | "key"> & {
    draggingOver?: boolean;
    rightAligned?: boolean;
    onClick?: MouseEventHandler<HTMLElement>;
    onPointerMove?: PointerEventHandler<HTMLElement>;
    onSelect?: (key: K, e: MouseEvent<HTMLElement>) => void;
    selected?: boolean;
    hovered?: boolean;
    /** Called when a mouse or pen moves over the item. */
    onHover?: () => void;
  };

export type ItemRenderProp<K extends record.Key> = Component.RenderProp<
  ItemRenderProps<K>
>;

/**
 * itemNameID returns the DOM id for editable name text rendered inside the list item
 * with the given key. Item assigns the raw key as the row element's own id, so nested
 * editable text (e.g. Text.edit rename targets) must carry this derived id to keep
 * DOM ids unique.
 */
export const itemNameID = (itemKey: record.Key): string => `${itemKey}-name`;

export const Item = <K extends record.Key, E extends Button.ElementType = "div">({
  itemKey,
  className,
  index,
  el,
  draggingOver = false,
  rightAligned = false,
  selected = false,
  translate,
  onSelect,
  onClick,
  onPointerMove,
  hovered,
  onHover,
  style,
  role,
  ref,
  ...rest
}: ItemProps<K, E>): ReactElement => {
  // Offset with `top`, not a transform. A transform leaves the row's real box at the
  // top of the list, and the browser snapshots that box for the drag preview, so every
  // row but the first dragged with no image. The offset is the row's place in the whole
  // list, so it is written once at mount rather than on every scroll frame.
  const itemStyle = { top: translate, ...style };
  const handleClick: MouseEventHandler<HTMLElement> = (e) => {
    onSelect?.(itemKey, e);
    onClick?.(e);
  };
  // A touch drag scrolls the list rather than pointing at an item.
  const handlePointerMove: PointerEventHandler<HTMLElement> = (e) => {
    if (e.pointerType !== "touch") onHover?.();
    onPointerMove?.(e);
  };
  const elementRef = useRef<HTMLElement>(null);
  const combinedRef = useCombinedRefs<HTMLElement>(ref, elementRef);
  const setElement = useItemsContext();
  useLayoutEffect(() => {
    if (setElement == null) return;
    setElement(itemKey, elementRef.current);
    return () => setElement(itemKey, null);
  }, [setElement, itemKey]);
  return (
    <Button.Button
      el={el}
      ref={combinedRef}
      defaultEl="div"
      id={itemKey.toString()}
      variant="text"
      onClick={handleClick}
      onPointerMove={handlePointerMove}
      className={CSS.cls(
        className,
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        hovered && CSS.M("hovered"),
        rightAligned && CSS.M("right-aligned"),
        draggingOver && CSS.M("dragging-over"),
        CSS.BE("list", "item"),
        CSS.M("reveals"),
        CSS.selected(selected),
      )}
      style={itemStyle}
      square={false}
      role={role}
      aria-selected={role != null && SELECTABLE_ROLES.has(role) ? selected : undefined}
      {...rest}
    />
  );
};
