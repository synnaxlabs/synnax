// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type ItemProps } from "@/list/types";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";

import "@/list/Item.css";

export interface ItemFrameProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Omit<ItemProps<K, E>, "columns">,
    Omit<Align.SpaceProps, "key" | "style" | "onSelect"> {
  draggingOver?: boolean;
}

export const ItemFrame = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  entry,
  selected,
  hovered,
  onSelect,
  className,
  draggingOver = false,
  ...props
}: ItemFrameProps<K, E>): ReactElement => (
  <Align.Space
    direction="x"
    onClick={() => onSelect?.(entry.key)}
    onContextMenu={() => onSelect?.(entry.key)}
    className={CSS(
      className,
      CONTEXT_TARGET,
      selected && CONTEXT_SELECTED,
      hovered && CSS.M("hovered"),
      CSS.BE("list", "item"),
      CSS.selected(selected),
    )}
    {...props}
  />
);
