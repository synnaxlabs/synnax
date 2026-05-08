// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { useCallback, useSyncExternalStore } from "react";

import { type base } from "@/flux/base";
import { useStore } from "@/flux/Provider";
import { createUpdate } from "@/flux/update";
import { Synnax } from "@/synnax";

const DEFAULT_COALESCE_MS = 500;
const DEFAULT_STACK_CAP = 200;
const STALE_AUTO_ADVANCE_CAP = 10;

export interface DispatchSendParams<Key extends record.Key, Action> {
  client: Client;
  key: Key;
  actions: Action[];
  sessionKey: string;
}

export interface DispatchReducer<State extends base.Shape, Action> {
  (state: State, actions: Action[]): { next: State; inverse: Action[] };
}

export interface CreateDispatchParams<
  Key extends record.Key,
  State extends base.Shape,
  Action,
  ScopedStore extends base.Store,
  SK extends keyof ScopedStore & string,
> {
  /** Human-readable name used in status messages (e.g., "schematic"). */
  name: string;
  /** Key into the substore that holds documents of this type. */
  storeKey: SK;
  /** Pure reducer producing the next state and the inverse of the applied actions. */
  reduce: DispatchReducer<State, Action>;
  /**
   * Optional pre-processing step that may expand the user's action list with
   * additional actions (e.g., synthesized companion mutations like edge-segment
   * updates triggered by a node move). The augmented list is used both for the
   * local apply and for the server send. Skipped for internal dispatches
   * (undo/redo), whose action lists are pre-recorded inverses that should not
   * be re-augmented.
   */
  preprocess?: (state: State, actions: Action[]) => Action[];
  /** Sends the actions to the server. The substrate calls this after the local apply. */
  send: (params: DispatchSendParams<Key, Action>) => Promise<void>;
  /** Returns true when the action contributes to the undo stack. Defaults to always true. */
  isUndoable?: (action: Action) => boolean;
  /** Classifies a transaction by kind for coalescing. Defaults to "default". */
  kindOf?: (actions: Action[]) => string;
  /** Coalesce window in milliseconds. Defaults to 500. */
  coalesceMs?: number;
  /** Maximum number of entries kept on the undo stack. Defaults to 200. */
  stackCap?: number;
}

interface StackEntry<Action> {
  forward: Action[];
  inverse: Action[];
  kind: string;
  ts: number;
  /** Keys this transaction targets, used for stale detection. */
  targets: Set<string>;
}

interface Stacks<Action> {
  undo: StackEntry<Action>[];
  redo: StackEntry<Action>[];
  /** Keys touched by remote (other-session) actions, used for stale detection. */
  touchedByRemote: Map<string, number>;
}

export interface DispatchInput<Key extends record.Key, Action> {
  key: Key;
  actions: Action | Action[];
}

export interface BeginTransactionInput<Key extends record.Key> {
  key: Key;
  kind?: string;
}

export interface Transaction<Action> {
  add: (actions: Action | Action[]) => void;
  commit: () => Promise<void>;
  abort: () => void;
  readonly committed: boolean;
  readonly aborted: boolean;
}

export interface UseDispatchReturn<Key extends record.Key, Action> {
  dispatch: (input: DispatchInput<Key, Action>) => void;
  dispatchAsync: (input: DispatchInput<Key, Action>) => Promise<boolean>;
  beginTransaction: (input: BeginTransactionInput<Key>) => Transaction<Action>;
}

export interface UseUndoReturn {
  undo: () => void;
  canUndo: boolean;
}

export interface UseRedoReturn {
  redo: () => void;
  canRedo: boolean;
}

export interface CreateDispatchReturn<Key extends record.Key, Action> {
  useDispatch: () => UseDispatchReturn<Key, Action>;
  useUndo: (input: { key: Key }) => UseUndoReturn;
  useRedo: (input: { key: Key }) => UseRedoReturn;
  /**
   * Records the keys touched by a remote (other-session) action list so that
   * subsequent undo calls can detect stale entries. The document's action
   * listener calls this whenever it applies a non-self-originated action;
   * the substrate does not subscribe to cluster channels itself.
   */
  notifyRemoteActions: (key: Key, actions: Action[]) => void;
}

