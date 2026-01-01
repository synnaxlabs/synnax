// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type state } from "@/state";

/** Options to control async operations. */
export interface FetchOptions {
  /** Optional AbortSignal to cancel the operation */
  signal?: AbortSignal;
}

/**
 * Represents a serializable data shape used throughout the flux query system.
 * This is the base type for all query parameters, request data, and response data.
 *
 * Shapes must be serializable objects to ensure proper comparison, memoization,
 * and transmission across network boundaries.
 *
 * @example
 * ```typescript
 * interface UserQuery extends Shape {
 *   userId: number;
 *   includeProfile?: boolean;
 * }
 *
 * interface UserData extends Shape {
 *   id: number;
 *   name: string;
 *   email: string;
 * }
 * ```
 */
export type Shape = state.State;

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
