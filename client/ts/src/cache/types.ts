// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors, type primitive } from "@synnaxlabs/x";

import { type State } from "@/cache/state";

/** Options to control async operations. */
export interface FetchOptions {
  /** Optional AbortSignal to cancel the operation */
  signal?: AbortSignal;
}

/**
 * Plain-data shape used to address a record in the cache query system. Queries
 * are hashed by {@link hashQuery} to produce stable cache keys, so they must be
 * primitives, arrays of queries, plain string-keyed objects of queries, or
 * class instances implementing {@link primitive.Hashable}. Maps, Sets, Dates,
 * and non-Hashable class instances are rejected at compile time because their
 * fields don't recursively reduce to {@link Query}.
 *
 * Consumer query types must be declared with `type` aliases, not `interface`,
 * because TypeScript interfaces lack an implicit string index signature and
 * fail the `{ readonly [k: string]: Query }` branch.
 */
export type Query =
  | string
  | number
  | bigint
  | boolean
  | null
  | undefined
  | primitive.Hashable
  | readonly Query[]
  | { readonly [key: string]: Query };

/**
 * Shape of values returned and stored by the cache query system. Unlike
 * {@link Query}, data values do not need to be hashable — they may carry
 * arbitrary state including class instances or non-serializable references.
 */
export type Data = State;

/**
 * Handles an error by reporting it and continuing. Accepts either a function
 * to run (sync or async) or an already-caught exception. `skip` suppresses
 * reporting for matching errors.
 */
export interface ErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): void;
  (exc: unknown, message?: string, skip?: errors.Matchable | errors.Matchable[]): void;
}

/** {@link ErrorHandler} whose function form resolves after the work settles. */
export interface AsyncErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
  (
    exc: unknown,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
}

export interface Verbs {
  present: string;
  past: string;
  participle: string;
}

export const RENAME_VERBS: Verbs = {
  present: "rename",
  past: "renamed",
  participle: "renaming",
};

export const DELETE_VERBS: Verbs = {
  present: "delete",
  past: "deleted",
  participle: "deleting",
};

export const UPDATE_VERBS: Verbs = {
  present: "update",
  past: "updated",
  participle: "updating",
};

export const CREATE_VERBS: Verbs = {
  present: "create",
  past: "created",
  participle: "creating",
};

export const SNAPSHOT_VERBS: Verbs = {
  present: "snapshot",
  past: "snapshotted",
  participle: "snapshotting",
};

export const COPY_VERBS: Verbs = {
  present: "copy",
  past: "copied",
  participle: "copying",
};

export const SET_VERBS: Verbs = {
  present: "set",
  past: "set",
  participle: "setting",
};

export const SAVE_VERBS: Verbs = {
  present: "save",
  past: "saved",
  participle: "saving",
};
