// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export type Variant = "set" | "delete";

export const Z = <V extends z.ZodType>(value: V) =>
  z.object({
    variant: z.enum(["set", "delete"]),
    key: z.string(),
    value,
  });

export interface Set<K, V> {
  variant: "set";
  key: K;
  value: V;
}

export interface Delete<K, V> {
  variant: "delete";
  key: K;
  value?: V;
}

export type Change<K, V> = Set<K, V> | Delete<K, V>;
