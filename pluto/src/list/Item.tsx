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

import { Align } from "@/align";
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

export interface ItemProps<K extends record.Key>
  extends Omit<Align.SpaceProps, "key" | "onSelect" | "translate">,
    ItemRenderProps<K> {
  draggingOver?: boolean;
  rightAligned?: boolean;
  highlightHovered?: boolean;
  allowSelect?: boolean;
  onSelect?: (key: K) => void;
  selected?: boolean;
  hovered?: boolean;
}

export type ItemRenderProp<K extends record.Key> = RenderProp<ItemRenderProps<K>>;

export const Item = <K extends record.Key>({
  itemKey,
  className,
  index,
  direction = "x",
  draggingOver: __,
  rightAligned = false,
  highlightHovered = false,
  allowSelect = true,
  selected = false,
  translate,
  onSelect,
  onClick,
  hovered,
  style,
  ...rest
}: ItemProps<K>): ReactElement => (
  <Align.Space
    id={itemKey.toString()}
    direction={direction}
    onClick={(e) => {
      onSelect?.(itemKey);
      onClick?.(e);
    }}
    tabIndex={-1}
    className={CSS(
      className,
      CONTEXT_TARGET,
      selected && CONTEXT_SELECTED,
      hovered && CSS.M("hovered"),
      rightAligned && CSS.M("right-aligned"),
      highlightHovered && CSS.M("highlight-hover"),
      CSS.BE("list", "item"),
      CSS.selected(selected),
      CSS.shade(0),
      allowSelect && CSS.M("clickable"),
      CSS.M("text"),
    )}
    style={{
      position: translate != null ? "absolute" : "relative",
      transform: `translateY(${translate}px)`,
      ...style,
    }}
    {...rest}
  />
);
