// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  isConnectionError,
  NotFoundError,
  query,
  type Synnax as Client,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  caseconv,
  compare,
  type destructor,
  sleep,
  type state,
  sync,
  TimeSpan,
} from "@synnaxlabs/x";
import { use, useEffect, useRef, useSyncExternalStore } from "react";
import { useSyncExternalStoreWithSelector } from "use-sync-external-store/with-selector";

import { type flux } from "@/flux/aether";
import { DeletedError, type Tombstone, tombstoneOf } from "@/flux/errors";
import {
  errorResult,
  loadingResult,
  noQueryResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import {
  ensureFetch,
  type EnsureFetchParams,
  fulfilled,
  invalidate,
  type LocalCache,
  localFor,
  promiseFor,
  type RetrieveParams,
  type Settled,
  subscribeReader,
  useMemoQuery,
  usePendingFetch,
} from "@/flux/suspend";
import { Synnax } from "@/synnax";

// Bound at module scope: hooks bind `query` to the caller's params object.
const { Deleted, isLive } = query;

export { type RetrieveParams };

/**
 * Binds a query definition onto its domain client's read surface: fetch =
 * `retrieve`, live updates = `onChange`, snapshot = `getCached`. Flux holds no
 * cache of its own; queries without `onChange` and `getCached` fetch on every
 * mount and never receive live updates.
 */
export interface CreateRetrieveParams<
  Query extends query.Params,
  Data extends query.Data,
> extends flux.Definition<Query, Data> {
  /**
   * Holds the previous answer whenever the next one compares equal, so readers
   * re-render only on changes the answer expresses. Defaults to element-wise
   * identity for list answers, which is what a `getCached` that composes entries
   * out of the domain client's tables needs. Override only for an answer whose
   * equality the default reads as a change.
   */
  equal?: (prev: Data, next: Data) => boolean;
  /**
   * Canonicalizes the caller's query before anything reads it: `retrieve`,
   * `onChange`, and `getCached` all receive the one normalized, identity-stable
   * object. Merge defaults here instead of at each callback, where a per-call
   * spread would mint a fresh object and miss the client's query memos. Must
   * preserve fields it does not set, so selector-only extensions survive.
   */
  normalizeQuery?: <Q extends Query>(query: Q) => Q;
  /**
   * Holds a not-found pending for a short wait instead of settling it, for documents a
   * reference can reach a reader ahead of: a panel tab minted in another window names
   * its view before that view's create broadcast lands. Every other reader shows
   * absence at once. Requires `onChange` and `getCached`.
   */
  awaitCreation?: boolean;
  /**
   * Asks again, with a widening wait, when a fetch fails because the Core could not be
   * reached. For reads a surface cannot render without: a settled failure holds until
   * something invalidates it, so one unlucky moment reads as a lasting one.
   */
  retryUnreachable?: boolean;
}

// The domain query cache interns its answers, so identity holds for anything
// it serves. This backstops a hand-written getCached that allocates instead,
// which would otherwise spin useSyncExternalStore forever.
const answersEqual = <Data>(prev: Data, next: Data): boolean =>
  Array.isArray(prev) && Array.isArray(next)
    ? compare.arraysEqual(prev, next)
    : Object.is(prev, next);

/**
 * Calls `notify` when the query's cached answer may have changed: on its change events,
 * and when a fetch answers, since a fetch can fill the cache with no change event.
 */
const subscribeToCache = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  client: Client,
  memoQuery: Query,
  onChange: CreateRetrieveParams<Query, Data>["onChange"],
  notify: () => void,
): destructor.Destructor => {
  const local = localFor(locals, client);
  const releaseReader = subscribeReader(local, query.hash(memoQuery), notify);
  const release = onChange?.({ client, query: memoQuery }, () => notify());
  return () => {
    releaseReader();
    release?.();
  };
};

interface UseCachedSnapshotParams<
  Query extends query.Params,
  Data extends query.Data,
> extends Pick<CreateRetrieveParams<Query, Data>, "onChange" | "getCached" | "equal"> {}

/**
 * Subscribes to the query's cached answer and reads it as a snapshot stable
 * enough for `useSyncExternalStore`: an answer equal to the one before it keeps
 * the earlier reference. Returns undefined when there is no client or query.
 */
