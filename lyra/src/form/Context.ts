// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { type z } from "zod";

import { context } from "@/context";
import { type State } from "@/form/state";
import { type Entry } from "@/form/store";
import { type Status } from "@/status";

export interface RemoveFunc {
  (path: string): void;
}

export interface InsertFunc {
  <E extends Entry>(path: string, value: E, after?: string): void;
}

export interface MoveFunc {
  (path: string, key: string, after?: string): void;
}

/** Options for {@link SetFunc}. */
export interface SetOptions {
  /** Whether to call the form's `onChange`. Defaults to true. */
  notifyOnChange?: boolean;
  /** Whether the write counts as a user edit. Defaults to true. */
  markTouched?: boolean;
}

export interface SetFunc {
  (path: string, value: unknown, options?: SetOptions): void;
}

export interface Listener {
  (): void;
}

export interface BindFunc {
  (props: Listener): destructor.Destructor;
}

/** Whether the form takes edits, or renders flat and inert. */
export type Mode = "normal" | "preview";

/** The form API. Every field hook and component reads it from context. */
export interface ContextValue<Z extends z.ZodType = z.ZodType> {
  mode: Mode;
  /** Subscribes to every change in the form, returning the unsubscribe. */
  bind: BindFunc;
  /** Writes the value at a dot-separated path, validating and notifying. */
  set: SetFunc;
  /** Reads the state at a path: its value, status, and whether it is required. */
  get: typeof State.prototype.getState;
  /** Restores the initial values, or the given ones. */
  reset: (values?: z.infer<Z>) => void;
  /** Removes the object field or the keyed list entry at the path. */
  remove: RemoveFunc;
  /**
   * Inserts an entry into the keyed list at the path, after the entry with the key
   * `after`. An absent `after` inserts first, and one that names no entry inserts last.
   */
  insert: InsertFunc;
  /** Moves the entry with the key after the entry with the key `after`, as insert. */
  move: MoveFunc;
  /**
   * Runs the edits that the function makes with one validation and one notification.
   * `onChange` still runs once for each edit.
   */
  batch: (fn: () => void) => void;
  /** Reads the whole value tree. It does not re-render the caller. */
  value: () => z.infer<Z>;
  /** Validates the whole form, or one subtree, writing statuses onto the fields. */
  validate: (path?: string) => boolean;
  validateAsync: (path?: string) => Promise<boolean>;
  has: (path: string) => boolean;
  setStatus: typeof State.prototype.setStatus;
  clearStatuses: () => void;
  /** Takes the current values as the baseline, clearing the touched flag. */
  setCurrentStateAsInitialValues: () => void;
  getStatuses: () => Status.Crude[];
}

const [Context, useCtx] = context.create<ContextValue | null>({
  defaultValue: null,
  displayName: "Form.Context",
});
export { Context };

/**
 * @returns the enclosing form's {@link ContextValue}, or the override when one is given.
 * @throws {Error} if there is neither an enclosing form nor an override.
 */
export const useContext = <Z extends z.ZodType = z.ZodType>(
  override?: ContextValue<Z>,
  funcName: string = "Form.useContext",
): ContextValue<Z> => {
  const internal = useCtx();
  if (internal == null && override == null)
    throw new Error(`${funcName} must be used within a Form context value`);
  return override ?? (internal as unknown as ContextValue<Z>);
};
