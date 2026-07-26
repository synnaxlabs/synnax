// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type Store, type UnknownAction } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";

/**
 * Dependencies the host hands to every synchronizer callback. S and A are the
 * synchronizer's slice-composed store shape, never the root store types.
 */
export interface Params<S = unknown, A extends Action = UnknownAction> {
  /** Connected client. The host only invokes callbacks while one exists. */
  client: Synnax;
  /** Session store, for state pulls, subscriptions, and dispatch. */
  store: Store<S, A>;
}

/**
 * Lifecycle callbacks keeping session state consistent with the cluster.
 * The host runs reconcile at client-ready and on every epoch bump, and mounts
 * listen once per client, tearing it down on client swap or unmount.
 */
// The any defaults erase the store shape at the registry boundary: the host
// always calls with the full session store, which satisfies every
// synchronizer's slice-composed S.
export interface Synchronizer<S = any, A extends Action = any> {
  /** Idempotent boundary repair. Pulls state at call time. */
  reconcile: (params: Params<S, A>) => void | Promise<void>;
  /** Mounts steady-state subscriptions: client feeds and store watches. */
  listen?: (params: Params<S, A>) => destructor.Destructor;
}

/**
 * Builds a synchronizer. The hook body only captures context (flux handles,
 * adders); execution timing belongs to the host.
 */
export type Use<S = any, A extends Action = any> = () => Synchronizer<S, A>;

/**
 * Synchronizer hooks keyed by name, mounted in declaration order by the host.
 * A session slice holding cluster references without an entry in a registry
 * is a structural omission.
 */
export type Synchronizers = Record<string, Use>;

export interface CreateParams<S, A extends Action> {
  /** Subscribes to the resource type's delete signal. */
  onDelete: (client: Synnax, handler: (key: string) => void) => destructor.Destructor;
  /** Returns the subset of keys that still exist in the cluster. */
  retrieveExisting: (client: Synnax, keys: string[]) => Promise<string[]>;
  /** Reads the session's referenced keys. */
  selectKeys: (state: S) => string[];
  /** Builds the repair action pruning the given keys. */
  remove: (keys: string[]) => A;
}

/**
 * Builds a synchronizer that prunes session references to deleted resources:
 * live delete events while connected, and an existence sweep at every
 * boundary for deletes missed during the gap.
 */
export const create = <S, A extends Action>({
  onDelete,
  retrieveExisting,
  selectKeys,
  remove,
}: CreateParams<S, A>): Use<S, A> => {
  const synchronizer: Synchronizer<S, A> = {
    reconcile: async ({ client, store }) => {
      const keys = selectKeys(store.getState());
      if (keys.length === 0) return;
      const existing = new Set(await retrieveExisting(client, keys));
      const missing = keys.filter((key) => !existing.has(key));
      if (missing.length > 0) store.dispatch(remove(missing));
    },
    listen: ({ client, store }) =>
      onDelete(client, (key) => {
        if (!selectKeys(store.getState()).includes(key)) return;
        store.dispatch(remove([key]));
      }),
  };
  return () => synchronizer;
};
