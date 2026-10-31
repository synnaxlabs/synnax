// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, type Synnax as Client } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import {
  useDebouncedCallback,
  useDestructors,
  useSyncedRef,
} from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import {
  type CrudeTimeSpan,
  deep,
  type destructor,
  type record,
  state,
  TimeSpan,
} from "@synnaxlabs/x";
import {
  type Dispatch,
  type SetStateAction,
  use,
  useEffect,
  useId,
  useLayoutEffect,
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
  fulfilled,
  type LocalCache,
  localFor,
  promiseFor,
  type RetrieveParams,
  useMemoQuery,
  usePendingFetch,
} from "@/flux/suspend";
import { type UpdateParams } from "@/flux/update";
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
   * How long to wait after a change before autosaving. Raise it for a form with a
   * continuous input, such as a drag handle or a color picker, where one gesture
   * emits a burst of changes. Zero saves on every change.
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

const DEFAULT_AUTO_SAVE_DEBOUNCE = TimeSpan.milliseconds(200);

const covers = (pending: Set<string>, path: string): boolean =>
  [...pending].some((p) => deep.pathsMatch(path, p));

const ABSENT = Symbol("absent");

/**
 * @returns the live value at the path, null and undefined included, or ABSENT when the
 * value that holds it is gone.
 */
const readLive = (live: unknown, path: string): unknown => {
  if (deep.has(live, path)) return deep.get(live, path);
  const split = path.lastIndexOf(deep.SEPARATOR);
  const parentPath = split === -1 ? "" : path.slice(0, split);
  const parent = deep.get<record.Unknown, unknown>(live, parentPath, {
    optional: true,
  });
  return parent == null ? ABSENT : deep.defaultGetter(parent, path.slice(split + 1));
};

/** @returns the value to write at the path, with live values kept at pending paths. */
const keepPending = <V>(
  live: unknown,
  path: string,
  value: V,
  pending: Set<string>,
): V => {
  let kept = value;
  for (const p of pending) {
    if (!deep.pathsMatch(p, path)) continue;
    const current = readLive(live, p);
    if (current === ABSENT) continue;
    if (kept === value) kept = deep.copy(value);
    deep.set(kept, path === "" ? p : p.slice(path.length + 1), current);
  }
  return kept;
};

/** A save waiting on the one before it, and the signal its callers issued it under. */
interface QueuedSave {
  promise: Promise<boolean>;
  signal?: AbortSignal;
}

interface PersistParams<
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
>
  extends
    Pick<UseFormParams<Query, Schema>, "beforeValidate" | "beforeSave" | "afterSave">,
    Pick<CreateFormParams<Query, Schema>, "name" | "update"> {
  params: FormBeforeSaveParams<Query, Schema>;
  setResult: Dispatch<SetStateAction<Result<undefined>>>;
  addStatus: Status.Adder;
  statusKey: string;
  signal?: AbortSignal;
}

/** Validates and writes the form, reporting progress through `setResult`. */
const persist = async <
  Query extends query.Params,
  Schema extends z.ZodType<query.Data>,
