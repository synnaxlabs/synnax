import { type Synnax } from "@synnaxlabs/client";
import { useCallback, useEffect, useState } from "react";
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
import { Sync } from "@/query/sync";
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
 *
 *   // Optionally update parameters (e.g., if ID changed)
 *   // onParamsChange({ id: newId });
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
  /** Function to update the query parameters */
  onParamsChange: state.PureSetter<P>;
}

/**
 * Configuration for the form query hook that combines data fetching with form management.
 * The update function now returns void and uses callbacks to modify state, allowing for
 * more flexible update patterns and better separation of concerns.
 *
 * @template P - The type of the parameters (must be a Params record)
 * @template Z - The Zod schema type for form validation
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
export interface UseFormArgs<P extends Params, Z extends z.ZodObject>
  extends UseArgs<P, z.infer<Z> | null> {
  /** Initial values for the form fields */
  initialValues: z.infer<Z>;
  /** Whether to automatically save form changes (not currently implemented) */
  autoSave?: boolean;
  /** Zod schema for form validation */
  schema: Z;
  /** Function to save form data to the server (now returns void, uses callbacks to update state) */
  update: (args: UpdateArgs<P, Z>) => Promise<void>;
  /** Optional callback to run after successful update (excludes onChange/onParamsChange) */
  afterUpdate?: (
    args: Omit<UpdateArgs<P, Z>, "onChange" | "onParamsChange">,
  ) => Promise<void>;
}

/**
 * Return type for form query hooks, combining query state with form management.
 *
 * @template Z - The Zod schema type for form validation
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
export type UseFormReturn<Z extends z.ZodType> = Omit<Result<z.infer<Z>>, "data"> & {
  /** Form management utilities for binding inputs and validation */
  form: Form.UseReturn<Z>;
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
}: UseFormArgs<P, Z>): UseFormReturn<Z> => {
  const [status, setStatus, statusRef] = useCombinedStateAndRef<
    Result<z.infer<Z> | null>
  >(loadingResult(name));
  const form = Form.use<Z>({ schema, values: initialValues });
  const client = PSynnax.use();

  const [params, setParams] = useState(propsParams);
  const memoPropsParams = useMemoDeepEqual(propsParams);
  useEffect(() => {
    setParams(propsParams);
  }, [memoPropsParams]);

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
          onParamsChange: setParams,
        });
        await afterUpdate?.({ client, params, values: result, form });
        handleResultChange(successResult(name, result));
      } catch (error) {
        setStatus(errorResult(name, error));
      }
    })();
  }, [client, form, name, params, update, afterUpdate]);

  const addListener = Sync.useAddListener();
  useBase({
    retrieve,
    listeners,
    name,
    params,
    onChange: handleResultChange,
    client,
    addListener,
  });

  return { form, save: handleSave, ...status };
};
