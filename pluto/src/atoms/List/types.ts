// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentType, CSSProperties } from "react";

interface ListColumn {
  label: string;
  visible?: boolean;
  width?: number;
}

type RenderF<E extends RenderableRecord<E> = RenderableRecord> = ComponentType<{
  entry: E;
  style: CSSProperties;
}>;

export type UntypedListColumn = {
  key: string;
  render?: RenderF;
} & ListColumn;

export type TypedListColumn<E extends RenderableRecord<E>> = {
  key: keyof E;
  render?: RenderF<E>;
} & ListColumn;

export type RenderableRecord<E = Record<string, string | number | undefined>> = {
  key: string;
} & Partial<Record<keyof E, string | number | undefined>>;

export type UntypedListTransform = (data: RenderableRecord[]) => RenderableRecord[];

export type TypedListTransform<E extends RenderableRecord<E>> = (data: E[]) => E[];

export interface ListItemProps<E extends RenderableRecord<E>> {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: Array<TypedListColumn<E>>;
  onSelect: (key: string) => void;
}
