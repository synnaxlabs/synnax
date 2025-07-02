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
import { CSS } from "@/css";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";
import { type RenderProp } from "@/util/renderProp";

export interface ItemFrameProps<K extends record.Key, E extends record.Keyed<K>>
  extends ItemProps<K, E>,
    Omit<Align.SpaceProps, "key" | "onSelect" | "translate"> {
  itemKey: K;
  draggingOver?: boolean;
  rightAligned?: boolean;
  highlightHovered?: boolean;
  allowSelect?: boolean;
  onSelect?: (key: K) => void;
  selected?: boolean;
  hovered?: boolean;
}

export interface ItemProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  index: number;
  key: K;
  itemKey: K;
  className?: string;
  translate?: number;
  useItem: (key: K) => E;
}

export type ItemRenderProp<
  K extends record.Key,
  E extends record.Keyed<K>,
> = RenderProp<ItemProps<K, E>>;

export const ItemFrame = <K extends record.Key, E extends record.Keyed<K>>({
  itemKey,
  className,
  direction = "x",
  draggingOver: __,
  rightAligned = false,
  highlightHovered = false,
  allowSelect = true,
  selected = false,
  translate,
  onSelect,
  hovered,
  style,
  ...rest
}: ItemFrameProps<K, E>): ReactElement => (
  <Align.Space
    id={itemKey.toString()}
    direction={direction}
    onClick={() => onSelect?.(itemKey)}
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