>({
  name,
  params,
  update,
  beforeValidate,
  beforeSave,
  afterSave,
  setResult,
  addStatus,
  statusKey,
  signal,
}: PersistParams<Query, Schema>): Promise<boolean> => {
  try {
    if (beforeValidate?.(params) === false) return false;
    if (!(await params.validateAsync())) return false;
    setResult(loadingResult(`saving ${name}`, undefined));
    if ((await beforeSave?.(params)) === false) {
      setResult(successResult(`saved ${name}`, undefined));
      return false;
    }
    if (signal?.aborted === true) {
      setResult(successResult(`saved ${name}`, undefined));
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
    setResult(successResult(`saved ${name}`, undefined));
    if (afterSave != null) afterSave(params);
    return true;
  } catch (error) {
    if (signal?.aborted === true) return false;
    const res = errorResult(`save ${name}`, error);
    addStatus({ ...res.status, key: statusKey });
    setResult(res);
    return false;
  }
};

const withKey = (result: Result<undefined>, key: string): Result<undefined> =>
  ({ ...result, status: { ...result.status, key } }) as Result<undefined>;

const readCached = <Query extends query.Params, Data>(
  getCached: ((params: RetrieveParams<Query>) => Data | undefined) | undefined,
  params: RetrieveParams<Query>,
): Promise<Data> | undefined => {
  const cached = getCached?.(params);
  return cached === undefined ? undefined : fulfilled(cached);
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
  const useForm = ({
    query,
    initialValues,
    autoSave = false,
    autoSaveDebounce = DEFAULT_AUTO_SAVE_DEBOUNCE,
    afterSave,
    beforeSave,
    beforeValidate,
    sync,
    onHasTouched,
    mode,
  }: UseFormParams<Query, Schema>): UseFormReturn<Schema> => {
    // Every status of the form has one key, so a view that shows the status in place
    // can place the key before a save fails.
    const statusKey = useId();
    const [result, setKeyedResult] = useState<Result<undefined>>(() =>
      withKey(successResult(`retrieved ${name}`, undefined), statusKey),
    );
    const setResult: Dispatch<SetStateAction<Result<undefined>>> = (setter) =>
      setKeyedResult((prev) => withKey(state.executeSetter(setter, prev), statusKey));
    const client = Synnax.use();
    const listeners = useDestructors();
    const addStatus = Status.useAdder();
    const memoQuery = useMemoQuery(query, normalizeQuery);

    // One read per query: a synced form resets when its values change identity. One
    // call keeps the read and its promise in one memo block.
    const cached =
      memoQuery == null || client == null
        ? undefined
        : readCached(getCached, { client, query: memoQuery });

    const pending = usePendingFetch<Query, z.infer<Schema>>(memoQuery);
    // A replay resumes through the promise the suspended attempt holds. Reading the
    // answer from anywhere else would skip the `use` call React needs to find the
    // end of the recorded hook list, corrupting every hook below.
    let retrieved: z.infer<Schema> | undefined =
      pending.promise == null && cached != null ? use(cached) : undefined;
    if (retrieved == null && memoQuery != null && client != null && retrieve != null)
      retrieved = use(
        promiseFor(
          { client, query: memoQuery },
          { name, retrieve, getCached, local: localFor(locals, client) },
          pending,
        ),
      );

    const abandonedRef = useRef(false);
    const [initialAbort] = useState(() => new AbortController());
    const abortRef = useRef(initialAbort);
    // Paths edited since the running save read the form, and the paths that save
    // carries. The Core holds neither yet, so a listener or a save writing back what it
    // sent would revert them, and the next save would store the reverted value.
    const unsaved = useRef(new Set<string>());
    const sending = useRef(new Set<string>());
    const values = retrieved ?? initialValues ?? baseInitialValues;
    const form = Form.use<Schema>({
      schema,
      values,
      onChange: ({ path }) => {
        // A reset reports an empty path. Saving on it would loop.
        if (path === "") return;
        unsaved.current.add(path);
        if (autoSave && !abandonedRef.current) debouncedSave();
      },
      sync,
      onHasTouched,
      mode,
    });
    const noNotifySet = (path: string, value: unknown, options?: Form.SetOptions) =>
      form.set(path, value, { ...options, ...DEFAULT_SET_OPTIONS });
    const pendingPaths = () => new Set([...unsaved.current, ...sending.current]);
    const listenerSet = (path: string, value: unknown, options?: Form.SetOptions) => {
      const pending = pendingPaths();
      if (covers(pending, path)) return;
      noNotifySet(path, keepPending(form.value(), path, value, pending), options);
    };
    const resetKeeping = (pending: () => Set<string>) => (next?: z.infer<Schema>) =>
      form.reset(next == null ? next : keepPending(form.value(), "", next, pending()));

    // Form state is built once, so a query pointing at a different record has
    // to replace it.
    const readQuery = useRef(memoQuery);
    const valuesRef = useSyncedRef(values);
    useLayoutEffect(() => {
      if (readQuery.current === memoQuery) return;
      readQuery.current = memoQuery;
      abandonedRef.current = false;
      abortRef.current = new AbortController();
      unsaved.current = new Set();
      sending.current = new Set();
      form.reset(valuesRef.current);
    }, [memoQuery, form]);

    const runSave = async (opts: query.FetchOptions = {}): Promise<boolean> => {
      const signal = opts.signal ?? abortRef.current.signal;
      if (client == null) {
        setResult(nullClientResult<undefined>(`save ${name}`));
        return false;
      }
      const issuedFor = readQuery.current;
      const sent = unsaved.current;
      sending.current = sent;
      unsaved.current = new Set();
      const saved = await persist({
        name,
        params: {
          client,
          query: memoQuery,
          ...form,
          set: noNotifySet,
          reset: resetKeeping(() => unsaved.current),
        },
        update,
        beforeValidate,
        beforeSave,
        afterSave,
        setResult,
        addStatus,
        statusKey,
        signal,
      });
      // The form re-pointed at another record while this save ran.
      if (readQuery.current !== issuedFor) return saved;
      if (!saved) sent.forEach((path) => unsaved.current.add(path));
      sending.current = new Set();
      return saved;
    };

    // Saves of one record run one at a time. A caller that reads the form, awaits the
    // Core, then writes back (a task deploy resolving its rack) leaves a window an
    // autosave fires in; concurrent writes reach the Core in either order, so the
    // pre-write values can land last and undo the save.
    const pendingSave = useRef<Promise<unknown>>(null);
    // A save that has not started reads the live form when it does, so it already
    // carries what a later caller would write. Later callers join it instead of
    // queueing a duplicate. Only callers under one signal join, so an abort still
    // reaches every save it was issued for.
    const queuedSave = useRef<QueuedSave>(null);
    const saveAsync = async (opts: query.FetchOptions = {}): Promise<boolean> => {
      const signal = opts.signal ?? abortRef.current.signal;
      const queued = queuedSave.current;
      if (queued != null && queued.signal === signal) return await queued.promise;
      const issuedFor = memoQuery;
      const save = (pendingSave.current ?? Promise.resolve()).then(async () => {
        if (queuedSave.current?.promise === save) queuedSave.current = null;
        // A queued save reads the live form, which now holds another record.
        return readQuery.current === issuedFor ? await runSave(opts) : false;
      });
      queuedSave.current = { promise: save, signal };
      pendingSave.current = save.catch(() => {});
      return await save;
    };
    const save = (opts?: query.FetchOptions) => void saveAsync(opts);

    const saveRef = useSyncedRef(save);
    // An explicit save that ran before the timer fired already sent every edit.
    const debouncedSave = useDebouncedCallback(
      () => {
        if (unsaved.current.size > 0) saveRef.current();
      },
      autoSaveDebounce,
      [],
    );
    useEffect(() => () => debouncedSave.flush(), [debouncedSave]);

    const abandon = () => {
      abandonedRef.current = true;
      debouncedSave.cancel();
      abortRef.current.abort();
    };

    useEffect(() => {
      if (memoQuery == null || client == null || mountListeners == null) return;
      listeners.cleanup();
      listeners.set(
        mountListeners({
          client,
          query: memoQuery,
          ...form,
          set: listenerSet,
          reset: resetKeeping(pendingPaths),
          abandon,
        }),
      );
      return () => listeners.cleanup();
    }, [client, memoQuery, form, listenerSet, resetKeeping, pendingPaths, abandon]);

    return { form, save, saveAsync, ...result };
  };
  return useForm;
};
