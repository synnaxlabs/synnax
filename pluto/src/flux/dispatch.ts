// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache, type dispatch, type Synnax as Client } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { useCallback, useSyncExternalStore } from "react";

import { errorResult } from "@/flux/result";
import { useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

export interface DispatchInput<Key extends record.Key, Action> {
  key: Key;
  actions: Action | Action[];
}

const INERT_TX: dispatch.Transaction<unknown> = {
  add: () => {},
  commit: async () => false,
  abort: () => {},
};

export interface CreateDispatchParams<
  Key extends record.Key,
  State extends cache.Data,
  Action,
> {
  /** Selects the domain's dispatch surface off the client. */
  domain: (client: Client) => dispatch.Domain<Key, State, Action>;
  /**
   * Rewrites an action batch against the current document before replay and
   * send. Lives here rather than in the domain client when the rewrite needs
   * visualization-layer context (e.g. diagram geometry).
   */
  preprocess?: dispatch.Preprocess<State, Action>;
}

export interface UseDispatchReturn<Key extends record.Key, Action> {
  dispatch: (input: DispatchInput<Key, Action>) => void;
  dispatchAsync: (input: DispatchInput<Key, Action>) => Promise<boolean>;
  beginTransaction: (input: {
    key: Key;
    kind?: string;
  }) => dispatch.Transaction<Action>;
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
  useUndo: (params: { key: Key }) => UseUndoReturn;
  useRedo: (params: record.Keyed<Key>) => UseRedoReturn;
  useSingleDispatch: (params: record.Keyed<Key>) => (action: Action | Action[]) => void;
}

export const createDispatch = <
  Key extends record.Key,
  State extends cache.Data,
  Action,
>({
  domain,
  preprocess,
}: CreateDispatchParams<Key, State, Action>): CreateDispatchReturn<Key, Action> => {
  const useCanReverse = ({
    key,
    side,
  }: {
    key: Key;
    side: "undo" | "redo";
  }): boolean => {
    const client = Synnax.use();
    const subscribe = useCallback(
      (onStoreChange: () => void) => {
        if (client == null) return () => {};
        return domain(client).onUndoStateChange(onStoreChange, key);
      },
      [client, key],
    );
    const getSnapshot = useCallback(() => {
      if (client == null) return false;
      const d = domain(client);
      return side === "undo" ? d.hasUndo(key) : d.hasRedo(key);
    }, [client, key, side]);
    return useSyncExternalStore(subscribe, getSnapshot);
  };

  const useDispatch = () => {
    const client = Synnax.use();
    const addStatus = useAdder();

    const dispatchAsync = useCallback(
      async (input: DispatchInput<Key, Action>): Promise<boolean> => {
        if (client == null) return false;
        try {
          return await domain(client).dispatch(input.key, input.actions, {
            preprocess,
          });
        } catch (e) {
          addStatus(errorResult("dispatch action", e).status);
          return false;
        }
      },
      [client, addStatus],
    );

    const dispatch = useCallback(
      (input: DispatchInput<Key, Action>) => void dispatchAsync(input),
      [dispatchAsync],
    );

    const beginTransaction = useCallback(
      (input: { key: Key; kind?: string }): dispatch.Transaction<Action> => {
        if (client == null) return INERT_TX;
        const tx = domain(client).beginTransaction(input.key, input.kind);
        return {
          ...tx,
          // The substrate rolls back before rethrowing; surface the error to
          // the user and preserve the resolve-false contract for callers.
          commit: async () => {
            try {
              return await tx.commit();
            } catch (e) {
              addStatus(errorResult("commit transaction", e).status);
              return false;
            }
          },
        };
      },
      [client, addStatus],
    );

    return { dispatch, dispatchAsync, beginTransaction };
  };

  const useUndo = ({ key }: { key: Key }) => {
    const client = Synnax.use();
    const addStatus = useAdder();
    const canUndo = useCanReverse({ key, side: "undo" });
    const undo = useCallback(() => {
      if (client == null) return;
      domain(client)
        .undo(key)
        .catch((e: unknown) => addStatus(errorResult("undo action", e).status));
    }, [client, key, addStatus]);
    return { undo, canUndo };
  };

  const useRedo = ({ key }: record.Keyed<Key>) => {
    const client = Synnax.use();
    const addStatus = useAdder();
    const canRedo = useCanReverse({ key, side: "redo" });
    const redo = useCallback(() => {
      if (client == null) return;
      domain(client)
        .redo(key)
        .catch((e: unknown) => addStatus(errorResult("redo action", e).status));
    }, [client, key, addStatus]);
    return { redo, canRedo };
  };

  const useSingleDispatch = ({
    key,
  }: record.Keyed<Key>): ((action: Action | Action[]) => void) => {
    const { dispatch } = useDispatch();
    return useCallback(
      (actions: Action | Action[]) => dispatch({ key, actions }),
      [key, dispatch],
    );
  };

  return { useDispatch, useUndo, useRedo, useSingleDispatch };
};
