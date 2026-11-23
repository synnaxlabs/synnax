// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, errors, observe, sync, TimeSpan } from "@synnaxlabs/x";

import type { Key } from "@/channel/types.gen";

/**
 * The built-in keys of a client. Keys are known when keys is not null, even while a
 * later fetch runs or after it fails. With no keys, a non-null error means the last
 * fetch failed, and a null error means none has answered yet.
 */
export interface BuiltInsState {
  keys: Set<Key> | null;
  error: Error | null;
}

const PENDING: BuiltInsState = { keys: null, error: null };

/** How long a fetch of the keys may take before it counts as failed. */
const FETCH_TIMEOUT = TimeSpan.seconds(10);

interface BuiltInsConfig {
  /** Fetches the keys from the Core. */
  fetch: () => Promise<Key[]>;
  /** Subscribes to each new connection to the Core: 0 for a new one, then 1 and up. */
  onEpoch: (callback: (epoch: number) => void) => destructor.Destructor;
  /** Reports a failed fetch. */
  onError: (error: Error) => void;
}

const equal = (a: Set<Key>, b: Set<Key>): boolean =>
  a.size === b.size && [...a].every((key) => b.has(key));

/**
 * Holds the keys of the built-in channels, which Core services declare for their own
 * work. The first awaited read fetches them, and each later connection to the Core
 * fetches them again, keeping the last keys until the new answer lands. A client that
 * never awaits them fetches nothing. Reach it through `client.channels.builtIns`.
 */
export class BuiltIns {
  private readonly cfg: BuiltInsConfig;
  private current: BuiltInsState = PENDING;
  private readonly changes = new observe.Observer<BuiltInsState>();
  private started = false;
  private fetching: Promise<void> | null = null;
  private refetch = false;
  private generation = 0;
  private reported = false;

  constructor(cfg: BuiltInsConfig) {
    this.cfg = cfg;
    cfg.onEpoch((epoch) => {
      this.generation++;
      // A new Core behind the same address may declare other built-ins.
      if (epoch === 0) {
        this.set(PENDING);
        return;
      }
      if (!this.started) return;
      if (this.fetching != null) this.refetch = true;
      else this.fetch();
    });
  }

  /** @returns the state. It starts no fetch and keeps its identity until it changes. */
  state(): BuiltInsState {
    return this.current;
  }

  /**
   * Fetches the keys when none are known and no fetch runs, so a fetch that failed runs
   * again, and waits until no fetch runs.
   * @returns the state then. It never rejects.
   */
  async retrieve(): Promise<BuiltInsState> {
    this.started = true;
    if (this.current.keys != null) return this.current;
    if (this.fetching == null) this.fetch();
    while (this.fetching != null) await this.fetching;
    return this.current;
  }

  /** Calls handler with the state each time it changes. */
  onChange(handler: (state: BuiltInsState) => void): destructor.Destructor {
    return this.changes.onChange(handler);
  }

  private fetch(): void {
    const { generation } = this;
    this.fetching = sync
      .withTimeout(
        this.cfg.fetch(),
        FETCH_TIMEOUT,
        () => new Error(`no answer in ${FETCH_TIMEOUT.toString()}`),
      )
      .then((keys) => {
        if (generation === this.generation) this.answer(new Set(keys));
      })
      .catch((exc: unknown) => {
        if (generation === this.generation) this.fail(exc);
      })
      .finally(() => {
        this.fetching = null;
        if (!this.refetch) return;
        this.refetch = false;
        this.fetch();
      });
  }

  private answer(keys: Set<Key>): void {
    this.reported = false;
    const { current } = this;
    if (current.keys == null || !equal(current.keys, keys))
      this.set({ keys, error: null });
    else if (current.error != null) this.set({ keys: current.keys, error: null });
  }

  private fail(exc: unknown): void {
    const error = new Error("failed to fetch built-in channels", { cause: exc });
    // One report per run of failures, which a fetch that passes ends. The report's
    // message is the status description: a sentence, then the cause's own text.
    if (!this.reported) {
      this.reported = true;
      const cause = errors.fromUnknown(exc).message;
      this.cfg.onError(new Error(`Failed to fetch built-in channels. ${cause}`));
    }
    this.set({ keys: this.current.keys, error });
  }

  private set(state: BuiltInsState): void {
    this.current = state;
    this.changes.notify(state);
  }
}
