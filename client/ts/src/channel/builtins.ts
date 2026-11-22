// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, observe } from "@synnaxlabs/x";

import { type Key } from "@/channel/types.gen";
import { isConnectionError } from "@/errors";

/**
 * The built-in keys of a client: the keys of the last fetch that answered, and the
 * error of this connection's fetch when it failed. Both are null while the first fetch
 * runs.
 */
export interface BuiltInsState {
  keys: Set<Key> | null;
  error: Error | null;
}

interface BuiltInsConfig {
  /** Fetches the keys from the Core. */
  fetch: () => Promise<Key[]>;
  /** Subscribes to each new connection to the Core. */
  onEpoch: (callback: () => void) => destructor.Destructor;
  /** Reports a failed fetch. */
  onError: (error: Error) => void;
}

/**
 * Holds the keys of the built-in channels, which Core services declare for their own
 * work. The first read starts the fetch, and each later connection to the Core fetches
 * them again, keeping the last keys until the new answer lands. A client that never
 * reads them fetches nothing. Reach it through `client.channels.builtIns`.
 */
export class BuiltIns {
  private readonly cfg: BuiltInsConfig;
  private current: BuiltInsState = { keys: null, error: null };
  private readonly changes = new observe.Observer<BuiltInsState>();
  private started = false;
  private epoch = 0;

  constructor(cfg: BuiltInsConfig) {
    this.cfg = cfg;
    cfg.onEpoch(() => {
      this.epoch++;
      if (this.current.error != null)
        this.set({ keys: this.current.keys, error: null });
      if (this.started) this.fetch();
    });
  }

  /** @returns the built-in keys, or null before the first fetch answers. */
  get(): Set<Key> | null {
    return this.state().keys;
  }

  /** @returns the state, which keeps its identity until it changes. */
  state(): BuiltInsState {
    this.start();
    return this.current;
  }

  /**
   * @returns the built-in keys, once the fetch of this connection answers when none
   * have arrived yet.
   * @throws the error of this connection's fetch when it failed.
   */
  async retrieve(): Promise<Set<Key>> {
    const { keys, error } = this.state();
    if (keys != null) return keys;
    if (error != null) throw error;
    return await new Promise((resolve, reject) => {
      const release = this.changes.onChange((next) => {
        if (next.keys != null) {
          release();
          resolve(next.keys);
        } else if (next.error != null) {
          release();
          reject(next.error);
        }
      });
    });
  }

  /** Calls handler with the state each time it changes. */
  onChange(handler: (state: BuiltInsState) => void): destructor.Destructor {
    this.start();
    return this.changes.onChange(handler);
  }

  private start(): void {
    if (this.started) return;
    this.started = true;
    if (this.epoch > 0) this.fetch();
  }

  private fetch(): void {
    const { epoch } = this;
    this.cfg
      .fetch()
      .then((keys) => {
        if (epoch === this.epoch) this.set({ keys: new Set(keys), error: null });
      })
      .catch((exc: unknown) => {
        // A later connection fetches the keys again.
        if (epoch !== this.epoch || isConnectionError(exc)) return;
        const error = new Error("failed to fetch built-in channels", { cause: exc });
        this.cfg.onError(error);
        this.set({ keys: this.current.keys, error });
      });
  }

  private set(state: BuiltInsState): void {
    this.current = state;
    this.changes.notify(state);
  }
}
