// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Destructor, type status } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { useDebouncedCallback } from "@/hooks";
import { state } from "@/state";
import { useAdder } from "@/status/Aggregator";
import { Synnax } from "@/synnax";

/**
 * Arguments passed to the update function.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Input The type of data being updated
 */
export interface UpdateParams<
  Input extends core.Shape,
  Store extends core.Store,
  StatusData = undefined,
> {
  /** The data to be updated */
  data: Input;
  /** The Synnax client instance for making requests */
  client: Client;
  /** The store to update */
  store: Store;
  /** Set of rollback functions to execute if the update fails */
  rollbacks: Set<Destructor>;
  /** Set the status of the update */
  setStatus: (setter: state.SetArg<status.Status<StatusData>>) => void;
}

/**
 * Configuration arguments for creating an update query.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Input The type of data being updated
 */
export type CreateUpdateParams<
  Input extends core.Shape,
  ScopedStore extends core.Store,
  Output extends core.Shape = Input,
  StatusData = undefined,
> = {
  /** The name of the resource being updated (used for status messages) */
  name: string;
  verbs: core.Verbs;
  /** Function that performs the actual update operation */
  update: (
    params: UpdateParams<Input, ScopedStore, StatusData>,
  ) => Promise<Output | false>;
} & (StatusData extends undefined ? {} : { statusData: StatusData });

/**
 * Return type for the observable update hook.
 *
 * @template Input The type of data being updated
 */
export interface UseObservableUpdateReturn<Input extends core.Shape> {
  /** Function to trigger an update (fire-and-forget) */
  update: (data: Input, opts?: core.FetchOptions) => void;
  /** Function to trigger an update and await the result */
  updateAsync: (data: Input, opts?: core.FetchOptions) => Promise<boolean>;
}

/**
 * Arguments for the observable update hook.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Input The type of data being updated
 * @template Output The type of data returned by the update operation
 */
export interface UseObservableUpdateParams<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusData = undefined,
> {
  debounce?: number;
  /** Callback function to handle state changes */
  onChange: state.Setter<Result<Input | undefined, StatusData>>;
  /** The scope to use for the update operation */
  scope?: string;
  /** Function to run before the update operation. If the function returns undefined,
   * the update will be cancelled. */
  beforeUpdate?: (
    params: BeforeUpdateParams<Input>,
  ) => Promise<Input | boolean> | Input | boolean;
  /** Function to run after the update operation. */
  afterSuccess?: (params: AfterSuccessParams<Output>) => Promise<void> | void;
  /** Function to run after the update operation fails. */
  afterFailure?: (params: AfterFailureParams<Input>) => Promise<void> | void;
  statusData: StatusData;
}

export interface BeforeUpdateParams<Data extends core.Shape> {
  rollbacks: Set<Destructor>;
  client: Client;
  data: Data;
}

export interface AfterSuccessParams<Output extends core.Shape> {
  client: Client;
  data: Output;
}

export interface AfterFailureParams<Data extends core.Shape> {
  client: Client;
  status: status.Status<status.ExceptionDetails, "error">;
  data: Data;
}

/**
 * Arguments for the direct update hook.
 *
 * @template UpdateParams The type of parameters for the update operation
 */
export interface UseDirectUpdateParams<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusData = undefined,
> extends Omit<UseObservableUpdateParams<Input, Output, StatusData>, "onChange"> {}

/**
 * Return type for the direct update hook, combining result state with update functions.
 *
 * @template Input The type of data being updated
 */
export type UseDirectUpdateReturn<
  Input extends core.Shape,
  StatusData = undefined,
> = Result<Input | undefined, StatusData> & UseObservableUpdateReturn<Input>;

export interface UseObservableUpdate<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusData = undefined,
> {
  (
    args: UseObservableUpdateParams<Input, Output, StatusData>,
  ): UseObservableUpdateReturn<Input>;
}

export interface UseUpdate<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusData = undefined,
> {
  (
    args?: UseDirectUpdateParams<Input, Output, StatusData>,
  ): UseDirectUpdateReturn<Input, StatusData>;
}

/**
 * Return type for the createUpdate function.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Input The type of data being updated
 */
export interface CreateUpdateReturn<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusData = undefined,
> {
  /** Hook that provides update functions with external state management */
  useObservableUpdate: UseObservableUpdate<Input, Output, StatusData>;
  /** Hook that provides update functions with internal state management */
  useUpdate: UseUpdate<Input, Output, StatusData>;
}

/**
 * Internal hook for observable updates with external state management.
 * @internal
 */
const useObservable = <
  Input extends core.Shape,
  Store extends core.Store,
  Output extends core.Shape = Input,
  StatusData = undefined,
