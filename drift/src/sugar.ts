import type { Action, AnyAction } from "@reduxjs/toolkit";

import { DriftAction } from "@/state";

const DRIFT_ACTION_INDICATOR = "DA@";
const DRIFT_PREFIX_SPLITTER = "://";

const sugarType = (type: string, emitter: string): string =>
  DRIFT_ACTION_INDICATOR.concat(emitter, DRIFT_PREFIX_SPLITTER, type);

const desugarType = (type: string): [string, string] => {
  const [prefix, embedded] = type.split(DRIFT_PREFIX_SPLITTER);
  if (embedded.length === 0) return [type, ""];
  const [, winKey] = prefix.split(DRIFT_ACTION_INDICATOR);
  return [embedded, winKey];
};

/**
 * Sugars an action, embedding the window key in the type.
 * @param action - The action to sugar.
 * @param emitter - The window key to embed.
 * @returns - The sugared action.
 */
export const sugar = <A extends Action = AnyAction>(action: A, emitter: string): A => ({
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
export const desugar = <A extends Action = AnyAction>(
  action: A | DriftAction
): {
  emitted: boolean;
  emitter: string;
  action: A | DriftAction;
} => {
  let emitter: string;
  [action.type, emitter] = desugarType(action.type);
  return { emitted: emitter !== "", emitter, action };
};
