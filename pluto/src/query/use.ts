// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import {
  loadingResult,
  type Params,
  type Result,
  useBase,
  type UseBaseArgs,
} from "@/query/base";
import { type state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

/**
 * Configuration arguments for the use hook.
 *
 * @template QParams - The type of parameters passed to the query
 * @template Value - The type of the state being managed
 *
 * @example
 * ```typescript
 * const queryArgs: UseArgs<{ workspaceId: string }, User[]> = {
 *   name: "users",
 *   params: { workspaceId: "workspace-123" },
 *   retrieve: async ({ client, params }) => {
 *     return await client.users.list(params.workspaceId);
 *   },
 *   listeners: [{
 *     channel: "users.changes",
 *     onChange: async ({ changed, onChange, client, params }) => {
 *       // Handle real-time updates
 *       const updatedUsers = await client.users.list(params.workspaceId);
 *       onChange(updatedUsers);
 *     }
 *   }]
 * };
 * ```
 */
export interface UseArgs<QParams extends Params, Value extends state.State>
  extends Pick<
    UseBaseArgs<QParams, Value>,
    "retrieve" | "listeners" | "name" | "params"
  > {}

/**
 * Return type for query hooks, representing the current state of a data fetch operation.
 * Uses a discriminated union to ensure type safety across different states.
 *
 * @template Data - The type of data being fetched
 *
 * @example
 * ```typescript
 * const result: UseReturn<User[]> = useQuery(args);
 *
 * if (result.variant === "loading") {
 *   return <div>Loading...</div>;
 * }
 *
 * if (result.variant === "error") {
 *   return <div>Error: {result.message}</div>;
 * }
 *
 * // result.variant === "success"
 * return <UserList users={result.data} />;
 * ```
 */
export type UseReturn<Data extends state.State> = Result<Data>;

/**
 * Main query hook for fetching data with optional real-time updates.
 * Handles loading states, error handling, and automatic re-fetching when parameters change.
 *
 * @template P - The type of parameters passed to the query
 * @template V - The type of the state being managed
 * @param config - Configuration object containing retrieve function, listeners, and parameters
 * @returns Query result with loading, error, or success state
 *
 * @example
 * ```typescript
 * const { data, variant, error } = use({
 *   name: "user-profile",
 *   params: { id: userId },
 *   retrieve: async ({ client, params }) => {
 *     return await client.users.get(params.id);
 *   },
 *   listeners: [{
 *     channel: "user.updates",
 *     onChange: async ({ changed, onChange, client, params }) => {
 *       const updatedUser = await client.users.get(params.id);
 *       onChange(updatedUser);
 *     }
 *   }]
 * });
 *
 * if (variant === "loading") return <Spinner />;
 * if (variant === "error") return <ErrorMessage error={error} />;
 * return <UserProfile user={data} />;
 * ```
 */
export const use = <P extends Params, V extends state.State>({
  retrieve,
  listeners,
  name,
  params,
}: UseArgs<P, V>): UseReturn<V> => {
  const [result, setResult] = useState<Result<V>>(loadingResult(name));
  const client = PSynnax.use();
  useBase<P, V>({
    retrieve,
    listeners,
    name,
    params,
    client,
    onChange: setResult,
  });
  return result;
};
