// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RenderableValue } from "@/renderable";

export type KeyedRecord<E extends Record<string, unknown> = Record<string, unknown>> = {
  key: string;
} & Partial<Record<keyof E, unknown>>;

export type UnknownRecord<E extends Record<string, unknown> = Record<string, unknown>> =
  Partial<Record<keyof E, unknown>>;

export type RenderableRecord<
  E extends Record<string, RenderableValue> = Record<string, RenderableValue>
> = Record<keyof E, RenderableValue>;

export type KeyedRenderableRecord<
  E extends Record<string, RenderableValue> = Record<string, RenderableValue>
> = KeyedRecord<E> & Omit<RenderableRecord<E>, "key">;
