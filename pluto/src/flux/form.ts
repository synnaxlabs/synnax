// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import { type z } from "zod";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  type Status,
  successResult,
} from "@/flux/result";
import {
  type RetrieveMountListenersParams,
  type RetrieveParams,
} from "@/flux/retrieve";
import { type UpdateParams } from "@/flux/update";
import { Form } from "@/form";
import { useAsyncEffect, useDestructors } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { useAdder } from "@/status/Aggregator";
import { Synnax } from "@/synnax";

export interface FormUpdateParams<
  Schema extends z.ZodType<core.Shape>,
  ScopedStore extends core.Store = {},
> extends Omit<UpdateParams<z.infer<Schema>, ScopedStore>, "data" | "onChange">,
    Omit<Form.UseReturn<Schema>, "setStatus"> {}

export interface FormRetrieveParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store = {},
> extends Form.UseReturn<Schema>,
    RetrieveParams<Query, Store> {}

/**
 * Configuration arguments for creating a form query.
 *
 * @template Query The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 */
export interface CreateFormParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> {
  name: string;
  /** Zod schema for form validation */
  schema: Schema;
  /** Default values to use when creating new forms */
  initialValues: z.infer<Schema>;
  update: (args: FormUpdateParams<Schema, Store>) => Promise<void>;
  retrieve: (args: FormRetrieveParams<Query, Schema, Store>) => Promise<void>;
  mountListeners?: (
    args: FormMountListenersParams<Query, Schema, Store>,
  ) => Destructor | Destructor[];
}

/**
 * Return type for the form hook, providing form management utilities.
 *
 * @template Schema The Zod schema type for form validation
 */
export type UseFormReturn<Schema extends z.ZodType<core.Shape>> = Omit<
  Result<z.infer<Schema>>,
  "data"
> & {
  /** Form management utilities for binding inputs and validation */
  form: Form.UseReturn<Schema>;
  /** Function to save the current form values */
  save: (opts?: core.FetchOptions) => void;
};

/**
 * Arguments passed to the afterSave callback.
 *
 * @template Query The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 */
export interface FormBeforeSaveParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends Form.UseReturn<Schema>,
    RetrieveParams<Query, Store> {}

interface FormMountListenersParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends Form.UseReturn<Schema>,
    Omit<RetrieveMountListenersParams<Query, core.Shape, Store>, "onChange"> {}

/**
 * Arguments passed to the afterSave callback.
 *
 * @template Query The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 */
export interface AfterSaveParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends FormBeforeSaveParams<Query, Schema, Store> {}

export interface BeforeValidateArgs<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends FormBeforeSaveParams<Query, Schema, Store> {}

/**
 * Arguments for using a form hook.
 *
 * @template Query The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 */
export interface UseFormArgs<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends Pick<Form.UseArgs<Schema>, "sync" | "onHasTouched" | "mode"> {
  /** Initial values for the form fields */
  initialValues?: z.infer<Schema>;
  /** Whether to automatically save form changes */
  autoSave?: boolean;
  /** Parameters for the form query */
  params: Query;
  /** Function to run before the validation operation. If the function returns undefined,
   * the validation will be cancelled. */
  beforeValidate?: (args: BeforeValidateArgs<Query, Schema, Store>) => boolean | void;
  /** Function to run before the save operation. If the function returns undefined,
   * the save will be cancelled. */
  beforeSave?: (args: FormBeforeSaveParams<Query, Schema, Store>) => Promise<boolean>;
  /** Callback function called after successful save */
  afterSave?: (args: AfterSaveParams<Query, Schema, Store>) => void;
  /** The scope to use for the form operation */
  scope?: string;
}

/**
 * Form hook function signature.
 *
 * @template Query The type of parameters for the form query
 * @template Schema The Zod schema type for form validation
 */
export interface UseForm<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> {
  (args: UseFormArgs<Query, Schema, Store>): UseFormReturn<Schema>;
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
 * @template Query The type of parameters for the form query
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
    Query extends core.Shape,
    Schema extends z.ZodType<core.Shape>,
    Store extends core.Store = {},
  >({
    name,
    schema,
    retrieve,
    mountListeners,
    update,
    initialValues: baseInitialValues,
  }: CreateFormParams<Query, Schema, Store>): UseForm<Query, Schema, Store> =>
  ({
    params: query,
    initialValues,
    autoSave = false,
    afterSave,
    beforeSave,
    beforeValidate,
    sync,
    onHasTouched,
    mode,
    scope: argsScope,
  }) => {
    const [result, setResult] = useState<Result<undefined>>(
      loadingResult(`retrieving ${name}`),
    );
    const scope = useUniqueKey(argsScope);
    const client = Synnax.use();
    const store = useStore<Store>(scope);
    const listeners = useDestructors();
    const addStatus = useAdder();

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
      async (query: Query, options: core.FetchOptions = {}) => {
        const { signal } = options;
        try {
          if (client == null)
            return setResult(nullClientResult<undefined>(`retrieve ${name}`));
          setResult((p) => loadingResult(`retrieving ${name}`, p.data));
          if (signal?.aborted) return;
          const args = { client, query, store, ...form, set: noNotifySet };
          await retrieve(args);
          if (signal?.aborted) return;
          listeners.cleanup();
          listeners.set(mountListeners?.(args));
          setResult(
            successResult<undefined, undefined>(
              `retrieved ${name}`,
              undefined,
              undefined,
            ),
          );
        } catch (error) {
          if (signal?.aborted) return;
          const res = errorResult(`retrieve ${name}`, error);
          addStatus(res.status);
          setResult(res);
        }
      },
      [client, name, form, store, noNotifySet],
    );
    const memoQuery = useMemoDeepEqual(query);
    useAsyncEffect(
      async (signal) => await retrieveAsync(memoQuery, { signal }),
      [retrieveAsync, memoQuery],
    );

    const saveAsync = useCallback(
      async (opts: core.FetchOptions = {}): Promise<boolean> => {
        const { signal } = opts;
        const rollbacks = new Set<Destructor>();
        try {
          if (client == null) {
            setResult(nullClientResult<undefined>(`update ${name}`));
            return false;
          }
          const args = { client, query, store, rollbacks, ...form, set: noNotifySet };
          if (beforeValidate?.(args) === false) return false;
          if (!(await form.validateAsync())) return false;
          setResult(loadingResult(`updating ${name}`, undefined));
          if ((await beforeSave?.(args)) === false) {
            setResult(successResult(`updated ${name}`, undefined));
            return false;
          }
          if (signal?.aborted === true) return false;
          const setStatus = (setter: state.SetArg<Status<never>>) =>
            setResult((p) => {
              const nextStatus = state.executeSetter(setter, p.status);
              return {
                ...p,
                status: nextStatus,
                variant: nextStatus.variant,
              } as Result<undefined>;
            });

          await update({ ...args, setStatus });
          setResult(successResult(`updated ${name}`, undefined));
          if (afterSave != null) afterSave(args);
          return true;
        } catch (error) {
          try {
            rollbacks.forEach((rollback) => rollback());
          } catch (rollbackError) {
            console.error("Error rolling back changes:", rollbackError);
          }
          if (signal?.aborted === true) return false;
          const res = errorResult(`update ${name}`, error);
          addStatus(res.status);
          setResult(res);
          return false;
        }
      },
      [name, query, beforeSave, afterSave, beforeValidate],
    );
    const save = useCallback(
      (opts?: core.FetchOptions) => void saveAsync(opts),
      [saveAsync],
    );

    return { form, save, ...result };
  };
