// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@synnaxlabs/x";
import { set } from "react-hook-form";

import { type aether } from "@/aether/aether";
import { type Factory } from "@/telem/core/factory";
import { type Spec } from "@/telem/core/telem";

export interface Provider {
  key: string;
  use: <T>(key: string, props: Spec, extension?: Factory) => UseResult<T>;
}

const CONTEXT_KEY = "pluto-telem-context";

export const getProvider = (ctx: aether.Context): Provider =>
  ctx.get<Provider>(CONTEXT_KEY);

export const setProvider = (ctx: aether.Context, prov: Provider): void =>
  ctx.set(CONTEXT_KEY, prov);

export const hijackProvider = (
  ctx: aether.Context,
  prov: Provider,
): Provider | null => {
  const old = getProvider(ctx);
  if (old.key === prov.key) return null;
  setProvider(ctx, prov);
  return prov;
};

export const use = <T>(
  ctx: aether.Context,
  key: string,
  props: Spec,
  extension?: Factory,
): UseResult<T> => getProvider(ctx).use<T>(key, props, extension);

export type UseResult<T> = [T, Destructor];
