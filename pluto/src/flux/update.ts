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

import { type FetchOptions } from "@/flux/core/params";
import { type Store } from "@/flux/core/store";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type state } from "@/state";
import { Synnax } from "@/synnax";

/**
 * Arguments passed to the update function.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 */
export interface UpdateArgs<Data extends state.State, ScopedStore extends Store = {}> {
  /** The data to be updated */
  value: Data;
  /** The Synnax client instance for making requests */
  client: Client;
  /** The store to update */
  store: ScopedStore;
  /** Set of rollback functions to execute if the update fails */
  rollbacks: Set<Destructor>;
}

/**
 * Configuration arguments for creating an update query.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 */
export interface CreateUpdateArgs<
  Data extends state.State,
  ScopedStore extends Store = {},
  OutputData extends state.State = Data,
> {
  /** The name of the resource being updated (used for status messages) */
  name: string;
  /** Function that performs the actual update operation */
  update: (args: UpdateArgs<Data, ScopedStore>) => Promise<OutputData | false>;
}

/**
 * Return type for the observable update hook.
 *
 * @template Data The type of data being updated
 */
export interface UseObservableUpdateReturn<Data extends state.State> {
  /** Function to trigger an update (fire-and-forget) */
  update: (value: Data, opts?: FetchOptions) => void;
  /** Function to trigger an update and await the result */
  updateAsync: (value: Data, opts?: FetchOptions) => Promise<boolean>;
}

/**
 * Arguments for the observable update hook.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 * @template OutputData The type of data returned by the update operation
 */
export interface UseObservableUpdateArgs<
  Data extends state.State,
  OutputData extends state.State = Data,
> {
  /** Callback function to handle state changes */
  onChange: state.Setter<Result<Data | undefined>>;
  /** The scope to use for the update operation */
  scope?: string;
  /** Function to run before the update operation. If the function returns undefined,
   * the update will be cancelled. */
  beforeUpdate?: (
    args: BeforeUpdateArgs<Data>,
  ) => Promise<Data | boolean> | Data | boolean;
  /** Function to run after the update operation. */
  afterSuccess?: (args: AfterSuccessArgs<OutputData>) => Promise<void> | void;
  /** Function to run after the update operation fails. */
  afterFailure?: (args: AfterFailureArgs<Data>) => Promise<void> | void;
}

export interface BeforeUpdateArgs<Data extends state.State> {
  rollbacks: Set<Destructor>;
  client: Client;
  value: Data;
}

export interface AfterSuccessArgs<Data extends state.State> {
  client: Client;
  value: Data;
}

export interface AfterFailureArgs<Data extends state.State> {
  client: Client;
  status: status.Status;
  value: Data;
}

/**
 * Arguments for the direct update hook.
 *
 * @template UpdateParams The type of parameters for the update operation
 */
export interface UseDirectUpdateArgs<
  Data extends state.State,
  OutputData extends state.State = Data,
> extends Omit<UseObservableUpdateArgs<Data, OutputData>, "onChange"> {}

/**
 * Return type for the direct update hook, combining result state with update functions.
 *
 * @template Data The type of data being updated
 */
export type UseDirectUpdateReturn<Data extends state.State> = Result<Data | undefined> &
  UseObservableUpdateReturn<Data>;

/**
 * Return type for the createUpdate function.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 */
export interface CreateUpdateReturn<
  Data extends state.State,
  OutputData extends state.State = Data,
> {
  /** Hook that provides update functions with external state management */
  useObservableUpdate: (
    args: UseObservableUpdateArgs<Data, OutputData>,
  ) => UseObservableUpdateReturn<Data>;
  /** Hook that provides update functions with internal state management */
  useUpdate: (
    args?: UseDirectUpdateArgs<Data, OutputData>,
  ) => UseDirectUpdateReturn<Data>;
}

/**
 * Internal hook for observable updates with external state management.
 * @internal
 */
const useObservable = <
  Data extends state.State,
  ScopedStore extends Store = {},
  OutputData extends state.State = Data,
>({
  onChange,
  update,
  name,
  scope,
  beforeUpdate,
  afterSuccess,
  afterFailure,
}: UseObservableUpdateArgs<Data, OutputData> &
  CreateUpdateArgs<Data, ScopedStore, OutputData>): UseObservableUpdateReturn<Data> => {
  const client = Synnax.use();
  const store = useStore<ScopedStore>(scope);
  const handleUpdate = useCallback(
    async (value: Data, opts: FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;
      const rollbacks = new Set<Destructor>();
      const runRollbacks = () => {
        console.log("running rollbacks");
        try {
          rollbacks.forEach((rollback) => rollback());
        } catch (error) {
          console.error(`failed to rollback changes to ${name}`, error);
        }
      };
      if (client == null) {
        onChange(nullClientResult(name, "update"));
        return false;
      }
      try {
        onChange((p) => pendingResult(name, "updating", p.data));
        if (beforeUpdate != null) {
          const updatedValue = await beforeUpdate({ client, value, rollbacks });
          if (updatedValue === false) {
            runRollbacks();
            return false;
          }
          if (updatedValue !== true) value = updatedValue;
        }
        const oValue = await update({
          client,
          value,
          store,
          rollbacks,
        });
        if (signal?.aborted === true || oValue == false) {
          runRollbacks();
          return false;
        }
        onChange(successResult(name, "updated", value));
        await afterSuccess?.({ client, value: oValue });
        return true;
      } catch (error) {
        runRollbacks();
        if (signal?.aborted !== true) {
          const result = errorResult<Data | undefined>(name, "update", error);
          onChange(result);
          await afterFailure?.({ client, status: result.status, value });
        }
        return false;
      }
    },
    [name],
  );
  const handleSyncUpdate = useCallback(
    (value: Data, opts?: FetchOptions) => void handleUpdate(value, opts),
    [handleUpdate],
  );
  return { update: handleSyncUpdate, updateAsync: handleUpdate };
};

/**
 * Internal hook for direct updates with internal state management.
 * @internal
 */
const useDirect = <
  Data extends state.State,
  ScopedStore extends Store = {},
  OutputData extends state.State = Data,
>({
  name,
  ...restArgs
}: UseDirectUpdateArgs<Data, OutputData> &
  CreateUpdateArgs<Data, ScopedStore, OutputData>): UseDirectUpdateReturn<Data> => {
  const [result, setResult] = useState<Result<Data | undefined>>(
    successResult(name, "updated", undefined),
  );
  const methods = useObservable<Data, ScopedStore, OutputData>({
    ...restArgs,
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
 * @param createArgs Configuration object containing the update function and resource name
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
  Data extends state.State,
  ScopedStore extends Store = {},
  OutputData extends state.State = Data,
>(
  createArgs: CreateUpdateArgs<Data, ScopedStore, OutputData>,
): CreateUpdateReturn<Data, OutputData> => ({
  useObservableUpdate: (args: UseObservableUpdateArgs<Data, OutputData>) =>
    useObservable<Data, ScopedStore, OutputData>({ ...args, ...createArgs }),
  useUpdate: (args?: UseDirectUpdateArgs<Data, OutputData>) =>
    useDirect<Data, ScopedStore, OutputData>({ ...args, ...createArgs }),
});
