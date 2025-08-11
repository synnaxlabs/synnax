// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { type z } from "zod";

import { type FetchOptions, type Params } from "@/flux/aether/params";
import { errorResult, pendingResult, type Result } from "@/flux/result";
import { createRetrieve, type CreateRetrieveArgs } from "@/flux/retrieve";
import { createUpdate, type CreateUpdateArgs } from "@/flux/update";
import { Form } from "@/form";
import { useCombinedStateAndRef } from "@/hooks";
import { state } from "@/state";

/**
 * Configuration arguments for creating a form query.
 *
 * @template FormParams The type of parameters for the form query
 * @template DataSchema The Zod schema type for form validation
 */
export interface CreateFormArgs<
  FormParams extends Params,
  DataSchema extends z.ZodObject,
> extends CreateRetrieveArgs<FormParams, z.infer<DataSchema> | null>,
    CreateUpdateArgs<FormParams, z.infer<DataSchema>> {
  /** Zod schema for form validation */
  schema: DataSchema;
  /** Default values to use when creating new forms */
  initialValues: z.infer<DataSchema>;
}

/**
 * Return type for the form hook, providing form management utilities.
 *
 * @template DataSchema The Zod schema type for form validation
 */
export type UseFormReturn<DataSchema extends z.ZodObject> = Omit<
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
export interface AfterSaveArgs<FormParams extends Params, Z extends z.ZodObject> {
  /** The form management utilities */
  form: Form.UseReturn<Z>;
  /** The current form parameters */
  params: FormParams;
}

/**
 * Arguments for using a form hook.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface UseFormArgs<FormParams extends Params, Z extends z.ZodObject>
  extends Pick<Form.UseArgs<Z>, "sync" | "onHasTouched" | "mode"> {
  /** Initial values for the form fields */
  initialValues?: z.infer<Z>;
  /** Whether to automatically save form changes */
  autoSave?: boolean;
  /** Parameters for the form query */
  params: FormParams;
  /** Callback function called after successful save */
  afterSave?: (args: AfterSaveArgs<FormParams, Z>) => void;
}

/**
 * Form hook function signature.
 *
 * @template FormParams The type of parameters for the form query
 * @template Z The Zod schema type for form validation
 */
export interface UseForm<FormParams extends Params, Z extends z.ZodObject> {
  (args: UseFormArgs<FormParams, Z>): UseFormReturn<Z>;
}

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
export const createForm = <FormParams extends Params, Schema extends z.ZodObject>({
  name,
  schema,
  retrieve,
  listeners,
  update,
  initialValues: baseInitialValues,
}: CreateFormArgs<FormParams, Schema>): UseForm<FormParams, Schema> => {
  const retrieveHook = createRetrieve<FormParams, z.infer<Schema> | null>({
    name,
    retrieve,
    listeners,
  });
  const updateHook = createUpdate<FormParams, z.infer<Schema>>({ name, update });

  return ({
    params,
    initialValues,
    autoSave = false,
    afterSave,
    sync,
    onHasTouched,
    mode,
  }) => {
    const [result, setResult, resultRef] = useCombinedStateAndRef<
      Result<z.infer<Schema> | null>
    >(pendingResult(name, "retrieving", null, false));

    const form = Form.use<Schema>({
      schema,
      values: initialValues ?? baseInitialValues,
      onChange: ({ path }) => {
        if (autoSave && path !== "") handleSave();
      },
      sync,
      onHasTouched,
      mode,
    });

    const handleResultChange = useCallback(
      (
        setter: state.SetArg<Result<z.infer<Schema> | null>>,
        resetForm: boolean = true,
      ) => {
        const nextStatus = state.executeSetter(setter, resultRef.current);
        resultRef.current = nextStatus;
        if (nextStatus.data != null) {
          form.set("", nextStatus.data);
          if (resetForm) form.setCurrentStateAsInitialValues();
        }
        setResult(nextStatus);
      },
      [form],
    ) satisfies state.Setter<Result<z.infer<Schema> | null>>;

    retrieveHook.useEffect({ params, onChange: handleResultChange });

    const handleUpdateResultChange = useCallback(
      (setter: state.SetArg<Result<z.infer<Schema> | null>>) =>
        handleResultChange(setter, false),
      [handleResultChange],
    );

    const { updateAsync } = updateHook.useObservable({
      params,
      onChange: handleUpdateResultChange,
    });

    const handleSave = useCallback(
      (opts?: FetchOptions) =>
        void (async () => {
          try {
            if (!(await form.validateAsync())) return;
            await updateAsync(form.value(), opts);
            afterSave?.({ form, params });
            form.setCurrentStateAsInitialValues();
          } catch (error) {
            setResult((p) => errorResult(name, "update", error, p.listenersMounted));
          }
        })(),
      [form, updateAsync, afterSave, params],
    );

    return { form, save: handleSave, ...result };
  };
};
