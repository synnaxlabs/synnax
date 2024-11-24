import {
  createSlice as baseCreateSlice,
  type CreateSliceOptions,
  type Reducer,
  type Slice,
  type SliceCaseReducers,
  type SliceSelectors,
} from "@reduxjs/toolkit";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { type ActionCreators, actionCreators, TYPES } from "@/undo/actions";
import * as debug from "@/undo/debug";
import { isHistory, newHistory, parseActions } from "@/undo/helpers";
import { type Action, type History, type UndoableConfig as Config } from "@/undo/types";

// createHistory
export const createHistory = <T>(state: T, ignoreInitialState: boolean): History<T> => {
  const history = newHistory<T>([], state, []);
  if (ignoreInitialState) history._latestUnfiltered = null;
  return history;
};

// insert: insert `state` into history
export const insert = <T>(
  history: History<T>,
  state: T,
  limit?: number,
  group: string | null = null,
): History<T> => {
  const lengthWithoutFuture = history.past.length + 1;

  debug.log("inserting", state);
  debug.log("new free: ", limit - lengthWithoutFuture);

  const { past, _latestUnfiltered } = history;
  const isHistoryOverflow = limit && limit <= lengthWithoutFuture;

  const pastSliced = past.slice(isHistoryOverflow ? 1 : 0);
  const newPast =
    _latestUnfiltered != null ? [...pastSliced, _latestUnfiltered] : pastSliced;

  return newHistory(newPast, state, [], group);
};

// jumpToFuture: jump to requested index in future history
export const jumpToFuture = <T>(history: History<T>, index: number): History<T> => {
  if (index < 0 || index >= history.future.length) return history;

  const { past, future, _latestUnfiltered } = history;

  const newPast = [...past, _latestUnfiltered, ...future.slice(0, index)];
  const newPresent = future[index];
  const newFuture = future.slice(index + 1);

  return newHistory(newPast, newPresent, newFuture);
};

// jumpToPast: jump to requested index in past history
export const jumpToPast = <T>(history: History<T>, index: number): History<T> => {
  if (index < 0 || index >= history.past.length) return history;

  const { past, future, _latestUnfiltered } = history;

  const newPast = past.slice(0, index);
  const newFuture = [...past.slice(index + 1), _latestUnfiltered, ...future];
  const newPresent = past[index];

  return newHistory(newPast, newPresent, newFuture);
};

// jump: jump n steps in the past or forward
export const jump = <T>(history: History<T>, n: number): History<T> => {
  if (n > 0) return jumpToFuture(history, n - 1);
  if (n < 0) return jumpToPast(history, history.past.length + n);
  return history;
};

// helper to dynamically match in the reducer's switch-case
export const actionTypeAmongClearHistoryType = (
  actionType: string,
  clearHistoryType: string[],
): boolean | string =>
  clearHistoryType.indexOf(actionType) > -1 ? actionType : !actionType;