const useCachedSnapshot = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  memoQuery: Query | null,
  client: Client | null,
  { onChange, getCached, equal = answersEqual }: UseCachedSnapshotParams<Query, Data>,
): query.Cached<Data> | undefined => {
  const held = useRef<{ query: Query; value: Data } | null>(null);
  return useSyncExternalStore(
    (notify) => {
      if (client == null || memoQuery == null) return NOOP_SUBSCRIBE();
      return subscribeToCache(locals, client, memoQuery, onChange, notify);
    },
    () => {
      if (client == null || memoQuery == null) return undefined;
      const next = getCached?.({ client, query: memoQuery });
      if (!isLive<Data>(next)) return next;
      const prev = held.current;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    },
  );
};

/**
 * Subscribes to the query's settled entry, which holds the answer to a retrieve the
 * domain cache never serves. Returns undefined when there is no client, no query, or no
 * entry.
 */
const useSettled = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  client: Client | null,
  memoQuery: Query | null,
): Settled<Data> | undefined => {
  const hash = memoQuery == null ? null : query.hash(memoQuery);
  return useSyncExternalStore(
    (notify: () => void) => {
      if (client == null || hash == null) return NOOP_SUBSCRIBE();
      return subscribeReader(localFor(locals, client), hash, notify);
    },
    () => {
      if (client == null || hash == null) return undefined;
      return localFor(locals, client).settled.get(hash);
    },
  );
};

/// A Suspense-shaped retrieve hook that returns the data directly. The hook
/// either returns the resolved value or suspends on the in-flight promise.
/// Concurrent reads of the same query share one fetch via the domain client's
/// query cache. Client-side listeners push new values into the cache, which
/// notifies subscribed consumers without re-suspending.
export interface Use<Query extends query.Params, Data extends state.State> {
  (query: Query): Data;
}

/// A Suspense-shaped hook that ensures a query has been retrieved into the
/// cache, then forgets about it: no cache subscription, no listener mounting.
/// Returns nothing; the data lives in the cache for children to read.
///
/// This is a narrow tool. Prefer letting children suspend independently inside
/// their own Suspense boundaries (the Twitter-shell pattern: parent renders
/// immediately, each child shows its own loading state). Reach for this hook
/// only when one of:
///
///   1. The parent must branch on the data before deciding which subtree to
///      render (e.g. resource-type dispatch).
///   2. Multiple children would otherwise issue the same query and each
///      suspend independently; pre-warming the cache here collapses them into
///      a single fetch.
///
/// Reach for `use` instead if the caller itself needs the data or
/// needs to re-render when it changes.
export interface UseEnsure<Query extends query.Params> {
  (query: Query): void;
}

/**
 * Returns a callback discarding a query's settled answer and any fetch still running,
 * so the next suspending read fetches again. A settled failure re-throws on every
 * render, so resetting an error boundary alone lands straight back on it.
 */
export interface UseInvalidate<Query extends query.Params> {
  (): (query: Query) => void;
}

/// A hook reading a deletion as a value instead of throwing it: the corpse
/// while the record is deleted, null while it is present, re-rendering when
/// that flips. Reach for it only where rendering absence is the caller's job,
/// so a restore heals the surface without an error boundary to reset. Every
/// other read should suspend and throw.
export interface UseTombstone<Query extends query.Params> {
  (query: Query): Tombstone | null;
}

/**
 * A warm-cache projected read: returns the selected slice of the query's cached
 * answer and re-renders only when that slice changes. Never suspends and never
 * fetches; a parent must have retrieved the query (see useEnsure).
 * @throws {NotFoundError} when nothing is cached for the query.
 * @throws {DeletedError} when the cached answer is a tombstone.
 * @throws {DisconnectedError} when no Core is connected.
 */
export interface UseSelect<Query extends query.Params, Selected> {
  (query: Query): Selected;
}

/**
 * A reactive read for callers that must handle loading and failure themselves, where
 * suspension is illegal or absence is a state to render rather than an error. Serves
 * the cached answer, subscribes for changes, and kicks a deduped background fetch on a
 * cold miss. Never suspends and never throws: the variant reports loading, success, and
 * failure, a null query or absent client reads as disabled, and an answer the domain
 * cache never holds is served once, from the fetch. Reach for `use` wherever the caller
 * may suspend.
 */
export interface UseResult<Query extends query.Params, Data extends state.State> {
  (query: Query | null): Result<Data>;
}

