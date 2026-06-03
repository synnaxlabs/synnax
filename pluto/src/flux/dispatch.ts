// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type destructor, errors, id, type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type base } from "@/flux/base";
import {
  type ReplayResult,
  type Transaction,
  type UndoableUnaryStore,
} from "@/flux/base/undoable";
import { useStore } from "@/flux/Provider";
import { errorResult } from "@/flux/result";
import { createSelector } from "@/flux/select";
import { type Adder, useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

export interface DispatchInput<Key extends record.Key, Action> {
  key: Key;
  actions: Action | Action[];
}

const INERT_TX: Transaction<never> = {
  add: () => {},
  commit: async () => false,
  abort: () => {},
};

export interface CreateDispatchParams<
  Key extends record.Key,
  Action,
  SK extends string,
> {
  storeKey: SK;
  send: (params: {
    client: Client;
    key: Key;
    actions: Action[];
    dispatchKey: string;
  }) => Promise<void>;
}

export const createDispatch = <
  Key extends record.Key,
  State extends base.Data,
  Action,
  SK extends string,
  ScopedStore extends base.Store & Record<SK, UndoableUnaryStore<Key, State, Action>>,
>({
  storeKey,
  send,
}: CreateDispatchParams<Key, Action, SK>) => {
  // Reduce + send with rollback. Used by dispatch, undo, and redo. The
  // `commitStack` closure applies the per-op stack mutation (push entry on
  // dispatch; transition undo↔redo on undo/redo) and returns its rollback.
  // A fresh dispatchKey is generated per batch and registered as outstanding
  // *before* the network send: the broadcast echo can race with the HTTP
  // response back to us, and registering up front means the echo will always
  // find the entry and skip its redundant reduce.
  const apply = async (
    store: ScopedStore,
    client: Client,
    key: Key,
    actions: Action[],
    commitStack: (r: ReplayResult<Action>) => destructor.Destructor,
    addStatus: Adder,
    op: string,
    skipPreprocess = false,
  ): Promise<boolean> => {
    const r = store[storeKey].replay(key, actions, { skipPreprocess });
    if (r == null) return false;
    const stackRollback = commitStack(r);
    const dispatchKey = id.create();
    store[storeKey].registerOutstandingDispatch(key, dispatchKey);
    try {
      await send({ client, key, actions: r.processed, dispatchKey });
      return true;
    } catch (e) {
      stackRollback();
      r.rollback();
      addStatus(errorResult(op, e).status);
      return false;
    }
  };

  const useCanReverse = createSelector<
    ScopedStore,
    { key: Key; side: "undo" | "redo" },
    boolean
  >({
    subscribe: (s, { key }, notify) => s[storeKey].onUndoStateChange(notify, key),
    select: (s, { key, side }) =>
      side === "undo" ? s[storeKey].hasUndo(key) : s[storeKey].hasRedo(key),
  });

  const useDispatch = () => {
    const store = useStore<ScopedStore>();
    const client = Synnax.use();
    const addStatus = useAdder();

    const dispatchAsync = useCallback(
      async (input: DispatchInput<Key, Action>): Promise<boolean> => {
        if (client == null) return false;
        const actions = Array.isArray(input.actions) ? input.actions : [input.actions];
        return await apply(
          store,
          client,
          input.key,
          actions,
          ({ processed, inverse, targets }) =>
            store[storeKey].recordEntry(input.key, processed, inverse, targets),
          addStatus,
          "dispatch action",
        );
      },
      [store, client, addStatus],
    );

    const dispatch = useCallback(
      (input: DispatchInput<Key, Action>) => void dispatchAsync(input),
      [dispatchAsync],
    );

    const beginTransaction = useCallback(
      (input: { key: Key; kind?: string }): Transaction<Action> => {
        if (client == null) return INERT_TX as Transaction<Action>;
        // Wrap `send` so the substrate's rollback runs as before, but the
        // error also surfaces to the user via the status aggregator. A
        // dispatchKey is allocated per commit (one batch, one outstanding
        // entry) and registered before the network send so the broadcast
        // echo wins the race.
        const wrappedSend = async (actions: Action[]) => {
          const dispatchKey = id.create();
          store[storeKey].registerOutstandingDispatch(input.key, dispatchKey);
          try {
            await send({ client, key: input.key, actions, dispatchKey });
          } catch (e) {
            addStatus(errorResult("commit transaction", e).status);
            throw errors.fromUnknown(e);
          }
        };
        return store[storeKey].beginTransaction(input.key, wrappedSend, input.kind);
      },
      [store, client, addStatus],
    );

    return { dispatch, dispatchAsync, beginTransaction };
  };

  const useUndo = ({ key }: { key: Key }) => {
    const store = useStore<ScopedStore>();
    const client = Synnax.use();
    const addStatus = useAdder();
    const canUndo = useCanReverse({ key, side: "undo" });
    const undo = useCallback(() => {
      if (client == null) return;
      const prepared = store[storeKey].prepareUndo(key);
      if (prepared == null) return;
      void apply(
        store,
        client,
        key,
        prepared.actions,
        () => prepared.commit(),
        addStatus,
        "undo action",
        true,
      );
    }, [store, client, key, addStatus]);
    return { undo, canUndo };
  };

  const useRedo = ({ key }: { key: Key }) => {
    const store = useStore<ScopedStore>();
    const client = Synnax.use();
    const addStatus = useAdder();
    const canRedo = useCanReverse({ key, side: "redo" });
    const redo = useCallback(() => {
      if (client == null) return;
      const prepared = store[storeKey].prepareRedo(key);
      if (prepared == null) return;
      void apply(
        store,
        client,
        key,
        prepared.actions,
        () => prepared.commit(),
        addStatus,
        "redo action",
        true,
      );
    }, [store, client, key, addStatus]);
    return { redo, canRedo };
  };

  return { useDispatch, useUndo, useRedo };
};
