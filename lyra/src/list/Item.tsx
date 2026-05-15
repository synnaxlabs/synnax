// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x/record";
import "@/list/Item.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import type { Component } from "@/component";
import { CSS } from "@/css";
import { Menu } from "@/menu";
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

export type ItemRenderProp<K extends record.Key> = Component.RenderProp<
  ItemRenderProps<K>
>;

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
  <Button.Button
    // Cast needed because Button is wrapped by Tooltip.wrap which loses generic type info
    el={el as Button.ElementType}
    defaultEl="div"
    id={itemKey.toString()}
    variant="text"
    onClick={(e: any) => {
      onSelect?.(itemKey);
      onClick?.(e);
    }}
    className={CSS(
      className,
      Menu.CONTEXT_TARGET,
      selected && Menu.CONTEXT_SELECTED,
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