/**
 * Mints a {@link UseSelect} sharing the definition's cache wiring. ExtendedQuery adds
 * selector-only fields (a node key, a tab key) the select projection needs; the cache
 * layer ignores them when addressing the record.
 */
export interface CreateSelector<Query extends query.Params, Data extends query.Data> {
  <Selected, ExtendedQuery extends Query = Query>(
    select: (data: Data, query: ExtendedQuery) => Selected,
    equal?: (a: Selected, b: Selected) => boolean,
  ): UseSelect<ExtendedQuery, Selected>;
}

/**
 * Mints a {@link UseResult}-shaped hook that re-renders only when the selected slice
 * of the answer changes: {@link UseSelect}'s render gating with {@link UseResult}'s
 * contract — fetch on a cold miss, never suspend, never throw. For absence-tolerant
 * narrow readers; warm-cache readers under a parent retrieve use
 * {@link CreateSelector}.
 */
export interface CreateResultSelector<
  Query extends query.Params,
  Data extends query.Data,
> {
  <Selected extends state.State, ExtendedQuery extends Query = Query>(
    select: (data: Data, query: ExtendedQuery) => Selected,
    equal?: (a: Selected, b: Selected) => boolean,
  ): UseResult<ExtendedQuery, Selected>;
}

export interface CreateRetrieveReturn<
  Query extends query.Params,
  Data extends state.State,
> {
  use: Use<Query, Data>;
  useEnsure: UseEnsure<Query>;
  useResult: UseResult<Query, Data>;
  useInvalidate: UseInvalidate<Query>;
  useTombstone: UseTombstone<Query>;
  createSelector: CreateSelector<Query, Data>;
  createResultSelector: CreateResultSelector<Query, Data>;
}

/** The definition plus its per-client local cache, built once per createRetrieve and
 *  shared by every hook call, so renders mint no per-call params object. */
interface Context<
  Query extends query.Params,
  Data extends query.Data,
> extends CreateRetrieveParams<Query, Data> {
  locals: WeakMap<Client, LocalCache<Data>>;
}

const NOOP_SUBSCRIBE = () => () => {};

/**
 * How long a not-found suspending read stays pending before the not-found
 * becomes final. Only queries that set `awaitCreation` wait.
 */
const NOT_FOUND_WAIT = TimeSpan.seconds(5);

const waitForCreation = async <Query extends query.Params, Data extends query.Data>(
  params: RetrieveParams<Query>,
  {
    name,
    error,
    onChange,
    getCached,
  }: Required<Pick<CreateRetrieveParams<Query, Data>, "onChange" | "getCached">> & {
    name: string;
    error: Error;
  },
): Promise<Data> => {
  let disconnect: destructor.Destructor | undefined;
  const found = new Promise<query.Cached<Data>>((resolve) => {
    disconnect = onChange(params, (result) => {
      if (result !== undefined) resolve(result);
    });
    // The document may have landed, or been deleted, between the failed
    // fetch and the subscription mounting.
    const cached = getCached(params);
    if (cached !== undefined) resolve(cached);
  });
  let result: query.Cached<Data>;
  try {
    result = await sync.withTimeout(found, NOT_FOUND_WAIT, () => error);
  } finally {
    disconnect?.();
  }
  if (Deleted.matches<Data>(result))
    throw new DeletedError(`${caseconv.capitalize(name)} was deleted`, result.corpse);
  return result;
};

/** Attempts a fetch gets after the Core first proves unreachable. */
const UNREACHABLE_RETRIES = 3;

/** Wait before the first retry. Each later one waits twice as long. */
const RETRY_BASE = TimeSpan.milliseconds(250);

/**
 * Repeats a fetch the Core was unreachable for, failing once the attempts run out or
 * the Core answers with something other than a dead connection.
 */
const retryUnreachableFetch = <Query extends query.Params, Data extends query.Data>(
  params: RetrieveParams<Query>,
  { name, retrieve }: Pick<CreateRetrieveParams<Query, Data>, "name" | "retrieve">,
): Promise<Data> => {
  const attempt = async (n: number): Promise<Data> => {
    await sleep.sleep(RETRY_BASE.milliseconds * 2 ** (n - 1));
    try {
      return await retrieve(params);
    } catch (cause) {
      if (n < UNREACHABLE_RETRIES && isConnectionError(cause))
        return await attempt(n + 1);
      throw new Error(`Failed to retrieve ${name}`, { cause });
    }
  };
  return attempt(1);
};

