// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import {
  loadingResult,
  type Params,
  type Result,
  useObservable,
  type UseObservableArgs,
} from "@/query/observable";
import { type state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

/** Configuration arguments for the `use` hook. */
export interface UseArgs<QParams extends Params, Value extends state.State>
  extends Pick<UseObservableArgs<QParams, Value>, "retrieve" | "listeners" | "name"> {
  params: QParams;
}

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

export const useStateful = <P extends Params, V extends state.State>(
  query: UseReturn<V>,
): Result<V> => {
  const [result, setResult] = useState<Result<V>>(query);
  const client = PSynnax.use();
  const { retrieveAsync } = useObservable<P, V>({
    ...rest,
    name,
    client,
    onChange: setResult,
  });
};
