// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action as CoreAction, UnknownAction } from "@reduxjs/toolkit";

import { type Action } from "@/state";

const DRIFT_ACTION_INDICATOR = "DA@";
const DRIFT_PREFIX_SPLITTER = "://";

const sugarType = (type: string, emitter: string): string =>
  DRIFT_ACTION_INDICATOR.concat(emitter, DRIFT_PREFIX_SPLITTER, type);

const desugarType = (type: string): [string, string] => {
  const [prefix, embedded] = type.split(DRIFT_PREFIX_SPLITTER);
  if (embedded == null) return [type, ""];
  const [, winKey] = prefix.split(DRIFT_ACTION_INDICATOR);
  return [embedded, winKey];
};

/**
 * Sugars an action, embedding the window key in the type.
 * @param action - The action to sugar.
 * @param emitter - The window key to embed.
 * @returns - The sugared action.
 */
export const sugar = <A extends CoreAction = UnknownAction>(
  action: A,
  emitter: string,
): A => ({
  ...action,
  type: sugarType(action.type, emitter),
});

/**
 * Desugars an action, extracting the window key from the type.
 * @param action - The action to desugar.
 * @returns - {
 *    emitted: Whether the action was emitted by another window.
 *    emitter: The window key that emitted the action.
 *    action: The desugared action.
 * }
 */
export const desugar = <A extends CoreAction = UnknownAction>(
  action: A | Action,
): {
  emitted: boolean;
  emitter: string;
  action: A | Action;
} => {
  const [type, emitter] = desugarType(action.type);
  return {
    emitted: emitter != null && emitter.length > 0,
    emitter,
    action: { ...action, type },
  };
};
