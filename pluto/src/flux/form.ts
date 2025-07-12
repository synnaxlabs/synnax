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

import { type Params } from "@/flux/params";
import { errorResult, pendingResult, type Result } from "@/flux/result";
import { createRetrieve, type CreateRetrieveArgs } from "@/flux/retrieve";
import { createUpdate, type CreateUpdateArgs } from "@/flux/update";
import { Form } from "@/form";
import { useCombinedStateAndRef } from "@/hooks";
import { state } from "@/state";

export interface CreateFormArgs<
  FormParams extends Params,
  DataSchema extends z.ZodObject,
> extends CreateRetrieveArgs<FormParams, z.infer<DataSchema> | null>,
    CreateUpdateArgs<FormParams, z.infer<DataSchema>> {
  /** Zod schema for form validation */
  schema: DataSchema;
  initialValues: z.infer<DataSchema>;
}

export type UseFormReturn<DataSchema extends z.ZodObject> = Omit<
  Result<z.infer<DataSchema>>,
  "data"
> & {
  /** Form management utilities for binding inputs and validation */
  form: Form.UseReturn<DataSchema>;
  /** Function to save the current form values */
  save: () => void;
};

export interface AfterSaveArgs<FormParams extends Params, Z extends z.ZodObject> {
  form: Form.UseReturn<Z>;
  params: FormParams;
}

export interface UseFormArgs<FormParams extends Params, Z extends z.ZodObject> {
  /** Initial values for the form fields */
  initialValues?: z.infer<Z>;
  /** Whether to automatically save form changes (not currently implemented) */
  autoSave?: boolean;
  params: FormParams;
  afterSave?: (args: AfterSaveArgs<FormParams, Z>) => void;
}

export interface UseForm<FormParams extends Params, Z extends z.ZodObject> {
  (args: UseFormArgs<FormParams, Z>): UseFormReturn<Z>;
}

/**
 * Form query hook that combines data fetching, form management, and real-time updates.
 * Automatically handles form validation, saving, and syncing with server state.
 *
 * @template K - The type of the key parameter (must be a primitive value)
 * @template Z - The Zod schema type for form validation
 * @param config - Configuration object with form schema, update function, and query settings
 * @returns Form management utilities and query state
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
  return ({ params, initialValues, autoSave = false, afterSave }) => {
    const [result, setResult, resultRef] = useCombinedStateAndRef<
      Result<z.infer<Schema> | null>
    >(pendingResult(name, "retrieving"));

    const handleResultChange: state.Setter<Result<z.infer<Schema> | null>> = (
      setter,
    ) => {
      const nextStatus = state.executeSetter(setter, {
        ...resultRef.current,
        data: form.value() as any,
      });
      if (nextStatus.data != null) {
        form.set("", nextStatus.data);
        form.setCurrentStateAsInitialValues();
      }
      setResult(nextStatus);
    };

    retrieveHook.useEffect({ params, onChange: handleResultChange });

    const { updateAsync } = updateHook.useObservable({
      params,
      onChange: handleResultChange,
    });

    const form = Form.use<Schema>({
      schema,
      values: initialValues ?? baseInitialValues,
      onChange: ({ path }) => {
        if (autoSave && path !== "") handleSave();
      },
    });

    const handleSave = useCallback(
      () =>
        void (async () => {
          try {
            if (!(await form.validateAsync())) return;
            await updateAsync(form.value());
            afterSave?.({ form, params });
          } catch (error) {
            setResult(errorResult(name, "update", error));
          }
        })(),
      [form, updateAsync, afterSave, params],
    );

    return { form, save: handleSave, ...result };
  };
};
