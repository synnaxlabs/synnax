import {
  type CaseReducerActions,
  createSlice as baseCreateSlice,
  type CreateSliceOptions,
  type Reducer,
  type SliceCaseReducers,
  type SliceSelectors,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { type ActionCreators, newActionCreators } from "@/undo/actions";
import * as debug from "@/undo/debug";
import { isHistory, newHistory, parseActions } from "@/undo/helpers";
import {
  type Action,
  type CreateSliceConfig as Config,
  type History,
} from "@/undo/types";

const emptyHistory = <T>(present: T): History<T> => newHistory<T>([], present, []);

const insert = <T>(
  history: History<T>,
  state: T,
  limit?: number,
  group: string | null = null,
): History<T> => {
  const lengthWithoutFuture = history.past.length + 1;
  debug.log("inserting", state);
  debug.log("new free: ", limit ?? 0 - lengthWithoutFuture);
  const { past, _latestUnfiltered } = history;
  const isHistoryOverflow = limit && limit <= lengthWithoutFuture;
  const pastSliced = past.slice(isHistoryOverflow ? 1 : 0);
  const newPast =
    _latestUnfiltered != null ? [...pastSliced, _latestUnfiltered] : pastSliced;
  return newHistory(newPast, state, [], group);
};

const jumpToFuture = <T>(history: History<T>, index: number): History<T> => {
  if (index < 0 || index >= history.future.length) return history;
  const { past, future, _latestUnfiltered } = history;
  const newPast = [...past, _latestUnfiltered, ...future.slice(0, index)].filter(
    (s) => s != null,
  );
  const newPresent = future[index];
  const newFuture = future.slice(index + 1);
  return newHistory<T>(newPast, newPresent, newFuture);
};

const jumpToPast = <T>(history: History<T>, index: number): History<T> => {
  if (index < 0 || index >= history.past.length) return history;

  const { past, future, _latestUnfiltered } = history;
  const newPast = past.slice(0, index);
  const newFuture = [...past.slice(index + 1), _latestUnfiltered, ...future].filter(
    (s) => s != null,
  );
  const newPresent = past[index];
  return newHistory<T>(newPast, newPresent, newFuture);
};

const jump = <T>(history: History<T>, n: number): History<T> => {
  if (n > 0) return jumpToFuture(history, n - 1);
  if (n < 0) return jumpToPast(history, history.past.length + n);
  return history;
};

const actionTypeAmongClearHistoryType = (
  actionType: string,
  clearHistoryType: string[],
): boolean | string =>
  clearHistoryType.indexOf(actionType) > -1 ? actionType : !actionType;

export const wrapReducer = <
  S = any,
  A extends Action = UnknownAction,
  PreloadedState = S,
>(
  reducer: Reducer<S, A, PreloadedState>,
  name: string = "",
  rawConfig: Config = {},
): Reducer<History<S>> => {
  const config: Required<Config> = {
    limit: 20,
    debug: false,
    filter: () => true,
    groupBy: () => null,
    undoType: `${name}/undo`,
    redoType: `${name}/redo`,
    syncFilter: false,
    ...rawConfig,
    initTypes: parseActions(rawConfig.initTypes, ["@@redux-undo/INIT"]),
    clearHistoryType: parseActions(rawConfig.clearHistoryType, []),
  };

  let last = TimeStamp.now();

  let initialState: History<S>;

  debug.set(true);
  return (state: History<S> = initialState, action: Action): History<S> => {
    debug.start(action, state);

    let history = state;
    if (!initialState) {
      debug.log("history is uninitialized");

      if (state === undefined) {
        const createHistoryAction: Action = { type: "@@redux-undo/CREATE_HISTORY" };
        const start = reducer(state, createHistoryAction as A);

        history = emptyHistory(start);

        debug.log("do not set initialState on probe actions");
        debug.end(history);
        return history;
      }
      if (isHistory(state)) {
        history = initialState = newHistory(state.past, state.present, state.future);
        debug.log(
          "initialHistory initialized: initialState is a history",
          initialState,
        );
      } else {
        history = initialState = emptyHistory(state);
        debug.log(
          "initialHistory initialized: initialState is not a history",
          initialState,
        );
      }
    }

    let res: History<S>;
    switch (action.type) {
      case undefined:
        return history;

      case config.undoType:
        res = jump(history, -1);
        debug.log("perform undo");
        debug.end(res);
        return res;

      case config.redoType:
        res = jump(history, 1);
        debug.log("perform redo");
        debug.end(res);
        return res;

      default: {
        const newPresent = reducer(history.present, action as A);

        if (config.initTypes.some((actionType) => actionType === action.type)) {
          debug.log("reset history due to init action");
          debug.end(initialState);
          return initialState;
        }

        if (actionTypeAmongClearHistoryType(action.type, config.clearHistoryType))
          return emptyHistory(newPresent);

        if (history._latestUnfiltered === newPresent) return history;

        const filtered =
          typeof config.filter === "function" &&
          !config.filter(action, newPresent, history);
        const throttle = TimeStamp.since(last).lessThan(TimeSpan.fromMilliseconds(500));
        if (filtered || throttle) {
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

        history = insert(history, newPresent, config.limit, group);

        debug.log("inserted new state into history");
        debug.end(history);
        return history;
      }
    }
  };
};

interface HistorySlice<
  State,
  CaseReducers extends SliceCaseReducers<State>,
  Name extends string = string,
  ReducerPath extends string = Name,
  Selectors extends SliceSelectors<State> = {},
> {
  name: Name;
  reducer: Reducer<History<State>>;
  actions: CaseReducerActions<CaseReducers, Name> & ActionCreators<Name>;
  caseReducers: CaseReducers;
  getInitialState: () => History<State>;
  selectors: Selectors;
}

// Adjust the UndoCreateSliceOptions type
type UndoCreateSliceOptions<
  State,
  CaseReducers extends SliceCaseReducers<State>,
  Name extends string = string,
  Selectors extends SliceSelectors<State> = {},
  ReducerPath extends string = Name,
> = CreateSliceOptions<State, CaseReducers, Name, ReducerPath, Selectors> & {
  exclude: (keyof CaseReducers)[];
};

// Update the createSlice function
export const createSlice = <
  State,
  CaseReducers extends SliceCaseReducers<State>,
  Name extends string = string,
  Selectors extends SliceSelectors<State> = {},
  ReducerPath extends string = Name,
>({
  exclude,
  ...options
}: UndoCreateSliceOptions<
  State,
  CaseReducers,
  Name,
  Selectors,
  ReducerPath
>): HistorySlice<State, CaseReducers, Name, ReducerPath, Selectors> => {
  const base = baseCreateSlice<State, CaseReducers, Name, Selectors, ReducerPath>(
    options,
  );
  const historic = wrapReducer(base.reducer, options.name, {
    clearHistoryType: exclude.map((key) => `${options.name}/${key.toString()}`),
  });

  return {
    ...base,
    reducer: historic,
    actions: {
      ...base.actions,
      ...newActionCreators(options.name),
    },
  } as unknown as HistorySlice<State, CaseReducers, Name, ReducerPath, Selectors>;
};
