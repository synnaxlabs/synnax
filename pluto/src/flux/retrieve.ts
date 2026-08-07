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
  NotFoundError,
  query,
  type Synnax as Client,
  UnexpectedError,
} from "@synnaxlabs/client";
import { compare, type destructor, type state, TimeSpan } from "@synnaxlabs/x";
import { useCallback, useMemo, useReducer, useRef, useSyncExternalStore } from "react";
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
  type LocalCache,
  localFor,
  type RetrieveParams,
  suspendOnFetch,
  usePendingFetch,
} from "@/flux/suspend";
import { useMemoDeepEqual } from "@/memo";
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
   * Builds the answer synchronously from records already cached under other
   * queries. Consulted only when `getCached` misses, so suspending reads
   * resolve without a fetch. Returns undefined to fall through to `retrieve`.
   */
  deriveCached?: (params: RetrieveParams<Query>) => Data | undefined;
  /**
   * Holds the previous answer whenever the next one compares equal, so readers
   * re-render only on changes the answer expresses. Defaults to element-wise
   * identity for list answers, which is what a `getCached` that composes rows
   * out of the domain client's tables needs. Override only for an answer whose
   * equality the default reads as a change.
   */
  equal?: (prev: Data, next: Data) => boolean;
}

// A getCached that composes a list answer allocates a fresh array per call,
// which spins useSyncExternalStore forever. Single-record answers come back as
// the domain client's own row, so identity already holds for them.
const answersEqual = <Data>(prev: Data, next: Data): boolean =>
  Array.isArray(prev) && Array.isArray(next)
    ? compare.arraysEqual(prev, next)
    : Object.is(prev, next);

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
  memoQuery: Query | null,
  client: Client | null,
  { onChange, getCached, equal = answersEqual }: UseCachedSnapshotParams<Query, Data>,
): query.Cached<Data> | undefined => {
  const held = useRef<{ query: Query; value: Data } | null>(null);
  return useSyncExternalStore(
    useCallback(
      (notify) => {
        if (onChange == null || client == null || memoQuery == null)
          return NOOP_SUBSCRIBE();
        return onChange({ client, query: memoQuery }, () => notify());
      },
      [client, memoQuery],
    ),
    useCallback(() => {
      if (client == null || memoQuery == null) return undefined;
      const next = getCached?.({ client, query: memoQuery });
      if (!isLive<Data>(next)) return next;
      const prev = held.current;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    }, [client, memoQuery]),
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
 * Returns a callback discarding a query's settled answer so the next suspending
 * read fetches again. A settled failure re-throws on every render, so resetting
 * an error boundary alone lands straight back on it.
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
}

interface UseSuspendedParams<Query extends query.Params, Data extends query.Data> {
  query: Query;
  locals: WeakMap<Client, LocalCache<Data>>;
}

const NOOP_SUBSCRIBE = () => () => {};

/**
 * How long a not-found suspending read stays pending before the not-found
 * becomes final. Covers a reference arriving ahead of its document's create
 * broadcast (e.g. a panel tab minted in another window).
 */
const NOT_FOUND_WAIT = TimeSpan.seconds(5);

const awaitCreation = <Query extends query.Params, Data extends query.Data>(
  params: RetrieveParams<Query>,
  {
    name,
    error,
    hash,
    onChange,
    getCached,
    local,
  }: Required<Pick<CreateRetrieveParams<Query, Data>, "onChange" | "getCached">> & {
    name: string;
    error: Error;
    hash: string;
    local: LocalCache<Data>;
  },
): Promise<Data> =>
  new Promise<Data>((resolve, reject) => {
    const epoch = local.epoch;
    let settled = false;
    let disconnect: destructor.Destructor = () => {};
    const finish = () => {
      settled = true;
      clearTimeout(timer);
      disconnect();
      local.inFlight.delete(hash);
    };
    const timer = setTimeout(() => {
      finish();
      // A reconnect landed during the wait, so the not-found may no longer hold.
      if (local.epoch === epoch) local.settled.set(hash, { error });
      reject(error);
    }, NOT_FOUND_WAIT.milliseconds);
    disconnect = onChange(params, (result) => {
      if (result === undefined) return;
      finish();
      if (Deleted.matches<Data>(result))
        reject(new DeletedError(`${name} was deleted`, result.corpse));
      else resolve(result);
    });
    // An already-answered query delivers during onChange itself, before the
    // destructor exists to be called; tear it down now.
    if (settled) disconnect();
    // The document may have landed, or been deleted, between the failed
    // fetch and the subscription mounting.
    const cached = getCached(params);
    if (cached !== undefined) {
      finish();
      if (Deleted.matches<Data>(cached))
        reject(new DeletedError(`${name} was deleted`, cached.corpse));
      else resolve(cached);
    }
  });

interface FetchParamsSource<
  Query extends query.Params,
  Data extends query.Data,
> extends Pick<
  CreateRetrieveParams<Query, Data>,
  "name" | "retrieve" | "onChange" | "getCached"
> {
  local: LocalCache<Data>;
}

const fetchParamsFor = <Query extends query.Params, Data extends query.Data>({
  name,
  retrieve,
  onChange,
  getCached,
  local,
}: FetchParamsSource<Query, Data>): EnsureFetchParams<Query, Data> => ({
  name,
  retrieve,
  getCached,
  local,
  // A domain-cached not-found stays pending: the reference may have outrun its
  // document's create broadcast, which the subscription will deliver.
  // Everything else settles.
  onFetchError: (params, { cause, error, hash }) =>
    onChange != null && getCached != null && NotFoundError.matches(cause)
      ? awaitCreation(params, { name, error, hash, onChange, getCached, local })
      : null,
});

const useSuspended = <Query extends query.Params, Data extends query.Data>({
  query,
  locals,
  name,
  retrieve,
  onChange,
  getCached,
  deriveCached,
  equal,
}: UseSuspendedParams<Query, Data> & CreateRetrieveParams<Query, Data>): Data => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();

  // Every hook runs before the disconnected throw. Gating them on the client
  // would change this hook's count as the connection comes and goes, which
  // corrupts the caller's hook order.
  const cached = useCachedSnapshot<Query, Data>(memoQuery, client, {
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
  if (pending.promise == null) {
    if (cached !== undefined) {
      if (Deleted.matches<Data>(cached))
        throw new DeletedError(`${name} was deleted`, cached.corpse);
      return cached;
    }
    const derived = deriveCached?.(params);
    if (derived != null) return derived;
  }
  return suspendOnFetch(
    params,
    fetchParamsFor({ name, retrieve, onChange, getCached, local }),
    pending,
  );
};

const useResultValue = <Query extends query.Params, Data extends query.Data>({
  query: q,
  locals,
  name,
  retrieve,
  onChange,
  getCached,
  deriveCached,
  equal,
}: Omit<UseSuspendedParams<Query, Data>, "query"> &
  CreateRetrieveParams<Query, Data> & { query: Query | null }): Result<Data> => {
  const memoQuery = useMemoDeepEqual(q);
  // A retrieve whose answer never reaches the domain cache is served from the local
  // settled entry, which no subscription announces, so settling re-renders by hand.
  const [, bump] = useReducer((x: number) => x + 1, 0);
  const client = Synnax.use();

  const cached = useCachedSnapshot<Query, Data>(memoQuery, client, {
    onChange,
    getCached,
    equal,
  });

  if (client == null) return nullClientResult<Data>(`retrieve ${name}`);
  if (memoQuery == null) return noQueryResult<Data>(`retrieve ${name}`);
  if (cached !== undefined) {
    if (!Deleted.matches<Data>(cached))
      return successResult(`retrieved ${name}`, cached);
    return errorResult(
      `retrieve ${name}`,
      new DeletedError(`${name} was deleted`, cached.corpse),
    );
  }

  const local = localFor(locals, client);
  const params = { client, query: memoQuery };
  const derived = deriveCached?.(params);
  if (derived != null) return successResult(`retrieved ${name}`, derived);
  const settled = local.settled.get(query.hash(memoQuery));
  if (settled != null)
    return "data" in settled
      ? successResult(`retrieved ${name}`, settled.data)
      : errorResult(`retrieve ${name}`, settled.error);
  ensureFetch(
    params,
    fetchParamsFor({ name, retrieve, onChange, getCached, local }),
  ).then(bump, bump);
  return loadingResult<Data>(`retrieving ${name}`);
};

const useEnsure = <Query extends query.Params, Data extends query.Data>({
  query,
  locals,
  name,
  retrieve,
  onChange,
  getCached,
  deriveCached,
}: UseSuspendedParams<Query, Data> & CreateRetrieveParams<Query, Data>): void => {
  const memoQuery = useMemoDeepEqual(query);
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
        throw new DeletedError(`${name} was deleted`, cached.corpse);
      return;
    }
    if (deriveCached?.(params) != null) return;
  }
  suspendOnFetch(
    params,
    fetchParamsFor({ name, retrieve, onChange, getCached, local }),
    pending,
  );
};

