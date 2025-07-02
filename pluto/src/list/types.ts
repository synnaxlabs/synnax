// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { type RenderProp } from "@/util/renderProp";

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
