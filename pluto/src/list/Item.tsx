// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Item.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";

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
> = Omit<Button.ButtonProps<E>, "key" | "onSelect" | "translate"> &
  ItemRenderProps<K> & {
    draggingOver?: boolean;
    rightAligned?: boolean;
    highlightHovered?: boolean;
    onSelect?: (key: K) => void;
    selected?: boolean;
    hovered?: boolean;
  };

export type ItemRenderProp<K extends record.Key> = RenderProp<ItemRenderProps<K>>;

export const Item = <K extends record.Key, E extends Button.ElementType = "div">({
  itemKey,
  className,
  index,
  el,
  draggingOver = false,
  rightAligned = false,
  highlightHovered = false,
  selected = false,
  translate,
  onSelect,
  onClick,
  hovered,
  style,
  ...rest
}: ItemProps<K, E>): ReactElement => (
  // @ts-expect-error - generic element issues
  <Button.Button<E>
    el={el}
    defaultEl="div"
    id={itemKey.toString()}
    variant="text"
    onClick={(e: any) => {
      onSelect?.(itemKey);
      onClick?.(e);
    }}
    className={CSS(
      className,
      CONTEXT_TARGET,
      selected && CONTEXT_SELECTED,
      hovered && CSS.M("hovered"),
      rightAligned && CSS.M("right-aligned"),
      highlightHovered && CSS.M("highlight-hover"),
      draggingOver && CSS.M("dragging-over"),
      CSS.BE("list", "item"),
      CSS.selected(selected),
    )}
    style={{
      position: translate != null ? "absolute" : "relative",
      transform: `translateY(${translate}px)`,
      ...style,
    }}
    square={false}
    {...rest}
  />
);