>({
  onChange,
  update,
  name,
  verbs: { present, past, participle },
  debounce = 0,
  scope,
  beforeUpdate,
  afterSuccess,
  afterFailure,
  statusData,
}: UseObservableUpdateParams<Input, Output, StatusData> &
  CreateUpdateParams<
    Input,
    Store,
    Output,
    StatusData
  >): UseObservableUpdateReturn<Input> => {
  const client = Synnax.use();
  const store = useStore<Store>(scope);
  const addStatus = useAdder();
  const handleUpdate = useDebouncedCallback(
    async (data: Input, opts: core.FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;

      const rollbacks = new Set<Destructor>();
      const runRollbacks = () => {
        try {
          rollbacks.forEach((rollback) => rollback());
        } catch (error) {
          console.error(`failed to rollback changes to ${name}`, error);
        }
      };

      if (client == null) {
        onChange(nullClientResult(`${present} ${name}`));
        return false;
      }

      try {
        onChange((p) => pendingResult(`${participle} ${name}`, p.data));

        if (beforeUpdate != null) {
          const updatedValue = await beforeUpdate({ client, data, rollbacks });
          if (signal?.aborted === true) return false;
          if (updatedValue === false) {
            onChange(successResult(`${past} ${name}`, data));
            runRollbacks();
            return false;
          }
          if (updatedValue !== true) data = updatedValue;
        }

        const setStatus = (setter: state.SetArg<status.Status>) =>
          onChange((p) => {
            const nextStatus = state.executeSetter(setter, p.status);
            return {
              ...p,
              status: nextStatus,
              variant: nextStatus.variant,
            } as Result<Input>;
          });

        const output = await update({ client, data, store, rollbacks, setStatus });
        if (signal?.aborted === true) return false;
        onChange(successResult(`${past} ${name}`, data));

        if (output === false) return false;
        await afterSuccess?.({ client, data: output });
        return true;
      } catch (error) {
        runRollbacks();
        if (signal?.aborted === true) return false;

        const result = errorResult<Input | undefined>(`${present} ${name}`, error);
        const { status } = result;
        onChange(result);
        addStatus(status);
        await afterFailure?.({ client, status, data });

        return false;
      }
    },
    debounce,
    [name, onChange, beforeUpdate, afterSuccess, afterFailure],
  );
  const handleSyncUpdate = useCallback(
    (data: Input, opts?: core.FetchOptions) => void handleUpdate(data, opts),
    [handleUpdate],
  );
  return { update: handleSyncUpdate, updateAsync: handleUpdate };
};

/**
 * Internal hook for direct updates with internal state management.
 * @internal
 */
const useDirect = <
  Input extends core.Shape,
  ScopedStore extends core.Store = {},
  Output extends core.Shape = Input,
>({
  name,
  verbs,
  ...restParams
}: UseDirectUpdateParams<Input, Output> &
  CreateUpdateParams<Input, ScopedStore, Output>): UseDirectUpdateReturn<Input> => {
  const [result, setResult] = useState<Result<Input | undefined>>(
    successResult(`${verbs.past} ${name}`, undefined),
  );
  const methods = useObservable<Input, ScopedStore, Output>({
    ...restParams,
    verbs,
    name,
    onChange: setResult,
  });
  return { ...result, ...methods };
};

/**
 * Creates an update query system that provides hooks for updating data.
 *
 * This function creates a set of React hooks that handle data updates with
 * proper loading states, error handling, and optimistic updates. It provides
 * both observable and direct variants for different use cases.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 * @param createParams Configuration object containing the update function and resource name
 * @returns Object containing hooks for different update patterns
 *
 * @example
 * ```typescript
 * interface UserUpdateParams extends Params {
 *   userId: number;
 * }
 *
 * interface User {
 *   id: number;
 *   name: string;
 *   email: string;
 * }
 *
 * const userUpdate = createUpdate<UserUpdateParams, User>({
 *   name: "user",
 *   update: async ({ value, params, client }) => {
 *     await client.users.update(params.userId, value);
 *   }
 * });
 *
 * // Usage with external state management
 * const { update, updateAsync } = userUpdate.useObservable({
 *   params: { userId: 123 },
 *   onChange: (result) => {
 *     console.log("Update result:", result);
 *   }
 * });
 *
 * // Usage with internal state management
 * const { update, updateAsync, variant, error } = userUpdate.useDirect({
 *   params: { userId: 123 }
 * });
 *
 * // Trigger update
 * await updateAsync({ id: 123, name: "John", email: "john@example.com" });
 * ```
 */
export const createUpdate = <
  Input extends core.Shape,
  ScopedStore extends core.Store,
  Output extends core.Shape = Input,
>(
  createParams: CreateUpdateParams<Input, ScopedStore, Output>,
): CreateUpdateReturn<Input, Output> => ({
  useObservableUpdate: (params: UseObservableUpdateParams<Input, Output>) =>
    useObservable<Input, ScopedStore, Output>({ ...params, ...createParams }),
  useUpdate: (params?: UseDirectUpdateParams<Input, Output>) =>
    useDirect<Input, ScopedStore, Output>({ ...params, ...createParams }),
});
