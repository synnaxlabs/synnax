// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type RenderableValue } from "@/renderable";

export type Key = string | number;

export type KeyedRecord<
  K extends Key = Key,
  E extends Record<string, unknown> = Record<string, unknown>,
> = {
  key: K;
} & Partial<Record<keyof E, unknown>>;

export type UnknownRecord<
  E extends Record<string | number, unknown> = Record<string | number, unknown>,
> = Partial<Record<keyof E, unknown>>;

export type RenderableRecord<
  E extends Record<string, RenderableValue> = Record<string, RenderableValue>,
> = E;

export type KeyedRenderableRecord<
  K extends Key = Key,
  E extends Record<string, RenderableValue> = Record<string, RenderableValue>,
> = KeyedRecord<K, E> & Omit<RenderableRecord<E>, "key">;