interface FetchParamsSource<
  Query extends query.Params,
  Data extends query.Data,
> extends Pick<
  CreateRetrieveParams<Query, Data>,
  "name" | "retrieve" | "onChange" | "getCached" | "awaitCreation" | "retryUnreachable"
> {
  local: LocalCache<Data>;
}

const fetchParamsFor = <Query extends query.Params, Data extends query.Data>({
  name,
  retrieve,
  onChange,
  getCached,
  awaitCreation = false,
  retryUnreachable = false,
  local,
}: FetchParamsSource<Query, Data>): EnsureFetchParams<Query, Data> => ({
  name,
  retrieve,
  getCached,
  local,
  onFetchError: (params, { cause, error }) => {
    // A not-found on a query that awaits creation stays pending: the reference may have
    // outrun its document's create broadcast, which the subscription will deliver.
    if (
      awaitCreation &&
      onChange != null &&
      getCached != null &&
      NotFoundError.matches(cause)
    )
      return waitForCreation(params, { name, error, onChange, getCached });
    if (retryUnreachable && isConnectionError(cause))
      return retryUnreachableFetch(params, { name, retrieve });
    // Every other failure settles.
    return null;
  },
});

const useSuspended = <Query extends query.Params, Data extends query.Data>(
  {
    locals,
    name,
    retrieve,
    onChange,
    getCached,
    equal,
    normalizeQuery,
    awaitCreation,
    retryUnreachable,
  }: Context<Query, Data>,
  query: Query,
): Data => {
  const memoQuery = useMemoQuery(query, normalizeQuery);
  const client = Synnax.use();

  // Every hook runs before the disconnected throw. Gating them on the client
  // would change this hook's count as the connection comes and goes, which
  // corrupts the caller's hook order.
  const cached = useCachedSnapshot<Query, Data>(locals, memoQuery, client, {
    onChange,
    getCached,
    equal,
  });
  const pending = usePendingFetch<Query, Data>(memoQuery);

  if (client == null)
    throw new DisconnectedError(`Cannot retrieve ${name}: no Core connected.`);

  const local = localFor(locals, client);
  const params = { client, query: memoQuery };

  // A replay resumes through the promise the suspended attempt holds. Serving it the
  // answer the fetch just put in the cache would skip the `use` call React needs to
  // find the end of the recorded hook list.
  if (pending.promise == null && cached !== undefined) {
    if (Deleted.matches<Data>(cached))
      throw new DeletedError(`${caseconv.capitalize(name)} was deleted`, cached.corpse);
    return use(fulfilled(cached));
  }
  return use(
    promiseFor(
      params,
      fetchParamsFor({
        name,
        retrieve,
        onChange,
        getCached,
        awaitCreation,
        retryUnreachable,
        local,
      }),
      pending,
    ),
  );
};

const useEnsure = <Query extends query.Params, Data extends query.Data>(
  {
    locals,
    name,
    retrieve,
    onChange,
    getCached,
    normalizeQuery,
    awaitCreation,
    retryUnreachable,
  }: Context<Query, Data>,
  query: Query,
): void => {
  const memoQuery = useMemoQuery(query, normalizeQuery);
  const client = Synnax.use();
  const pending = usePendingFetch<Query, Data>(memoQuery);

  if (client == null)
    throw new DisconnectedError(`Cannot retrieve ${name}: no Core connected.`);

  const local = localFor(locals, client);
  const params = { client, query: memoQuery };

  if (pending.promise == null) {
    const cached = getCached?.(params);
    if (cached !== undefined) {
      if (Deleted.matches<Data>(cached))
        throw new DeletedError(
          `${caseconv.capitalize(name)} was deleted`,
          cached.corpse,
        );
      use(fulfilled(cached));
      return;
    }
  }
  use(
    promiseFor(
      params,
      fetchParamsFor({
        name,
        retrieve,
        onChange,
        getCached,
        awaitCreation,
        retryUnreachable,
        local,
      }),
      pending,
    ),
  );
};

const useInvalidate = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  normalizeQuery?: <Q extends Query>(query: Q) => Q,
): ((q: Query) => void) => {
  const client = Synnax.use();
  return (q: Query) => {
    if (client == null) return;
    const normalized = normalizeQuery?.(q) ?? q;
    invalidate(localFor(locals, client), query.hash(normalized));
  };
};

