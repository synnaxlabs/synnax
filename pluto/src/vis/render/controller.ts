// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";

export interface Requestor {
  (reason: string): void;
}

export type RenderReason = "layout" | "data" | "tool";

const CONTEXT_KEY = "pluto-vis-renderer";

export const control = (ctx: aether.Context, f: Requestor): void => {
  ctx.set(CONTEXT_KEY, f, false);
};

export const useRequestor = (ctx: aether.Context): Requestor =>
  ctx.get<Requestor>(CONTEXT_KEY);

export const useOptionalRequestor = (ctx: aether.Context): Requestor | null =>
  ctx.getOptional<Requestor>(CONTEXT_KEY);

export const request = (ctx: aether.Context, reason: RenderReason): void =>
  useRequestor(ctx)(reason);
