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
import {
  caseconv,
  compare,
  type destructor,
  type state,
  TimeSpan,
} from "@synnaxlabs/x";
import {
  useCallback,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useSyncExternalStore,
} from "react";
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
  setSettled,
  suspendOnFetch,
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
}

// The domain query cache interns its answers, so identity holds for anything
// it serves. This backstops a hand-written getCached that allocates instead,
// which would otherwise spin useSyncExternalStore forever.
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

const waitForCreation = <Query extends query.Params, Data extends query.Data>(
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
      if (local.epoch === epoch) setSettled(local, hash, { error });
      reject(error);
    }, NOT_FOUND_WAIT.milliseconds);
    disconnect = onChange(params, (result) => {
      if (result === undefined) return;
      finish();
      if (Deleted.matches<Data>(result))
        reject(
          new DeletedError(`${caseconv.capitalize(name)} was deleted`, result.corpse),
        );
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
        reject(
          new DeletedError(`${caseconv.capitalize(name)} was deleted`, cached.corpse),
        );
      else resolve(cached);
    }
  });

interface FetchParamsSource<
  Query extends query.Params,
  Data extends query.Data,
> extends Pick<
  CreateRetrieveParams<Query, Data>,
  "name" | "retrieve" | "onChange" | "getCached" | "awaitCreation"
> {
  local: LocalCache<Data>;
}

const fetchParamsFor = <Query extends query.Params, Data extends query.Data>({
  name,
  retrieve,
  onChange,
  getCached,
  awaitCreation = false,
  local,
}: FetchParamsSource<Query, Data>): EnsureFetchParams<Query, Data> => ({
  name,
  retrieve,
  getCached,
  local,
  // A not-found on a query that awaits creation stays pending: the reference may
  // have outrun its document's create broadcast, which the subscription will
  // deliver. Everything else settles.
  onFetchError: (params, { cause, error, hash }) =>
    awaitCreation &&
    onChange != null &&
    getCached != null &&
    NotFoundError.matches(cause)
      ? waitForCreation(params, { name, error, hash, onChange, getCached, local })
      : null,
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
  }: Context<Query, Data>,
  query: Query,
): Data => {
  const memoQuery = useMemoQuery(query, normalizeQuery);
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
  if (pending.promise == null && cached !== undefined) {
    if (Deleted.matches<Data>(cached))
      throw new DeletedError(`${caseconv.capitalize(name)} was deleted`, cached.corpse);
    return cached;
  }
  return suspendOnFetch(
    params,
    fetchParamsFor({ name, retrieve, onChange, getCached, awaitCreation, local }),
    pending,
  );
};

/**
 * A result minted per render defeats every memo keyed on it, and its status carries a
 * fresh key and timestamp. The returned hold keys the previous result on what produced
 * it and re-returns it until that changes.
 */
const useHeldResult = <V extends state.State>(): ((
  source: unknown[],
  make: () => Result<V>,
) => Result<V>) => {
  const held = useRef<{ source: unknown[]; result: Result<V> } | null>(null);
  return (source, make) => {
    const prev = held.current;
    if (
      prev != null &&
      prev.source.length === source.length &&
      prev.source.every((s, i) => s === source[i])
    )
      return prev.result;
    const result = make();
    held.current = { source, result };
    return result;
  };
};

const useResultValue = <Query extends query.Params, Data extends query.Data>(
  {
    locals,
    name,
    retrieve,
    onChange,
    getCached,
    equal,
    normalizeQuery,
    awaitCreation,
  }: Context<Query, Data>,
  q: Query | null,
): Result<Data> => {
  const memoQuery = useMemoQuery(q, normalizeQuery);
  // A retrieve whose answer never reaches the domain cache is served from the local
  // settled entry, which no subscription announces, so settling re-renders by hand.
  const [, bump] = useReducer((x: number) => x + 1, 0);
  const client = Synnax.use();

  const cached = useCachedSnapshot<Query, Data>(memoQuery, client, {
    onChange,
    getCached,
    equal,
  });

  const hold = useHeldResult<Data>();

  // A render React discards must not fetch, and nothing dedupes one once the
  // in-flight entry clears, so the cold path starts its fetch after commit.
  // A settled failure does not block the fetch: each new mount gets one fresh
  // attempt (deps hold within a mount, so a failure cannot loop).
  useEffect(() => {
    if (client == null || memoQuery == null || cached !== undefined) return;
    const local = localFor(locals, client);
    const params = { client, query: memoQuery };
    const settled = local.settled.get(query.hash(memoQuery));
    if (settled != null && "data" in settled) return;
    ensureFetch(
      params,
      fetchParamsFor({ name, retrieve, onChange, getCached, awaitCreation, local }),
    ).then(bump, bump);
  }, [client, memoQuery, cached]);

  if (client == null)
    return hold(["disabled", client], () => nullClientResult<Data>(`retrieve ${name}`));
  if (memoQuery == null)
    return hold(["disabled", memoQuery], () => noQueryResult<Data>(`retrieve ${name}`));
  if (cached !== undefined) {
    if (!Deleted.matches<Data>(cached))
      return hold(["success", cached], () =>
        successResult(`retrieved ${name}`, cached),
      );
    return hold(["deleted", cached], () =>
      errorResult(
        `retrieve ${name}`,
        new DeletedError(`${caseconv.capitalize(name)} was deleted`, cached.corpse),
      ),
    );
  }

  const local = localFor(locals, client);
  const settled = local.settled.get(query.hash(memoQuery));
  if (settled != null)
    return hold(["settled", settled], () =>
      "data" in settled
        ? successResult(`retrieved ${name}`, settled.data)
        : errorResult(`retrieve ${name}`, settled.error),
    );
  return hold(["loading", memoQuery], () => loadingResult<Data>(`retrieving ${name}`));
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
      return;
    }
  }
  suspendOnFetch(
    params,
    fetchParamsFor({ name, retrieve, onChange, getCached, awaitCreation, local }),
    pending,
  );
};

