// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { CSSProperties } from "react";
import React from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { RenderProp } from "@/util/renderProp";

type RenderF<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> = RenderProp<{
  key: string | number | symbol;
  entry: E;
  style: CSSProperties;
}>;

export interface ListColumn<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  /** The key of the object to render. */
  key: keyof E | string;
  /** A custom render function for each item in the colummn. */
  render?: RenderF<K, E>;
  stringer?: (entry: E) => string;
  /** The name/title of the column. */
  name: string;
  /** Whether the column is visible by default. */
  visible?: boolean;
  /**
   * The width of the column in pixels. Used to structure the list as a table.
   * If not provided, the column will be sized to fit the content. This should
   * always be specified when the render function is provided.
   */
  width?: number;
  cWidth?: number;
}

export interface ListItemProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  key: string | number;
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: Array<ListColumn<K, E>>;
  onSelect?: (key: K) => void;
}
