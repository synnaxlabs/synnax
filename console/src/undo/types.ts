// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * History tracks the past, present, and future state of the undo history.
 */
export interface History<T> {
  /** The past states of the history. */
  past: T[];
  /** The present state of the history. */
  present: T;
  /** The future states of the history. */
  future: T[];
  /** The latest state of the history before it was filtered. */
  _latestUnfiltered: T | null;
  /** The group of the history. */
  group: string | null;
  /** The index of the present state in the past history. */
  index: number;
  /** The limit of the history. */
  limit: number;
}

export interface Action {
  type: string;
  index?: number;
}

export interface CreateSliceConfig {
  /** Whether to enable debug logging. */
  debug?: boolean;
  /** The maximum number of states to store in the history. */
  limit?: number;
  /** A function that determines whether an action should be included in the history. */
  filter?: (action: Action, currentState: any, history: History<any>) => boolean;
  /** A function that determines the group of an action. */
  groupBy?: (action: Action, currentState: any, history: History<any>) => string | null;
  /** The type of the undo action. */
  undoType?: string;
  /** The type of the redo action. */
  redoType?: string;
  /** Whether to sync the filter. */
  syncFilter?: boolean;
  /** The types of the initial states. */
  initTypes?: string[];
  /** The type of the clear history action. */
  clearHistoryType?: string[];
}
