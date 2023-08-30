// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { Factory } from "@/telem/core/factory";
import { Spec } from "@/telem/core/telem";

export interface Provider {
  use: <T>(key: string, props: Spec, extension?: Factory) => UseResult<T>;
}

const CONTEXT_KEY = "pluto-telem-context";

export const get = (ctx: aether.Context): Provider => ctx.get<Provider>(CONTEXT_KEY);

export const set = (ctx: aether.Context, prov: Provider): void =>
  ctx.set(CONTEXT_KEY, prov);

export const use = <T>(
  ctx: aether.Context,
  key: string,
  props: Spec,
  extension?: Factory
): UseResult<T> => get(ctx).use<T>(key, props, extension);

export type UseResult<T> = [T, Destructor];
