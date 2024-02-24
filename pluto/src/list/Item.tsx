// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Keyed, type Key } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type ItemProps } from "@/list/types";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";

import "@/list/Item.css";

export interface ItemFrameProps<K extends Key, E extends Keyed<K>>
  extends Omit<ItemProps<K, E>, "columns">,
    Omit<Align.SpaceProps, "key" | "style" | "onSelect" | "translate"> {
  draggingOver?: boolean;
  rightAligned?: boolean;
}

export const ItemFrame = <K extends Key, E extends Keyed<K>>({
  entry,
  selected,
  hovered,
  onSelect,
  className,
  draggingOver = false,
  rightAligned = false,
  translate,
  ...props
}: ItemFrameProps<K, E>): ReactElement => (
  <Align.Space
    id={entry.key.toString()}
    direction="x"
    onClick={() => onSelect?.(entry.key)}
    className={CSS(
      className,
      CONTEXT_TARGET,
      selected && CONTEXT_SELECTED,
      hovered && CSS.M("hovered"),
      rightAligned && CSS.M("right-aligned"),
      CSS.BE("list", "item"),
      CSS.selected(selected),
    )}
    style={{
      position: translate != null ? "absolute" : "relative",
      transform: `translateY(${translate}px)`,
    }}
    {...props}
  />
);
