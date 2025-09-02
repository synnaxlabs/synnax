// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import { type z } from "zod";

import { type FetchOptions, type Params } from "@/flux/core/params";
import { type Store } from "@/flux/core/store";
import { useStore } from "@/flux/external";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type UpdateArgs as BaseUpdateArgs } from "@/flux/update";
import { Form } from "@/form";
import { useAsyncEffect, useDestructors } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoDeepEqual } from "@/memo";
import { type state } from "@/state";
import { Synnax } from "@/synnax";

export interface FormUpdateArgs<
  UpdateParams extends Params,
  Schema extends z.ZodType<state.State>,
  ScopedStore extends Store = {},
> extends Omit<
      BaseUpdateArgs<UpdateParams, z.infer<Schema>, ScopedStore>,
      "value" | "onChange"
    >,
    Form.UseReturn<Schema> {}

export interface FormRetrieveArgs<
  RetrieveParams extends Params,
  Schema extends z.ZodType<state.State>,
  ScopedStore extends Store = {},
> extends Form.UseReturn<Schema> {
  client: Client;
  params: RetrieveParams;
  store: ScopedStore;
}

/**
 * Configuration arguments for creating a form query.
 *
 * @template FormParams The type of parameters for the form query
 * @template DataSchema The Zod schema type for form validation
 */
export interface CreateFormArgs<
  FormParams extends Params,
  DataSchema extends z.ZodType<state.State>,
  SubStore extends Store,
> {
  name: string;
  /** Zod schema for form validation */
  schema: DataSchema;
  /** Default values to use when creating new forms */
  initialValues: z.infer<DataSchema>;
  update: (args: FormUpdateArgs<FormParams, DataSchema, SubStore>) => Promise<void>;
  retrieve: (args: FormRetrieveArgs<FormParams, DataSchema, SubStore>) => Promise<void>;
  mountListeners?: (
    args: FormMountListenersArgs<SubStore, FormParams, DataSchema>,
  ) => Destructor | Destructor[];
}

/**
 * Return type for the form hook, providing form management utilities.
 *
 * @template DataSchema The Zod schema type for form validation
 */
export type UseFormReturn<DataSchema extends z.ZodType<state.State>> = Omit<
  Result<z.infer<DataSchema>>,
  "data"
> & {
  /** Form management utilities for binding inputs and validation */
  form: Form.UseReturn<DataSchema>;
  /** Function to save the current form values */
  save: (opts?: FetchOptions) => void;
};

/**
 * Arguments passed to the afterSave callback.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface BeforeSaveArgs<
  FormParams extends Params,
  Z extends z.ZodType<state.State>,
> extends Form.UseReturn<Z> {
  client: Client;
  /** The current form parameters */
  params: FormParams;
}

interface FormMountListenersArgs<
  ScopedStore extends Store,
  FormParams extends Params,
  Schema extends z.ZodType<state.State>,
> extends Form.UseReturn<Schema> {
  store: ScopedStore;
  client: Client;
  params: FormParams;
}

/**
 * Arguments passed to the afterSave callback.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface AfterSaveArgs<
  FormParams extends Params,
  Z extends z.ZodType<state.State>,
> extends BeforeSaveArgs<FormParams, Z> {}

/**
 * Arguments for using a form hook.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface UseFormArgs<
  FormParams extends Params,
  Z extends z.ZodType<state.State>,
> extends Pick<Form.UseArgs<Z>, "sync" | "onHasTouched" | "mode"> {
  /** Initial values for the form fields */
  initialValues?: z.infer<Z>;
  /** Whether to automatically save form changes */
  autoSave?: boolean;
  /** Parameters for the form query */
  params: FormParams;
  /** Function to run before the save operation. If the function returns undefined,
   * the save will be cancelled. */
  beforeSave?: (args: BeforeSaveArgs<FormParams, Z>) => Promise<boolean>;
  /** Callback function called after successful save */
  afterSave?: (args: AfterSaveArgs<FormParams, Z>) => void;
  /** The scope to use for the form operation */
  scope?: string;
}

