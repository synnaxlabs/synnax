// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NotUndefined } from "@synnaxlabs/x";
import { createContext, use } from "react";

export interface BaseCreateParams {
  displayName: string;
}

export interface CreateWithDefaultParams<T> extends BaseCreateParams {
  defaultValue: T;
}

export interface CreateWithoutDefaultParams extends BaseCreateParams {
  providerName: string;
}

export interface Creator {
  <T>(params: CreateWithDefaultParams<T>): [React.Context<T>, () => T];
  <T extends NotUndefined>(
    params: CreateWithoutDefaultParams,
  ): [React.Context<T | undefined>, (hookOrComponentName: string) => T];
}

/**
 * Creates a tuple containing a context and hook to use the context.
 *
 * @param params.displayName - The display name of the context.
 * @param params.defaultValue - The default value of the context.
 * @param params.providerName - The name of the provider where the context is used.
 * @returns A tuple containing the context and the hook to use the context. If a default
 * value is not provided, the context will be of type `React.Context<T | undefined>`,
 * and the hook will throw an error if used outside of the context. If a default value
 * is provided, the context will be of type `React.Context<T>`, and the hook will return
 * the default value if used outside of the context.
 */
export const create: Creator = (<T>(
  params: CreateWithDefaultParams<T> | CreateWithoutDefaultParams,
):
  | [React.Context<T>, () => T]
  | [React.Context<T | undefined>, (hookOrComponentName: string) => T] => {
  if ("defaultValue" in params) {
    const { defaultValue, displayName } = params;
    const ctx = createContext(defaultValue);
    ctx.displayName = displayName;
    return [ctx, () => use(ctx)];
  }
  const ctx = createContext<T | undefined>(undefined);
  const { displayName, providerName } = params;
  ctx.displayName = displayName;
  const useContext = (hookOrComponentName: string) => {
    const value = use(ctx);
    if (value === undefined)
      throw new Error(`${hookOrComponentName} must be used within ${providerName}`);
    return value;
  };
  return [ctx, useContext];
}) as Creator;
