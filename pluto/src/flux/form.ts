// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, type Synnax as Client } from "@synnaxlabs/client";
import { type CrudeTimeSpan, type destructor, state, TimeSpan } from "@synnaxlabs/x";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  type ResultStatus,
  successResult,
} from "@/flux/result";
import {
  type LocalCache,
  localFor,
  type RetrieveParams,
  suspendOnFetch,
  useMemoQuery,
  usePendingFetch,
} from "@/flux/suspend";
import { type UpdateParams } from "@/flux/update";
import { Form } from "@/form";
import { useDebouncedCallback, useDestructors, useSyncedRef } from "@/hooks";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";

export interface FormUpdateParams<Schema extends z.ZodType<query.Data>>
  extends
    Omit<UpdateParams<z.infer<Schema>>, "data" | "onChange" | "onOptimisticComplete">,
    Omit<Form.UseReturn<Schema>, "setStatus"> {}

/** Client and query handles for a form operation. */
interface FormClientParams<Query extends query.Params> {
  client: Client;
  query: Query | null;
}

export interface CreateFormParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
> {
  name: string;
  schema: Schema;
  initialValues: z.infer<Schema>;
  /**
   * Fetches the record's form values. Omit for a form that never reads. The
   * hook suspends on this, so the form is built from real values rather than
   * from a placeholder the fetch overwrites.
   */
  retrieve?: (params: RetrieveParams<Query>) => Promise<z.infer<Schema>>;
  /**
   * Projects the record's form values out of the domain client's cache.
   * A hit resolves the read synchronously, with no fetch and no suspension.
   */
  getCached?: (params: RetrieveParams<Query>) => z.infer<Schema> | undefined;
  update: (params: FormUpdateParams<Schema>) => Promise<void>;
  mountListeners?: (
    params: FormMountListenersParams<Query, Schema>,
  ) => destructor.Destructor | destructor.Destructor[];
  /**
   * Canonicalizes the caller's query before anything reads it: `retrieve`,
   * `mountListeners`, and `getCached` all receive the one normalized,
   * identity-stable object. Merge defaults here instead of at each callback,
   * where a per-call spread would mint a fresh object and miss the client's
   * query memos. Must preserve fields it does not set.
   */
  normalizeQuery?: <Q extends Query>(query: Q) => Q;
}

export type UseFormReturn<Schema extends z.ZodType<query.Data>> = Omit<
  Result<z.infer<Schema>>,
  "data"
> & {
  form: Form.UseReturn<Schema>;
  save: (opts?: query.FetchOptions) => void;
  /** Like save, but resolves true once the update has been persisted. */
  saveAsync: (opts?: query.FetchOptions) => Promise<boolean>;
};

export interface FormBeforeSaveParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
>
  extends Form.UseReturn<Schema>, FormClientParams<Query> {}

interface FormMountListenersParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
>
  extends Form.UseReturn<Schema>, Omit<FormClientParams<Query>, "query"> {
  query: Query;
  /**
   * Drops the pending autosave, aborts one already running, and stops further ones.
   * Call when the record no longer exists: a save queued before a delete would
   * otherwise write it back.
   */
  abandon: () => void;
}

export interface AfterSaveParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
> extends FormBeforeSaveParams<Query, Schema> {}

export interface BeforeValidateParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
> extends FormBeforeSaveParams<Query, Schema> {}

export interface UseFormParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
> extends Pick<Form.UseParams<Schema>, "sync" | "onHasTouched" | "mode"> {
  initialValues?: z.infer<Schema>;
  autoSave?: boolean;
  /**
   * How long to wait after a change before autosaving. Zero, the default, saves on
   * every change. Set it only for a form with a continuous input, such as a drag
   * handle or a color picker, where a single gesture emits a burst of changes.
   */
  autoSaveDebounce?: CrudeTimeSpan;
  /** The record to edit, or null for a form with nothing to read. */
  query: Query | null;
  beforeValidate?: (params: BeforeValidateParams<Query, Schema>) => boolean | void;
  beforeSave?: (params: FormBeforeSaveParams<Query, Schema>) => Promise<boolean>;
  afterSave?: (params: AfterSaveParams<Query, Schema>) => void;
}

export interface UseForm<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
> {
  (params: UseFormParams<Query, Schema>): UseFormReturn<Schema>;
}

const DEFAULT_SET_OPTIONS: Form.SetOptions = {
  markTouched: false,
  notifyOnChange: false,
};