const useTombstone = <Query extends query.Params, Data extends query.Data>(
  { locals, onChange, getCached, equal, normalizeQuery }: Context<Query, Data>,
  query: Query,
): Tombstone | null => {
  const memoQuery = useMemoQuery(query, normalizeQuery);
  const client = Synnax.use();
  const cached = useCachedSnapshot<Query, Data>(locals, memoQuery, client, {
    onChange,
    getCached,
    equal,
  });
  return Deleted.matches<Data>(cached) ? tombstoneOf(cached.corpse) : null;
};

const createSelector = <
  Query extends query.Params,
  Data extends query.Data,
  Selected,
  ExtendedQuery extends Query = Query,
>(
  {
    name,
    onChange,
    getCached,
    equal = answersEqual,
    normalizeQuery,
  }: CreateRetrieveParams<Query, Data> &
    Required<Pick<CreateRetrieveParams<Query, Data>, "getCached">>,
  select: (data: Data, query: ExtendedQuery) => Selected,
  selectedEqual?: (a: Selected, b: Selected) => boolean,
): UseSelect<ExtendedQuery, Selected> => {
  const useSelect = (q: ExtendedQuery): Selected => {
    const memoQuery = useMemoQuery<ExtendedQuery>(q, normalizeQuery);
    const client = Synnax.use();
    const held = useRef<{ query: Query; value: Data } | null>(null);
    const computed = useRef<{ raw: Data; query: Query; out: Selected } | null>(null);
    const subscribe = (notify: () => void) => {
      if (onChange == null || client == null) return NOOP_SUBSCRIBE();
      return onChange({ client, query: memoQuery }, () => notify());
    };
    const getSnapshot = (): query.Cached<Data> | undefined => {
      if (client == null) return undefined;
      const next = getCached({ client, query: memoQuery });
      const prev = held.current;
      // An invalidated entry repairs under its own subscription; serve the held
      // answer across the gap.
      if (next === undefined)
        return prev != null && prev.query === memoQuery ? prev.value : undefined;
      if (!isLive<Data>(next)) return next;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    };
    const selector = (raw: query.Cached<Data> | undefined): Selected => {
      if (client == null)
        throw new DisconnectedError(`Cannot select ${name}: no Core connected.`);
      if (raw === undefined)
        throw new NotFoundError(
          `Cannot select ${name}: nothing cached. A parent must retrieve it first.`,
        );
      if (Deleted.matches<Data>(raw))
        throw new DeletedError(`${caseconv.capitalize(name)} was deleted`, raw.corpse);
      const cached = computed.current;
      if (cached != null && cached.raw === raw && cached.query === memoQuery)
        return cached.out;
      const out = select(raw, memoQuery);
      computed.current = { raw, query: memoQuery, out };
      return out;
    };
    return useSyncExternalStoreWithSelector(
      subscribe,
      getSnapshot,
      undefined,
      selector,
      selectedEqual,
    );
  };
  return useSelect;
};

/** What a result selector's selection resolved to, compared slice-wise so an
 *  answer change outside the slice never re-renders the consumer. */
type Slice<Data, Selected> =
  | { kind: "live"; selected: Selected }
  | { kind: "deleted"; corpse: Data }
  | { kind: "none" };

const NONE_SLICE = { kind: "none" } as const;

const createResultSelector = <
  Query extends query.Params,
  Data extends query.Data,
  Selected extends state.State,
  ExtendedQuery extends Query = Query,