/**
 * Form hook function signature.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface UseForm<FormParams extends Params, Z extends z.ZodType<state.State>> {
  (args: UseFormArgs<FormParams, Z>): UseFormReturn<Z>;
}

const DEFAULT_SET_OPTIONS: Form.SetOptions = {
  markTouched: false,
  notifyOnChange: false,
};

/**
 * Creates a form query hook that combines data fetching, form management, and real-time updates.
 *
 * This function creates a React hook that automatically handles:
 * - Data fetching with loading states
 * - Form validation using Zod schemas
 * - Automatic form saving and persistence
 * - Real-time synchronization with server state
 * - Error handling and user feedback
 *
 * @template FormParams The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 * @param config Configuration object with form schema, update function, and query settings
 * @returns A React hook for managing the form
 *
 * @example
 * ```typescript
 * const userSchema = z.object({
 *   name: z.string().min(1),
 *   email: z.string().email(),
 *   age: z.number().optional()
 * });
 *
 * const useUserForm = createForm({
 *   name: "user",
 *   schema: userSchema,
 *   initialValues: { name: "", email: "", age: undefined },
 *   retrieve: async ({ params, client }) => {
 *     return await client.users.retrieve(params.userId);
 *   },
 *   update: async ({ value, params, client }) => {
 *     await client.users.update(params.userId, value);
 *   }
 * });
 *
 * // Usage in component
 * const { form, save, variant } = useUserForm({
 *   params: { userId: 123 },
 *   afterSave: ({ form }) => {
 *     console.log("User saved:", form.value());
 *   }
 * });
 * ```
 */
export const createForm =
  <
    FormParams extends Params,
    Schema extends z.ZodType<state.State>,
    SubStore extends Store = {},
  >({
    name,
    schema,
    retrieve,
    mountListeners,
    update,
    initialValues: baseInitialValues,
  }: CreateFormArgs<FormParams, Schema, SubStore>): UseForm<FormParams, Schema> =>
  ({
    params,
    initialValues,
    autoSave = false,
    afterSave,
    beforeSave,
    sync,
    onHasTouched,
    mode,
    scope: argsScope,
  }) => {
    const [result, setResult] = useState<Result<undefined>>(
      pendingResult(name, "retrieving", undefined),
    );
    const scope = useUniqueKey(argsScope);
    const client = Synnax.use();
    const store = useStore<SubStore>(scope);
    const listeners = useDestructors();

    const form = Form.use<Schema>({
      schema,
      values: initialValues ?? baseInitialValues,
      onChange: ({ path }) => {
        if (autoSave && path !== "") save();
      },
      sync,
      onHasTouched,
      mode,
    });
    const noNotifySet = useCallback(
      (path: string, value: unknown, options?: Form.SetOptions) =>
        form.set(path, value, { ...options, ...DEFAULT_SET_OPTIONS }),
      [form],
    );
    const retrieveAsync = useCallback(
      async (params: FormParams, options: FetchOptions = {}) => {
        const { signal } = options;
        try {
          if (client == null)
            return setResult(nullClientResult<undefined>(name, "retrieve"));
          setResult((p) => pendingResult(name, "retrieving", p.data));
          if (signal?.aborted) return;
          const args = { client, params, store, ...form, set: noNotifySet };
          await retrieve(args);
          if (signal?.aborted) return;
          listeners.cleanup();
          listeners.set(mountListeners?.(args));
          setResult(successResult<undefined>(name, "retrieved", undefined));
        } catch (error) {
          if (signal?.aborted) return;
          setResult(errorResult<undefined>(name, "retrieve", error));
        }
      },
      [client, name, form, store, noNotifySet],
    );
    const memoParams = useMemoDeepEqual(params);
    useAsyncEffect(
      async (signal) => await retrieveAsync(memoParams, { signal }),
      [retrieveAsync, memoParams],
    );

    const saveAsync = useCallback(
      async (opts: FetchOptions = {}): Promise<boolean> => {
        const { signal } = opts;
        try {
          if (client == null) {
            setResult(nullClientResult<undefined>(name, "update"));
            return false;
          }
          const args = { client, params, store, ...form, set: noNotifySet };
          if (!(await form.validateAsync())) return false;
          setResult(pendingResult(name, "updating", undefined));
          if ((await beforeSave?.(args)) === false) {
            setResult(successResult(name, "updated", undefined));
            return false;
          }
          if (signal?.aborted === true) return false;
          await update(args);
          setResult(successResult(name, "updated", undefined));
          if (afterSave != null) afterSave(args);
          return true;
        } catch (error) {
          if (signal?.aborted !== true)
            setResult(errorResult<undefined>(name, "update", error));
          return false;
        }
      },
      [name, params, beforeSave, afterSave],
    );
    const save = useCallback(
      (opts?: FetchOptions) => void saveAsync(opts),
      [saveAsync],
    );

    return { form, save, ...result };
  };