export const createForm = <
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
>({
  name,
  schema,
  retrieve,
  getCached,
  mountListeners,
  update,
  initialValues: baseInitialValues,
  normalizeQuery,
}: CreateFormParams<Query, Schema>): UseForm<Query, Schema> => {
  const locals = new WeakMap<Client, LocalCache<z.infer<Schema>>>();
  return ({
    query,
    initialValues,
    autoSave = false,
    autoSaveDebounce = TimeSpan.ZERO,
    afterSave,
    beforeSave,
    beforeValidate,
    sync,
    onHasTouched,
    mode,
  }) => {
    const [result, setResult] = useState<Result<undefined>>(
      successResult(`retrieved ${name}`, undefined),
    );
    const client = Synnax.use();
    const listeners = useDestructors();
    const addStatus = Status.useAdder();
    const memoQuery = useMemoQuery(query, normalizeQuery);

    const cached = useMemo(
      () =>
        memoQuery == null || client == null
          ? undefined
          : getCached?.({ client, query: memoQuery }),
      [client, memoQuery],
    );

    const pending = usePendingFetch<Query, z.infer<Schema>>(memoQuery);
    // A replay resumes through the promise the suspended attempt holds. Reading the
    // answer from anywhere else would skip the `use` call React needs to find the
    // end of the recorded hook list, corrupting every hook below.
    let retrieved = pending.promise == null ? cached : undefined;
    if (retrieved == null && memoQuery != null && client != null && retrieve != null)
      retrieved = suspendOnFetch(
        { client, query: memoQuery },
        { name, retrieve, getCached, local: localFor(locals, client) },
        pending,
      );

    const abandonedRef = useRef(false);
    const abortRef = useRef<AbortController>(null);
    abortRef.current ??= new AbortController();
    const values = retrieved ?? initialValues ?? baseInitialValues;
    const form = Form.use<Schema>({
      schema,
      values,
      onChange: ({ path }) => {
        // Don't save if the path is empty to prevent infinite save loops.
        if (autoSave && path !== "" && !abandonedRef.current) debouncedSave();
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

    // Form state is built once, so a query pointing at a different record has
    // to replace it.
    const readQuery = useRef(memoQuery);
    const valuesRef = useRef(values);
    valuesRef.current = values;
    useLayoutEffect(() => {
      if (readQuery.current === memoQuery) return;
      readQuery.current = memoQuery;
      abandonedRef.current = false;
      abortRef.current = new AbortController();
      form.reset(valuesRef.current);
    }, [memoQuery, form]);

    const runSave = useCallback(
      async (opts: query.FetchOptions = {}): Promise<boolean> => {
        const { signal = abortRef.current?.signal } = opts;
        try {
          if (client == null) {
            setResult(nullClientResult<undefined>(`update ${name}`));
            return false;
          }
          const params = { client, query: memoQuery, ...form, set: noNotifySet };
          if (beforeValidate?.(params) === false) return false;
          if (!(await form.validateAsync())) return false;
          setResult(loadingResult(`updating ${name}`, undefined));
          if ((await beforeSave?.(params)) === false) {
            setResult(successResult(`updated ${name}`, undefined));
            return false;
          }
          if (signal?.aborted === true) {
            setResult(successResult(`updated ${name}`, undefined));
            return false;
          }
          const setStatus = (setter: state.SetArg<ResultStatus<never>>) =>
            setResult((p) => {
              const nextStatus = state.executeSetter(setter, p.status);
              return {
                ...p,
                status: nextStatus,
                variant: nextStatus.variant,
              } as Result<undefined>;
            });

          await update({ ...params, setStatus });
          setResult(successResult(`updated ${name}`, undefined));
          if (afterSave != null) afterSave(params);
          return true;
        } catch (error) {
          if (signal?.aborted === true) return false;
          const res = errorResult(`update ${name}`, error);
          addStatus(res.status);
          setResult(res);
          return false;
        }
      },
      [name, memoQuery, beforeSave, afterSave, beforeValidate],
    );

    // Saves of one record run one at a time. A caller that reads the form, awaits the
    // cluster, then writes back (a task deploy resolving its rack) leaves a window an
    // autosave fires in; concurrent writes reach the cluster in either order, so the
    // pre-write values can land last and undo the save.
    const pendingSave = useRef<Promise<unknown>>(null);
    const saveAsync = useCallback(
      async (opts: query.FetchOptions = {}): Promise<boolean> => {
        const save = (pendingSave.current ?? Promise.resolve()).then(
          async () => await runSave(opts),
        );
        pendingSave.current = save.catch(() => {});
        return await save;
      },
      [runSave],
    );
    const save = useCallback(
      (opts?: query.FetchOptions) => void saveAsync(opts),
      [saveAsync],
    );

    const saveRef = useSyncedRef(save);
    const debouncedSave = useDebouncedCallback(
      () => saveRef.current(),
      autoSaveDebounce,
      [],
    );
    useEffect(() => () => debouncedSave.flush(), [debouncedSave]);

    const abandon = useCallback(() => {
      abandonedRef.current = true;
      debouncedSave.cancel();
      abortRef.current?.abort();
    }, [debouncedSave]);

    useEffect(() => {
      if (memoQuery == null || client == null || mountListeners == null) return;
      listeners.cleanup();
      listeners.set(
        mountListeners({
          client,
          query: memoQuery,
          ...form,
          set: noNotifySet,
          abandon,
        }),
      );
      return () => listeners.cleanup();
    }, [client, memoQuery, form, noNotifySet, abandon]);

    return { form, save, saveAsync, ...result };
  };
};