// redux-undo higher order reducer
export const wrapReducer = <T>(
  reducer: Reducer<T>,
  name: string = "",
  rawConfig: Config = {},
): Reducer<T> => {
  const config: Required<Config> = {
    limit: undefined,
    filter: () => true,
    groupBy: () => null,
    undoType: `${name}/undo`,
    redoType: `${name}/redo`,
    jumpToPastType: `${name}/jumpToPast`,
    jumpToFutureType: `${name}/jumpToFuture`,
    jumpType: `${name}/jumpType`,
    neverSkipReducer: false,
    ignoreInitialState: false,
    syncFilter: false,
    ...rawConfig,
    initTypes: parseActions(rawConfig.initTypes, ["@@redux-undo/INIT"]),
    clearHistoryType: parseActions(rawConfig.clearHistoryType, [TYPES.CLEAR_HISTORY]),
  };

  let last = TimeStamp.now();

  // Allows the user to call the reducer with redux-undo specific actions
  const skipReducer = config.neverSkipReducer
    ? (res: History<T>, action: Action, ...slices: any[]): History<T> => ({
        ...res,
        present: reducer(res.present, action, ...slices),
      })
    : (res: History<T>): History<T> => res;

  let initialState: History<T>;

  return (
    state: History<T> = initialState,
    action: Action,
    ...slices: any[]
  ): History<T> => {
    debug.start(action, state);

    let history = state;
    if (!initialState) {
      debug.log("history is uninitialized");

      if (state === undefined) {
        const createHistoryAction: Action = { type: "@@redux-undo/CREATE_HISTORY" };
        const start = reducer(state, createHistoryAction, ...slices);

        history = createHistory(start, config.ignoreInitialState);

        debug.log("do not set initialState on probe actions");
        debug.end(history);
        return history;
      }
      if (isHistory(state)) {
        history = initialState = config.ignoreInitialState
          ? state
          : newHistory(state.past, state.present, state.future);
        debug.log(
          "initialHistory initialized: initialState is a history",
          initialState,
        );
      } else {
        history = initialState = createHistory(state, config.ignoreInitialState);
        debug.log(
          "initialHistory initialized: initialState is not a history",
          initialState,
        );
      }
    }

    let res: History<T>;
    switch (action.type) {
      case undefined:
        return history;

      case config.undoType:
        res = jump(history, -1);
        debug.log("perform undo");
        debug.end(res);
        return skipReducer(res, action, ...slices);

      case config.redoType:
        res = jump(history, 1);
        debug.log("perform redo");
        debug.end(res);
        return skipReducer(res, action, ...slices);

      case config.jumpToPastType:
        res = jumpToPast(history, action.index!);
        debug.log(`perform jumpToPast to ${action.index}`);
        debug.end(res);
        return skipReducer(res, action, ...slices);

      case config.jumpToFutureType:
        res = jumpToFuture(history, action.index!);
        debug.log(`perform jumpToFuture to ${action.index}`);
        debug.end(res);
        return skipReducer(res, action, ...slices);

      case config.jumpType:
        res = jump(history, action.index!);
        debug.log(`perform jump to ${action.index}`);
        debug.end(res);
        return skipReducer(res, action, ...slices);

      case actionTypeAmongClearHistoryType(action.type, config.clearHistoryType):
        res = createHistory(history.present, config.ignoreInitialState);
        debug.log("perform clearHistory");
        debug.end(res);
        return skipReducer(res, action, ...slices);

      default: {
        const newPresent = reducer(history.present, action, ...slices);

        if (config.initTypes.some((actionType) => actionType === action.type)) {
          debug.log("reset history due to init action");
          debug.end(initialState);
          return initialState;
        }

        if (history._latestUnfiltered === newPresent)
          // Don't handle this action. Do not call debug.end here,
          // because this action should not produce side effects to the console
          return history;

        const filtered =
          typeof config.filter === "function" &&
          !config.filter(action, newPresent, history);
        const throttle = TimeStamp.since(last).lessThan(TimeSpan.fromSeconds(1));
        if (filtered || throttle) {
          // if filtering an action, merely update the present
          const filteredState = newHistory(
            history.past,
            newPresent,
            history.future,
            history.group,
          );
          if (!config.syncFilter)
            filteredState._latestUnfiltered = history._latestUnfiltered;

          debug.log("filter ignored action, not storing it in past");
          debug.end(filteredState);
          return filteredState;
        }
        last = TimeStamp.now();

        const group = config.groupBy(action, newPresent, history);
        if (group != null && group === history.group) {
          // if grouping with the previous action, only update the present
          const groupedState = newHistory(
            history.past,
            newPresent,
            history.future,
            history.group,
          );
          debug.log("groupBy grouped the action with the previous action");
          debug.end(groupedState);
          return groupedState;
        }

        // If the action wasn't filtered or grouped, insert normally
        history = insert(history, newPresent, config.limit, group);

        debug.log("inserted new state into history");
        debug.end(history);
        return history;
      }
    }
  };
};

export const createSlice = <
  State,
  CaseReducers extends SliceCaseReducers<State>,
  Name extends string,
  Selectors extends SliceSelectors<State>,
  ReducerPath extends string = Name,
>(
  options: CreateSliceOptions<State, CaseReducers, Name, ReducerPath, Selectors>,
): Slice<State, CaseReducers & ActionCreators, Name, ReducerPath, Selectors> => {
  const base = baseCreateSlice(options);
  const historic = wrapReducer(base.reducer, options.name);
  const r = {
    ...base,
    actions: {
      ...base.actions,
      ...actionCreators(options.name),
    },
    reducer: historic,
  };
  return r;
};
