// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Item.css";

import { type Key, type Keyed, type Optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type ItemProps } from "@/list/types";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";

export interface ItemFrameProps<K extends Key, E extends Keyed<K>>
  extends Optional<ItemProps<K, E>, "sourceIndex">,
    Omit<Align.SpaceProps, "key" | "onSelect" | "translate"> {
  draggingOver?: boolean;
  rightAligned?: boolean;
  highlightHovered?: boolean;
  allowSelect?: boolean;
}

export const ItemFrame = <K extends Key, E extends Keyed<K>>({
  entry,
  selected,
  hovered,
  onSelect,
  className,
  direction = "x",
  draggingOver: __,
  rightAligned = false,
  highlightHovered = false,
  allowSelect = true,
  translate,
  style,
  sourceIndex: _,
  ...rest
}: ItemFrameProps<K, E>): ReactElement => (
  <Align.Space
    id={entry.key.toString()}
    direction={direction}
    onClick={() => onSelect?.(entry.key)}
    tabIndex={0}
    className={CSS(
      className,
      CONTEXT_TARGET,
      selected && CONTEXT_SELECTED,
      allowSelect && CSS.M("selectable"),
      hovered && CSS.M("hovered"),
      rightAligned && CSS.M("right-aligned"),
      highlightHovered && CSS.M("highlight-hover"),
      CSS.BE("list", "item"),
      CSS.selected(selected),
    )}
    style={{
      position: translate != null ? "absolute" : "relative",
      transform: `translateY(${translate}px)`,
      ...style,
    }}
    {...rest}
  />
);