const useInvalidate = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  normalizeQuery?: <Q extends Query>(query: Q) => Q,
): ((q: Query) => void) => {
  const client = Synnax.use();
  return useCallback(
    (q: Query) => {
      if (client == null) return;
      const normalized = normalizeQuery?.(q) ?? q;
      localFor(locals, client).settled.delete(query.hash(normalized));
    },
    [client, locals],
  );
};

const useTombstone = <Query extends query.Params, Data extends query.Data>(
  { onChange, getCached, equal, normalizeQuery }: Context<Query, Data>,
  query: Query,
): Tombstone | null => {
  const memoQuery = useMemoQuery(query, normalizeQuery);
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
          throw new DeletedError(
            `${caseconv.capitalize(name)} was deleted`,
            raw.corpse,
          );
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
  context: Context<Query, Data> &
    Required<Pick<CreateRetrieveParams<Query, Data>, "getCached">>,
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
    const [, bump] = useReducer((x: number) => x + 1, 0);
    const client = Synnax.use();
    const held = useRef<{ query: Query; value: Data } | null>(null);
    const subscribeToCache = useCallback(
      (notify: () => void) => {
        if (onChange == null || client == null || memoQuery == null)
          return NOOP_SUBSCRIBE();
        return onChange({ client, query: memoQuery }, () => notify());
      },
      [client, memoQuery],
    );
    const getSnapshot = useCallback((): query.Cached<Data> | undefined => {
      if (client == null || memoQuery == null) return undefined;
      const next = getCached({ client, query: memoQuery });
      if (!isLive<Data>(next)) return next;
      const prev = held.current;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    }, [client, memoQuery]);
    const selector = useCallback(
      (raw: query.Cached<Data> | undefined): Slice<Data, Selected> => {
        if (memoQuery == null || raw === undefined) return NONE_SLICE;
        if (Deleted.matches<Data>(raw)) return { kind: "deleted", corpse: raw.corpse };
        return { kind: "live", selected: select(raw, memoQuery) };
      },
      [memoQuery],
    );
    const slice = useSyncExternalStoreWithSelector(
      subscribeToCache,
      getSnapshot,
      undefined,
      selector,
      sliceEqual,
    );
    const hold = useHeldResult<Selected>();
    // A render React discards must not fetch, and nothing dedupes one once the
    // in-flight entry clears, so the cold path starts its fetch after commit.
    // A settled failure does not block the fetch: each new mount gets one fresh
    // attempt (deps hold within a mount, so a failure cannot loop).
    useEffect(() => {
      if (client == null || memoQuery == null || slice.kind !== "none") return;
      const local = localFor(locals, client);
      const params = { client, query: memoQuery };
      const settled = local.settled.get(query.hash(memoQuery));
      if (settled != null && "data" in settled) return;
      ensureFetch(
        params,
        fetchParamsFor({ name, retrieve, onChange, getCached, awaitCreation, local }),
      ).then(bump, bump);
    }, [client, memoQuery, slice]);
    if (client == null)
      return hold(["disabled", client], () =>
        nullClientResult<Selected>(`retrieve ${name}`),
      );
    if (memoQuery == null)
      return hold(["disabled", memoQuery], () =>
        noQueryResult<Selected>(`retrieve ${name}`),
      );
    if (slice.kind === "live")
      return hold(["success", slice.selected], () =>
        successResult(`retrieved ${name}`, slice.selected),
      );
    if (slice.kind === "deleted")
      return hold(["deleted", slice.corpse], () =>
        errorResult(
          `retrieve ${name}`,
          new DeletedError(`${caseconv.capitalize(name)} was deleted`, slice.corpse),
        ),
      );
    const local = localFor(locals, client);
    const settled = local.settled.get(query.hash(memoQuery));
    if (settled != null)
      return hold(["settled", settled], () =>
        "data" in settled
          ? successResult(`retrieved ${name}`, select(settled.data, memoQuery))
          : errorResult(`retrieve ${name}`, settled.error),
      );
    return hold(["loading", memoQuery], () =>
      loadingResult<Selected>(`retrieving ${name}`),
    );
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
    useResult: (query: Query | null) => useResultValue(context, query),
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
      const { getCached } = createParams;
      if (getCached == null)
        throw new UnexpectedError(
          `Cannot create a result selector for ${createParams.name}: no getCached defined.`,
        );
      return createResultSelector({ ...context, getCached }, select, equal);
    },
  };
};
