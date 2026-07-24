// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { useEffect } from "react";
import { useDispatch, useStore } from "react-redux";

// Type-only: a value import of the store would cycle through its domain barrels.
import type { Action, State } from "@/session/store";
import { useEpochSweep } from "@/session/synchronizer/useEpochSweep";

/**
 * Synchronizer hooks keyed by name. Merged and mounted once via
 * useSynchronizers. A session slice holding cluster references without an
 * entry here is a structural omission.
 */
export type Synchronizers = Record<string, () => void>;

export interface CreateParams {
  /** Subscribes to the resource type's delete signal. */
  onDelete: (client: Client, handler: (key: string) => void) => destructor.Destructor;
  /** Returns the subset of keys that still exist in the cluster. */
  retrieveExisting: (client: Client, keys: string[]) => Promise<string[]>;
  /** Reads the session's referenced keys. */
  selectKeys: (state: State) => string[];
  /** Builds the repair action pruning the given keys. */
  remove: (keys: string[]) => Action;
}

/**
 * Builds a synchronizer hook that prunes session references to deleted
 * resources: live delete events while connected, and an existence sweep on
 * every reconnect for deletes missed during the gap.
 */
export const create =
  ({ onDelete, retrieveExisting, selectKeys, remove }: CreateParams): (() => void) =>
  () => {
    const dispatch = useDispatch<Dispatch<Action>>();
    const store = useStore<State>();
    const client = Synnax.use();
    useEffect(() => {
      if (client == null) return;
      return onDelete(client, (key) => {
        if (!selectKeys(store.getState()).includes(key)) return;
        dispatch(remove([key]));
      });
    }, [client, dispatch, store]);
    useEpochSweep(async () => {
      if (client == null) return;
      const keys = selectKeys(store.getState());
      if (keys.length === 0) return;
      const existing = new Set(await retrieveExisting(client, keys));
      const missing = keys.filter((key) => !existing.has(key));
      if (missing.length > 0) dispatch(remove(missing));
    });
  };