>(
  context: Context<Query, Data>,
  select: (data: Data, query: ExtendedQuery) => Selected,
  selectedEqual: (a: Selected, b: Selected) => boolean = Object.is,
): UseResult<ExtendedQuery, Selected> => {
  const {
    locals,
    name,
    retrieve,
    onChange,
    getCached,
    equal = answersEqual,
    normalizeQuery,
    awaitCreation,
    retryUnreachable,
  } = context;
  const sliceEqual = (a: Slice<Data, Selected>, b: Slice<Data, Selected>): boolean => {
    if (a.kind !== b.kind) return false;
    if (a.kind === "live" && b.kind === "live")
      return selectedEqual(a.selected, b.selected);
    if (a.kind === "deleted" && b.kind === "deleted") return a.corpse === b.corpse;
    return true;
  };
  const useResultSelect = (q: ExtendedQuery | null): Result<Selected> => {
    const memoQuery = useMemoQuery<ExtendedQuery>(q, normalizeQuery);
    const client = Synnax.use();
    const held = useRef<{ query: Query; value: Data } | null>(null);
    const subscribe = (notify: () => void) => {
      if (client == null || memoQuery == null) return NOOP_SUBSCRIBE();
      return subscribeToCache(locals, client, memoQuery, onChange, notify);
    };
    const getSnapshot = (): query.Cached<Data> | undefined => {
      if (client == null || memoQuery == null) return undefined;
      const next = getCached?.({ client, query: memoQuery });
      if (!isLive<Data>(next)) return next;
      const prev = held.current;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    };
    const selector = (raw: query.Cached<Data> | undefined): Slice<Data, Selected> => {
      if (memoQuery == null || raw === undefined) return NONE_SLICE;
      if (Deleted.matches<Data>(raw)) return { kind: "deleted", corpse: raw.corpse };
      return { kind: "live", selected: select(raw, memoQuery) };
    };
    const slice = useSyncExternalStoreWithSelector(
      subscribe,
      getSnapshot,
      undefined,
      selector,
      sliceEqual,
    );
    const settled = useSettled<ExtendedQuery, Data>(locals, client, memoQuery);
    // A render React discards must not fetch, and nothing dedupes one once the
    // in-flight entry clears, so the cold path starts its fetch after commit.
    // A settled failure does not block the fetch: each new mount gets one fresh
    // attempt (deps hold within a mount, so a failure cannot loop).
    useEffect(() => {
      if (client == null || memoQuery == null || slice.kind !== "none") return;
      const local = localFor(locals, client);
      const params = { client, query: memoQuery };
      const prior = local.settled.get(query.hash(memoQuery));
      if (prior != null && "data" in prior) return;
      // A failure settles, and the settled subscription reports it.
      ensureFetch(
        params,
        fetchParamsFor({
          name,
          retrieve,
          onChange,
          getCached,
          awaitCreation,
          retryUnreachable,
          local,
        }),
      ).catch(() => {});
    }, [client, memoQuery, slice]);
    if (client == null) return nullClientResult<Selected>(`retrieve ${name}`);
    if (memoQuery == null) return noQueryResult<Selected>(`retrieve ${name}`);
    if (slice.kind === "live")
      return successResult(`retrieved ${name}`, slice.selected);
    if (slice.kind === "deleted")
      return errorResult(
        `retrieve ${name}`,
        new DeletedError(`${caseconv.capitalize(name)} was deleted`, slice.corpse),
      );
    if (settled != null)
      return "data" in settled
        ? successResult(`retrieved ${name}`, select(settled.data, memoQuery))
        : errorResult(`retrieve ${name}`, settled.error);
    return loadingResult<Selected>(`retrieving ${name}`);
  };
  return useResultSelect;
};

export const createRetrieve = <Query extends query.Params, Data extends query.Data>(
  createParams: CreateRetrieveParams<Query, Data>,
): CreateRetrieveReturn<Query, Data> => {
  const context: Context<Query, Data> = { ...createParams, locals: new WeakMap() };
  const { locals } = context;
  return {
    use: (query: Query) => useSuspended(context, query),
    useEnsure: (query: Query) => useEnsure(context, query),
    useResult: createResultSelector<Query, Data, Data>(context, (data) => data),
    useInvalidate: () =>
      useInvalidate<Query, Data>(locals, createParams.normalizeQuery),
    useTombstone: (query: Query) => useTombstone(context, query),
    createSelector: <Selected, ExtendedQuery extends Query = Query>(
      select: (data: Data, query: ExtendedQuery) => Selected,
      equal?: (a: Selected, b: Selected) => boolean,
    ) => {
      const { getCached } = createParams;
      if (getCached == null)
        throw new UnexpectedError(
          `Cannot create a selector for ${createParams.name}: no getCached defined.`,
        );
      return createSelector({ ...createParams, getCached }, select, equal);
    },
    createResultSelector: <
      Selected extends state.State,
      ExtendedQuery extends Query = Query,
    >(
      select: (data: Data, query: ExtendedQuery) => Selected,
      equal?: (a: Selected, b: Selected) => boolean,
    ) => {
      if (createParams.getCached == null)
        throw new UnexpectedError(
          `Cannot create a result selector for ${createParams.name}: no getCached defined.`,
        );
      return createResultSelector(context, select, equal);
    },
  };
};
