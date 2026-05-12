// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { errorResult, type Result, successResult } from "@/flux/result";
import { type state } from "@/state";

/// Deterministically serializes a query to a stable string. Keys are sorted
/// recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key.
/// Works for plain objects, arrays, and primitives. Functions, Maps, Sets, and
/// other complex types are not supported as query inputs.
export const hashQuery = (query: unknown): string => {
  if (query == null) return "null";
  if (typeof query !== "object") return JSON.stringify(query);
  if (Array.isArray(query)) return `[${query.map(hashQuery).join(",")}]`;
  const entries = Object.entries(query as Record<string, unknown>).sort(([a], [b]) =>
    a < b ? -1 : a > b ? 1 : 0,
  );
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${hashQuery(v)}`).join(",")}}`;
};

/// Per-client cache of in-flight and resolved queries, keyed by a deterministic
/// hash of the query input. Holds `Result<T>` directly: loading results carry
/// an in-flight promise, success results carry the value, error results carry
/// the failure status. The cache auto-transitions a loading result with a
/// promise to a success or error result when the promise settles, gated by
/// identity so that listener-driven mid-flight updates are not clobbered by
/// a late resolution. Decouples the suspension state machine from the
/// per-record value store: per-record stores hold canonical values keyed by
/// record ID, while the query cache holds query lifecycles keyed by query
/// shape. A single-record retrieve with shape `{key: X}` and a list retrieve
/// with shape `{workspace, filter}` both live here, addressed by their
/// respective hashes.
export class QueryCache {
  private readonly entries: Map<string, Result<state.State>> = new Map();
  private readonly subscribers: Map<string, Set<() => void>> = new Map();

  /// Returns the result for the given query hash, or undefined if absent.
  get<T extends state.State>(hash: string): Result<T> | undefined {
    return this.entries.get(hash) as Result<T> | undefined;
  }

  /// Stores a result for the given query hash. If the result is a loading
  /// variant with an attached promise, the cache wires up `.then` handlers
  /// that auto-transition the entry to success or error when the promise
  /// settles.
  ///
  /// `equal` is an optional value-equality check applied on success-to-success
  /// transitions. When the new and previous data are equal under it, the entry
  /// is left in place and no notification fires - subscribers don't re-render.
  /// Selectors use this to skip re-renders when a derived slice didn't change
  /// despite the parent record changing.
  set<T extends state.State>(
    hash: string,
    result: Result<T>,
    equal?: (a: T, b: T) => boolean,
  ): void {
    if (this.skipBecauseEqual(hash, result, equal)) return;
    this.entries.set(hash, result);
    this.notify(hash);
    if (result.variant !== "loading" || result.promise == null) return;
    const name = result.name ?? "resource";
    const promise = result.promise;
    promise.then(
      (value) =>
        this.replaceIfStill<T>(
          hash,
          result,
          successResult(`retrieved ${name}`, value),
          equal,
        ),
      (reason: unknown) => {
        const err = reason instanceof Error ? reason : new Error(String(reason));
        this.replaceIfStill<T>(hash, result, errorResult(`retrieve ${name}`, err));
      },
    );
  }

  private skipBecauseEqual<T extends state.State>(
    hash: string,
    next: Result<T>,
    equal?: (a: T, b: T) => boolean,
  ): boolean {
    if (equal == null || next.variant !== "success") return false;
    const prev = this.entries.get(hash) as Result<T> | undefined;
    return prev?.variant === "success" && equal(prev.data, next.data);
  }

  /// Removes the entry for the given hash. The next read kicks off a fresh
  /// fetch.
  invalidate(hash: string): void {
    if (!this.entries.delete(hash)) return;
    this.notify(hash);
  }

  /// Subscribes to changes for the given query hash. The callback fires on
  /// every state transition for that hash. Returns a destructor that
  /// unsubscribes.
  subscribe(hash: string, callback: () => void): destructor.Destructor {
    let set = this.subscribers.get(hash);
    if (set == null) {
      set = new Set();
      this.subscribers.set(hash, set);
    }
    set.add(callback);
    return () => {
      const subs = this.subscribers.get(hash);
      if (subs == null) return;
      subs.delete(callback);
      if (subs.size === 0) this.subscribers.delete(hash);
    };
  }

  clear(): void {
    this.entries.clear();
  }

  /// Replace the entry only if it is still the same instance the caller
  /// expects. Guards against a channel listener (or another caller) replacing
  /// the entry mid-flight: a late promise resolution must not clobber a
  /// listener's update.
  private replaceIfStill<T extends state.State>(
    hash: string,
    expected: Result<T>,
    next: Result<T>,
    equal?: (a: T, b: T) => boolean,
  ): void {
    if (this.entries.get(hash) !== expected) return;
    if (this.skipBecauseEqual(hash, next, equal)) return;
    this.entries.set(hash, next);
    this.notify(hash);
  }

  private notify(hash: string): void {
    this.subscribers.get(hash)?.forEach((cb) => cb());
  }
}
