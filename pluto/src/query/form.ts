// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { useCallback } from "react";
import { type z } from "zod/v4";

import { Form } from "@/form";
import { useCombinedStateAndRef } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import {
  errorResult,
  loadingResult,
  type Params,
  type Result,
  successResult,
  useBase,
} from "@/query/base";
import { type UseArgs } from "@/query/use";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

/**
 * Arguments passed to the update function when saving form data.
 * The update function now receives callbacks to modify the form state and parameters,
 * allowing for more flexible update patterns.
 *
 * @template P - The type of parameters passed to the query
 * @template Schema - The Zod schema type for form validation
 *
 * @example
 * ```typescript
 * const updateUser = async ({
 *   client,
 *   params,
 *   values,
 *   form,
 *   onChange,
 *   onParamsChange
 * }: UpdateArgs<{ id: string }, UserSchema>) => {
 *   // Update the user on the server
 *   await client.users.update(params.id, values);
 *
 *   // Update the local form state with any server-returned data
 *   onChange({ ...values, updatedAt: new Date().toISOString() });
 * };
 * ```
 */
export interface UpdateArgs<P extends Params, Schema extends z.ZodObject> {
  /** The Synnax client instance for making requests */
  client: Synnax;
  /** The parameters identifying what to update */
  params: P;
  /** The validated form values to save */
  values: z.infer<Schema>;
  /** The form instance for additional operations */
  form: Form.UseReturn<Schema>;
  /** Function to update the form state with new data */
  onChange: state.PureSetter<z.infer<Schema>>;
}

/**
 * Configuration for the form query hook that combines data fetching with form management.
 * The update function now returns void and uses callbacks to modify state, allowing for
 * more flexible update patterns and better separation of concerns.
 *
 * @template QueryParams - The type of the parameters (must be a Params record)
 * @template DataSchema - The Zod schema type for form validation
 *
 * @example
 * ```typescript
 * const formArgs: UseFormArgs<{ id: string }, UserFormSchema> = {
 *   name: "user-form",
 *   params: { id: userId },
 *   initialValues: { name: "", email: "" },
 *   schema: UserFormSchema,
 *   retrieve: async ({ client, params }) => {
 *     return await client.users.get(params.id);
 *   },
 *   update: async ({ client, params, values, onChange }) => {
 *     await client.users.update(params.id, values);
 *     // Optionally update the form state with server response
 *     onChange({ ...values, lastModified: Date.now() });
 *   },
 *   listeners: [],
 *   afterUpdate: async ({ client, params }) => {
 *     // Optional post-update operations
 *     await client.audit.log("user_updated", params.id);
 *   }
 * };
 * ```
 */
export interface UseFormArgs<QueryParams extends Params, DataSchema extends z.ZodObject>
  extends UseArgs<QueryParams, z.infer<DataSchema> | null> {
  /** Initial values for the form fields */
  initialValues: z.infer<DataSchema>;
  /** Whether to automatically save form changes (not currently implemented) */
  autoSave?: boolean;
  /** Zod schema for form validation */
  schema: DataSchema;
  /** Function to save form data to the server (now returns void, uses callbacks to update state) */
  update: (args: UpdateArgs<QueryParams, DataSchema>) => Promise<void>;
  /** Optional callback to run after successful update (excludes onChange/onParamsChange) */
  afterUpdate?: (
    args: Omit<UpdateArgs<QueryParams, DataSchema>, "onChange" | "onParamsChange">,
  ) => Promise<void>;
}

/**
 * Return type for form query hooks, combining query state with form management.
 *
 * @template DataSchema - The Zod schema type for form validation
 *
 * @example
 * ```typescript
 * const { form, save, variant, error } = useForm(formArgs);
 *
 * if (variant === "loading") return <FormSkeleton />;
 * if (variant === "error") return <ErrorMessage error={error} />;
 *
 * return (
 *   <form onSubmit={(e) => { e.preventDefault(); save(); }}>
 *     <input {...form.bind("name")} />
 *     <input {...form.bind("email")} />
 *     <button type="submit" disabled={!form.canSubmit()}>
 *       Save
 *     </button>
 *   </form>
 * );
 * ```
 */
export type UseFormReturn<DataSchema extends z.ZodObject> = Omit<
  Result<z.infer<DataSchema>>,
  "data"
> & {
  /** Form management utilities for binding inputs and validation */
  form: Form.UseReturn<DataSchema>;
  /** Function to save the current form values */
  save: () => void;
};

/**
 * Form query hook that combines data fetching, form management, and real-time updates.
 * Automatically handles form validation, saving, and syncing with server state.
 *
 * @template K - The type of the key parameter (must be a primitive value)
 * @template Z - The Zod schema type for form validation
 * @param config - Configuration object with form schema, update function, and query settings
 * @returns Form management utilities and query state
 */
export const useForm = <P extends Params, Z extends z.ZodObject>({
  name,
  params: propsParams,
  initialValues,
  schema,
  retrieve,
  listeners,
  update,
  afterUpdate,
  autoSave = false,
}: UseFormArgs<P, Z>): UseFormReturn<Z> => {
  const [status, setStatus, statusRef] = useCombinedStateAndRef<
    Result<z.infer<Z> | null>
  >(loadingResult(name));

  const form = Form.use<Z>({
    schema,
    values: initialValues,
    onChange: ({ path }) => {
      if (autoSave && path !== "") handleSave();
    },
  });
  const client = PSynnax.use();

  const params = useMemoDeepEqual(propsParams);

  const handleResultChange: state.Setter<Result<z.infer<Z> | null>> = (setter) => {
    const nextStatus = state.executeSetter(setter, {
      ...statusRef.current,
      data: form.value() as any,
    });
    if (nextStatus.data != null) {
      form.set("", nextStatus.data);
      form.setCurrentStateAsInitialValues();
    }
    setStatus(nextStatus);
  };

  const handleSave = useCallback(() => {
    void (async () => {
      try {
        if (client == null || !(await form.validateAsync())) return;
        let result: z.infer<Z> = form.value();
        await update({
          client,
          params,
          values: result,
          form,
          onChange: (v) => (result = v),
        });
        await afterUpdate?.({ client, params, values: result, form });
        handleResultChange(successResult(name, result));
      } catch (error) {
        setStatus(errorResult(name, error));
      }
    })();
  }, [client, form, name, params, update, afterUpdate]);

  useBase({
    retrieve,
    listeners,
    name,
    params,
    onChange: handleResultChange,
    client,
  });

  return { form, save: handleSave, ...status };
};
