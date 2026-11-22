// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { aether } from "@/aether/aether";

export * from "@/aether/main";
export { Store } from "@/aether/store";

export type CallersFromSchema<T> = aether.CallersFromSchema<T>;
export type EmptyMethodsSchema = aether.EmptyMethodsSchema;
export type MethodsSchema = aether.MethodsSchema;
