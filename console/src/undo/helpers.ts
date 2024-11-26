import { type Action } from "@reduxjs/toolkit";

import { type History } from "@/undo/types";

export const parseActions = (
  rawActions: string | string[] | undefined,
  defaultValue: string[] = [],
): string[] => {
  if (Array.isArray(rawActions)) return rawActions;
  if (typeof rawActions === "string") return [rawActions];

  return defaultValue;
};

// isHistory helper: check for a valid history object
export const isHistory = (history: any): history is History<any> =>
  typeof history.present !== "undefined" &&
  typeof history.future !== "undefined" &&
  typeof history.past !== "undefined" &&
  Array.isArray(history.future) &&
  Array.isArray(history.past);

// includeAction helper: whitelist actions to be added to the history
export const includeAction = (
  rawActions: string | string[],
): ((action: Action) => boolean) => {
  const actions = parseActions(rawActions);
  return (action: Action): boolean => actions.indexOf(action.type) >= 0;
};

// excludeAction helper: blacklist actions from being added to the history
export const excludeAction = (
  rawActions: string | string[],
): ((action: Action) => boolean) => {
  const actions = parseActions(rawActions);
  return (action: Action): boolean => actions.indexOf(action.type) < 0;
};

// combineFilters helper: combine multiple filters to one
export const combineFilters = (
  ...filters: Array<
    (action: Action, currentState: any, previousHistory: History<any>) => boolean
  >
): ((action: Action, currentState: any, previousHistory: History<any>) => boolean) =>
  filters.reduce(
    (prev, curr) => (action, currentState, previousHistory) =>
      prev(action, currentState, previousHistory) &&
      curr(action, currentState, previousHistory),
    () => true,
  );

export const groupByActionTypes = (
  rawActions: string | string[],
): ((action: Action) => string | null) => {
  const actions = parseActions(rawActions);
  return (action: Action): string | null =>
    actions.indexOf(action.type) >= 0 ? action.type : null;
};

export const newHistory = <T>(
  past: T[],
  present: T,
  future: T[],
  group: string | null = null,
): History<T> => ({
  past,
  present,
  future,
  group,
  _latestUnfiltered: present,
  index: past.length,
  limit: past.length + future.length + 1,
});