const useInvalidate = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
): ((q: Query) => void) => {
  const client = Synnax.use();
  return useCallback(
    (q: Query) => {
      if (client == null) return;
      localFor(locals, client).settled.delete(query.hash(q));
    },
    [client, locals],
  );
};

interface UseTombstoneParams<Query extends query.Params> {
  query: Query;
}

const useTombstone = <Query extends query.Params, Data extends query.Data>({
  query,
  onChange,
  getCached,
  equal,
}: UseTombstoneParams<Query> & CreateRetrieveParams<Query, Data>): Tombstone | null => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();
  const cached = useCachedSnapshot<Query, Data>(memoQuery, client, {
    onChange,
    getCached,
    equal,
  });
  return useMemo(
    () => (Deleted.matches<Data>(cached) ? tombstoneOf(cached.corpse) : null),
    [cached],
  );
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
  }: CreateRetrieveParams<Query, Data> &
    Required<Pick<CreateRetrieveParams<Query, Data>, "getCached">>,
  select: (data: Data, query: ExtendedQuery) => Selected,
  selectedEqual?: (a: Selected, b: Selected) => boolean,
): UseSelect<ExtendedQuery, Selected> => {
  const useSelect = (q: ExtendedQuery): Selected => {
    const memoQuery = useMemoDeepEqual(q);
    const client = Synnax.use();
    const held = useRef<{ query: Query; value: Data } | null>(null);
    const computed = useRef<{ raw: Data; query: Query; out: Selected } | null>(null);
    const subscribeToCache = useCallback(
      (notify: () => void) => {
        if (onChange == null || client == null) return NOOP_SUBSCRIBE();
        return onChange({ client, query: memoQuery }, () => notify());
      },
      [client, memoQuery],
    );
    const getSnapshot = useCallback((): query.Cached<Data> | undefined => {
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
    }, [client, memoQuery]);
    const selector = useCallback(
      (raw: query.Cached<Data> | undefined): Selected => {
        if (client == null)
          throw new DisconnectedError(`Cannot select ${name}: no Core connected.`);
        if (raw === undefined)
          throw new NotFoundError(
            `Cannot select ${name}: nothing cached. A parent must retrieve it first.`,
          );
        if (Deleted.matches<Data>(raw))
          throw new DeletedError(`${name} was deleted`, raw.corpse);
        const cached = computed.current;
        if (cached != null && cached.raw === raw && cached.query === memoQuery)
          return cached.out;
        const out = select(raw, memoQuery);
        computed.current = { raw, query: memoQuery, out };
        return out;
      },
      [client, memoQuery],
    );
    return useSyncExternalStoreWithSelector(
      subscribeToCache,
      getSnapshot,
      undefined,
      selector,
      selectedEqual,
    );
  };
  return useSelect;
};

export const createRetrieve = <Query extends query.Params, Data extends query.Data>(
  createParams: CreateRetrieveParams<Query, Data>,
): CreateRetrieveReturn<Query, Data> => {
  const locals = new WeakMap<Client, LocalCache<Data>>();
  return {
    use: (query: Query) => useSuspended({ ...createParams, query, locals }),
    useEnsure: (query: Query) => useEnsure({ ...createParams, query, locals }),
    useResult: (query: Query | null) =>
      useResultValue({ ...createParams, query, locals }),
    useInvalidate: () => useInvalidate<Query, Data>(locals),
    useTombstone: (query: Query) => useTombstone({ ...createParams, query }),
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
  };
};
