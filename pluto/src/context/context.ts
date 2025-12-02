// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NotUndefined } from "@synnaxlabs/x";
import { createContext, use } from "react";

/**
 * Creates a required Context with a useContext hook that throws an error if the hook is
 * used outside of the context. The error message will be `{hookOrComponentName} must be
 * used within {providerName}`, where {hookOrComponentName} is the string passed to the
 * useContext hook.
 *
 * @param contextDisplayName - The display name of the context.
 * @param providerName - The name of the provider where the context is used.
 * @returns A tuple containing the context and the hook to use the context. The hook
 * will throw an error if used outside of the provider.
 */
export const createRequired = <T extends NotUndefined>(
  contextDisplayName: string,
  providerName: string,
): [React.Context<T | undefined>, (hookOrComponentName: string) => T] => {
  const ctx = createContext<T | undefined>(undefined);
  ctx.displayName = contextDisplayName;
  const useContext = (hookOrComponentName: string) => {
    const value = use(ctx);
    if (value === undefined)
      throw new Error(`${hookOrComponentName} must be used within ${providerName}`);
    return value;
  };
  return [ctx, useContext];
};