interface InternalDispatchInput<Key extends record.Key, Action> {
  key: Key;
  actions: Action[];
  /**
   * Pre-computed inverse for the actions. When provided, the substrate uses it
   * verbatim instead of re-running the reducer to derive one. Used by the
   * transaction API to ship the accumulated inverse, and by the undo/redo path
   * to avoid double-reducing.
   */
  precomputedInverse?: Action[];
  /** Reason for the dispatch. Internal dispatches do not push to the stack. */
  reason: "user" | "internal";
  /** Override for the kind tag; otherwise computed via config.kindOf. */
  kindOverride?: string;
  /** Targets to record on the new stack entry. Computed if absent. */
  targetsOverride?: Set<string>;
}

const collectTargets = <Action>(actions: Action[]): Set<string> => {
  const targets = new Set<string>();
  for (const a of actions) {
    const r = a as Record<string, unknown>;
    for (const v of Object.values(r)) {
      if (v == null || typeof v !== "object") continue;
      const sub = v as Record<string, unknown>;
      if (typeof sub.key === "string") targets.add(sub.key);
      const node = sub.node as Record<string, unknown> | undefined;
      if (node != null && typeof node.key === "string") targets.add(node.key);
      const edge = sub.edge as Record<string, unknown> | undefined;
      if (edge != null && typeof edge.key === "string") targets.add(edge.key);
    }
  }
  return targets;
};

export const createDispatch = <
  Key extends record.Key,
  State extends base.Shape,
  Action,
  ScopedStore extends base.Store,
  SK extends keyof ScopedStore & string,
