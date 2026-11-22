// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { types } from "@/arc/graph/node/types";

export * from "@/arc/graph/node/Base";
export * from "@/arc/graph/node/registry";

export type PreviewProps<C extends object = object> = types.PreviewProps<C>;
export type SymbolProps<C extends object = object> = types.SymbolProps<C>;
export type Spec<T extends string = string, C extends object = object> = types.Spec<
  T,
  C
>;
