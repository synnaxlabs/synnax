// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Item.css";

import { type record } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type MouseEventHandler,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";

import { Button } from "@/button";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";

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
> = Omit<Button.ButtonProps<E>, "key" | "onSelect" | "translate" | "onClick"> &
  ItemRenderProps<K> & {
    draggingOver?: boolean;
    rightAligned?: boolean;
    onClick?: MouseEventHandler<HTMLElement>;
    onSelect?: (key: K, e: MouseEvent<HTMLElement>) => void;
    selected?: boolean;
    hovered?: boolean;
  };

export type ItemRenderProp<K extends record.Key> = RenderProp<ItemRenderProps<K>>;

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
  hovered,
  style,
  ...rest
}: ItemProps<K, E>): ReactElement => {
  const itemStyle = useMemo(
    () => ({
      position: translate != null ? ("absolute" as const) : ("relative" as const),
      transform: `translateY(${translate}px)`,
      ...style,
    }),
    [translate, style],
  );
  const handleClick = useCallback<MouseEventHandler<HTMLElement>>(
    (e) => {
      onSelect?.(itemKey, e);
      onClick?.(e);
    },
    [onSelect, onClick, itemKey],
  );
  return (
    <Button.Button
      el={el}
      defaultEl="div"
      id={itemKey.toString()}
      variant="text"
      onClick={handleClick}
      className={CSS(
        className,
        CONTEXT_TARGET,
        selected && CONTEXT_SELECTED,
        hovered && CSS.M("hovered"),
        rightAligned && CSS.M("right-aligned"),
        draggingOver && CSS.M("dragging-over"),
        CSS.BE("list", "item"),
        CSS.M("reveals"),
        CSS.selected(selected),
      )}
      style={itemStyle}
      square={false}
      {...rest}
    />
  );
};
