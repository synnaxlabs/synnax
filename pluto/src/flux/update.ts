// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { useCallback, useState } from "react";

import { type FetchOptions, type Params } from "@/flux/core/params";
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
export interface UpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
  ScopedStore extends Store = {},
> {
  /** The data to be updated */
  value: Data;
  /** Parameters for the update operation */
  params: UpdateParams;
  /** The Synnax client instance for making requests */
  client: Client;
  /** Function to update the form state with new data */
  onChange: state.PureSetter<Data>;
  /** The store to update */
  store: ScopedStore;
}

/**
 * Configuration arguments for creating an update query.
 *
 * @template UpdateParams The type of parameters for the update operation
 * @template Data The type of data being updated
 */
export interface CreateUpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
  ScopedStore extends Store = {},
> {
  /** The name of the resource being updated (used for status messages) */
  name: string;
  /** Function that performs the actual update operation */
  update: (args: UpdateArgs<UpdateParams, Data, ScopedStore>) => Promise<void>;
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
 */
export interface UseObservableUpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
> {
  /** Callback function to handle state changes */
  onChange: state.Setter<Result<Data | undefined>>;
  /** Parameters for the update operation */
  params: UpdateParams;
  /** The scope to use for the update operation */
  scope?: string;
  /** Function to run before the update operation. If the function returns undefined,
   * the update will be cancelled. */
  beforeUpdate?: (args: BeforeUpdateArgs<Data>) => Promise<Data | boolean>;
  /** Function to run after the update operation. */
  afterUpdate?: (args: AfterUpdateArgs<Data>) => Promise<void>;
}

export interface BeforeUpdateArgs<Data extends state.State> {
  client: Client;
  value: Data;
}

export interface AfterUpdateArgs<Data extends state.State> {
  client: Client;
  value: Data;
}

/**
 * Arguments for the direct update hook.
 *
 * @template UpdateParams The type of parameters for the update operation
 */
export interface UseDirectUpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
> extends Omit<UseObservableUpdateArgs<UpdateParams, Data>, "onChange"> {}

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
  UpdateParams extends Params,
  Data extends state.State,
> {
  /** Hook that provides update functions with external state management */
  useObservable: (
    args: UseObservableUpdateArgs<UpdateParams, Data>,
  ) => UseObservableUpdateReturn<Data>;
  /** Hook that provides update functions with internal state management */
  useDirect: (
    args: UseDirectUpdateArgs<UpdateParams, Data>,
  ) => UseDirectUpdateReturn<Data>;
}

/**
 * Internal hook for observable updates with external state management.
 * @internal
 */
const useObservable = <
  UpdateParams extends Params,
  Data extends state.State,
  ScopedStore extends Store = {},
>({
  onChange,
  params,
  update,
  name,
  scope,
  beforeUpdate,
  afterUpdate,
}: UseObservableUpdateArgs<UpdateParams, Data> &
  CreateUpdateArgs<
    UpdateParams,
    Data,
    ScopedStore
  >): UseObservableUpdateReturn<Data> => {
  const client = Synnax.use();
  const store = useStore<ScopedStore>(scope);
  const handleUpdate = useCallback(
    async (value: Data, opts: FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;
      try {
        if (client == null) {
          onChange(nullClientResult(name, "update"));
          return false;
        }
        onChange((p) => pendingResult(name, "updating", p.data));
        let updated = false;
        if (beforeUpdate != null) {
          const updatedValue = await beforeUpdate({ client, value });
          if (updatedValue === false) return false;
          if (updatedValue !== true) value = updatedValue;
        }
        await update({
          client,
          onChange: (value) => {
            updated = true;
            onChange(successResult(name, "updated", value));
          },
          value,
          params,
          store,
        });
        if (signal?.aborted === true) return false;
        if (!updated) onChange(successResult(name, "updated", value));
        if (afterUpdate != null) await afterUpdate({ client, value });
        return true;
      } catch (error) {
        if (signal?.aborted !== true) onChange(errorResult(name, "update", error));
        return false;
      }
    },
    [name, params],
  );
  const handleSyncUpdate = useCallback(
    (value: Data, opts?: FetchOptions) => void handleUpdate(value, opts),
    [handleUpdate],
  );
  return {
    update: handleSyncUpdate,
    updateAsync: handleUpdate,
  };
};

/**
 * Internal hook for direct updates with internal state management.
 * @internal
 */
const useDirect = <
  UpdateParams extends Params,
  Data extends state.State,
  ScopedStore extends Store = {},
>({
  params,
  name,
  ...restArgs
}: UseDirectUpdateArgs<UpdateParams, Data> &
  CreateUpdateArgs<UpdateParams, Data, ScopedStore>): UseDirectUpdateReturn<Data> => {
  const [result, setResult] = useState<Result<Data | undefined>>(
    successResult(name, "updated", undefined),
  );
  const methods = useObservable<UpdateParams, Data, ScopedStore>({
    ...restArgs,
    name,
    onChange: setResult,
    params,
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
  UpdateParams extends Params,
  Data extends state.State,
  ScopedStore extends Store = {},
>(
  createArgs: CreateUpdateArgs<UpdateParams, Data, ScopedStore>,
): CreateUpdateReturn<UpdateParams, Data> => ({
  useObservable: (args: UseObservableUpdateArgs<UpdateParams, Data>) =>
    useObservable<UpdateParams, Data, ScopedStore>({ ...args, ...createArgs }),
  useDirect: (args: UseDirectUpdateArgs<UpdateParams, Data>) =>
    useDirect<UpdateParams, Data, ScopedStore>({ ...args, ...createArgs }),
});