>(
  config: CreateDispatchParams<Key, State, Action, ScopedStore, SK>,
): CreateDispatchReturn<Key, Action> => {
  const {
    name,
    storeKey,
    reduce,
    preprocess,
    send,
    isUndoable = () => true,
    kindOf = () => "default",
    coalesceMs = DEFAULT_COALESCE_MS,
    stackCap = DEFAULT_STACK_CAP,
  } = config;

  const stacksByKey = new Map<Key, Stacks<Action>>();
  const stackListeners = new Map<Key, Set<() => void>>();

  const getStacks = (key: Key): Stacks<Action> => {
    let s = stacksByKey.get(key);
    if (s == null) {
      s = { undo: [], redo: [], touchedByRemote: new Map() };
      stacksByKey.set(key, s);
    }
    return s;
  };

  const subscribeToStacks = (key: Key, callback: () => void): (() => void) => {
    let listeners = stackListeners.get(key);
    if (listeners == null) {
      listeners = new Set();
      stackListeners.set(key, listeners);
    }
    listeners.add(callback);
    return () => {
      listeners?.delete(callback);
    };
  };

  const notifyStackChange = (key: Key) => {
    stackListeners.get(key)?.forEach((cb) => cb());
  };

  const coalesceTop = (stacks: Stacks<Action>) => {
    const undo = stacks.undo;
    if (undo.length < 2) return;
    const top = undo[undo.length - 1];
    const prev = undo[undo.length - 2];
    if (top.kind !== prev.kind) return;
    if (top.ts - prev.ts > coalesceMs) return;
    if (top.targets.size !== prev.targets.size) return;
    for (const t of top.targets) if (!prev.targets.has(t)) return;
    // Merge top into prev.
    prev.forward.push(...top.forward);
    prev.inverse.unshift(...top.inverse);
    prev.ts = top.ts;
    undo.pop();
  };

  const pushUndo = (
    key: Key,
    forward: Action[],
    inverse: Action[],
    kind: string,
    targets: Set<string>,
  ) => {
    const stacks = getStacks(key);
    stacks.undo.push({
      forward: [...forward],
      inverse: [...inverse],
      kind,
      ts: Date.now(),
      targets,
    });
    stacks.redo = [];
    coalesceTop(stacks);
    if (stacks.undo.length > stackCap)
      stacks.undo.splice(0, stacks.undo.length - stackCap);
    notifyStackChange(key);
  };

  const isEntryStale = (key: Key, entry: StackEntry<Action>): boolean => {
    const stacks = getStacks(key);
    for (const t of entry.targets) {
      const touchedAt = stacks.touchedByRemote.get(t);
      if (touchedAt != null && touchedAt > entry.ts) return true;
    }
    return false;
  };

  type InternalParams = InternalDispatchInput<Key, Action>;

  const { useUpdate } = createUpdate<InternalParams, ScopedStore>({
    name,
    verbs: { present: "dispatch", past: "dispatched", participle: "dispatching" },
    update: async ({ client, data, store, rollbacks }) => {
      const { key, reason, precomputedInverse } = data;
      const subStore = (store as Record<string, unknown>)[storeKey] as
        | (ScopedStore[SK] & {
            get: (k: Key) => State | undefined;
            set: (k: Key, v: State) => () => void;
          })
        | undefined;
      const current = subStore?.get(key);
      let actions = data.actions;
      if (
        subStore != null &&
        current != null &&
        preprocess != null &&
        reason === "user"
      )
        actions = preprocess(current, actions);
      if (subStore != null && current != null) {
        const { next, inverse } = reduce(current, actions);
        rollbacks.push(subStore.set(key, next));
        if (reason === "user") {
          const undoableActions = actions.filter(isUndoable);
          if (undoableActions.length > 0) {
            const finalInverse = precomputedInverse ?? inverse;
            const kind = data.kindOverride ?? kindOf(actions);
            const targets = data.targetsOverride ?? collectTargets(actions);
            pushUndo(key, actions, finalInverse, kind, targets);
            // Roll back the stack push if the server rejects.
            rollbacks.push(() => {
              const stacks = getStacks(key);
              stacks.undo.pop();
              notifyStackChange(key);
            });
          }
        }
      }
      await send({ client, key, actions, sessionKey: client.key });
      return data;
    },
  });

  const useDispatch = (): UseDispatchReturn<Key, Action> => {
    const { update, updateAsync } = useUpdate();
    const store = useStore<ScopedStore>();
    const client = Synnax.use();

    const dispatchAsync = useCallback(
      (input: DispatchInput<Key, Action>): Promise<boolean> => {
        const actions = Array.isArray(input.actions) ? input.actions : [input.actions];
        return updateAsync({ key: input.key, actions, reason: "user" });
      },
      [updateAsync],
    );

    const dispatch = useCallback(
      (input: DispatchInput<Key, Action>) => void dispatchAsync(input),
      [dispatchAsync],
    );

    const beginTransaction = useCallback(
      (input: BeginTransactionInput<Key>): Transaction<Action> => {
        const { key, kind } = input;
        const subStore = (store as Record<string, unknown>)[storeKey] as
          | {
              get: (k: Key) => State | undefined;
              set: (k: Key, v: State) => () => void;
            }
          | undefined;
        const initial = subStore?.get(key);

        let current: State | undefined = initial;
        const accumulatedForward: Action[] = [];
        const accumulatedInverse: Action[] = [];
        const accumulatedTargets = new Set<string>();
        let committed = false;
        let aborted = false;
        let lastRollback: (() => void) | undefined;

        const tx: Transaction<Action> = {
          get committed() {
            return committed;
          },
          get aborted() {
            return aborted;
          },
          add: (actions) => {
            if (committed || aborted) throw new Error("transaction already finalized");
            if (subStore == null || current == null) return;
            const arr = Array.isArray(actions) ? actions : [actions];
            const augmented = preprocess != null ? preprocess(current, arr) : arr;
            const { next, inverse } = reduce(current, augmented);
            lastRollback = subStore.set(key, next);
            current = next;
            accumulatedForward.push(...augmented);
            accumulatedInverse.unshift(...inverse);
            for (const t of collectTargets(augmented)) accumulatedTargets.add(t);
          },
          commit: async () => {
            if (committed || aborted) return;
            committed = true;
            if (accumulatedForward.length === 0) return;
            // Send to server with the full transaction. The substrate's
            // update path will re-apply locally (a no-op since the state
            // already reflects the accumulated changes — reduce is
            // idempotent over its own output) and push to the stack.
            //
            // Simpler approach: we already applied locally; just send
            // and push manually to avoid double-apply.
            if (client == null) return;
            try {
              await send({
                client,
                key,
                actions: accumulatedForward,
                sessionKey: client.key,
              });
              // Push to stack only after server accepts.
              const undoable = accumulatedForward.filter(isUndoable);
              if (undoable.length > 0)
                pushUndo(
                  key,
                  accumulatedForward,
                  accumulatedInverse,
                  kind ?? kindOf(accumulatedForward),
                  accumulatedTargets,
                );
            } catch {
              // Server rejected: roll back the optimistic local state.
              if (subStore != null && initial != null) subStore.set(key, initial);
              else lastRollback?.();
            }
          },
          abort: () => {
            if (committed || aborted) return;
            aborted = true;
            if (subStore != null && initial != null) subStore.set(key, initial);
            else lastRollback?.();
          },
        };
        return tx;
      },
      [store, client],
    );

    return { dispatch, dispatchAsync, beginTransaction };
  };

  const useUndo = (input: { key: Key }): UseUndoReturn => {
    const { key } = input;
    const { updateAsync } = useUpdate();

    const subscribe = useCallback(
      (cb: () => void) => subscribeToStacks(key, cb),
      [key],
    );
    const getSnapshot = useCallback(() => getStacks(key).undo.length, [key]);
    const undoLength = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

    const undo = useCallback(() => {
      const stacks = getStacks(key);
      let staleSkipped = 0;
      while (stacks.undo.length > 0) {
        const entry = stacks.undo.pop()!;
        notifyStackChange(key);
        if (isEntryStale(key, entry)) {
          staleSkipped += 1;
          if (staleSkipped >= STALE_AUTO_ADVANCE_CAP) return;
          continue;
        }
        void updateAsync({
          key,
          actions: entry.inverse,
          reason: "internal",
          precomputedInverse: entry.forward,
        }).then((ok) => {
          if (ok) {
            stacks.redo.push({ ...entry, ts: Date.now() });
            notifyStackChange(key);
          }
        });
        return;
      }
    }, [key, updateAsync]);

    return { undo, canUndo: undoLength > 0 };
  };

  const useRedo = (input: { key: Key }): UseRedoReturn => {
    const { key } = input;
    const { updateAsync } = useUpdate();

    const subscribe = useCallback(
      (cb: () => void) => subscribeToStacks(key, cb),
      [key],
    );
    const getSnapshot = useCallback(() => getStacks(key).redo.length, [key]);
    const redoLength = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

    const redo = useCallback(() => {
      const stacks = getStacks(key);
      const entry = stacks.redo.pop();
      if (entry == null) return;
      notifyStackChange(key);
      void updateAsync({
        key,
        actions: entry.forward,
        reason: "internal",
        precomputedInverse: entry.inverse,
      }).then((ok) => {
        if (ok) {
          stacks.undo.push({ ...entry, ts: Date.now() });
          notifyStackChange(key);
        }
      });
    }, [key, updateAsync]);

    return { redo, canRedo: redoLength > 0 };
  };

  const notifyRemoteActions = (key: Key, actions: Action[]) => {
    const stacks = getStacks(key);
    const ts = Date.now();
    for (const t of collectTargets(actions)) stacks.touchedByRemote.set(t, ts);
  };

  return { useDispatch, useUndo, useRedo, notifyRemoteActions };
};
