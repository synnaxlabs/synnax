// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import { type z } from "zod";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  type ResultStatus,
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
import { Status } from "@/status/core";
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

export interface CreateFormParams<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> {
  name: string;
  schema: Schema;
  initialValues: z.infer<Schema>;
  update: (args: FormUpdateParams<Schema, Store>) => Promise<void>;
  retrieve: (args: FormRetrieveParams<Query, Schema, Store>) => Promise<void>;
  mountListeners?: (
    args: FormMountListenersParams<Query, Schema, Store>,
  ) => destructor.Destructor | destructor.Destructor[];
}

export type UseFormReturn<Schema extends z.ZodType<core.Shape>> = Omit<
  Result<z.infer<Schema>>,
  "data"
> & {
  form: Form.UseReturn<Schema>;
  save: (opts?: core.FetchOptions) => void;
};

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

export interface UseFormArgs<
  Query extends core.Shape,
  Schema extends z.ZodType<core.Shape>,
  Store extends core.Store,
> extends Pick<Form.UseArgs<Schema>, "sync" | "onHasTouched" | "mode"> {
  initialValues?: z.infer<Schema>;
  autoSave?: boolean;
  query: Query;
  beforeValidate?: (args: BeforeValidateArgs<Query, Schema, Store>) => boolean | void;
  beforeSave?: (args: FormBeforeSaveParams<Query, Schema, Store>) => Promise<boolean>;
  afterSave?: (args: AfterSaveParams<Query, Schema, Store>) => void;
  scope?: string;
}

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
    query,
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
    const addStatus = Status.useAdder();

    const form = Form.use<Schema>({
      schema,
      values: initialValues ?? baseInitialValues,
      onChange: ({ path }) => {
        // Don't save if the path is empty to prevent infinite save loops.
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
          setResult(successResult<undefined>(`retrieved ${name}`));
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
        const rollbacks: destructor.Destructor[] = [];
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
          const setStatus = (setter: state.SetArg<ResultStatus<never>>) =>
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
            rollbacks.reverse().forEach((rollback) => rollback());
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
