import {
  type ActionCreatorWithoutPayload,
  type ActionCreatorWithPayload,
  type CaseReducerActions,
} from "@reduxjs/toolkit";

import { type Action } from "@/undo/types";
export const TYPES = {
  UNDO: "UNDO",
  REDO: "REDO",
  JUMP_TO_FUTURE: "JUMP_TO_FUTURE",
  JUMP_TO_PAST: "JUMP_TO_PAST",
  JUMP: "JUMP",
  CLEAR_HISTORY: "CLEAR_HISTORY",
} as const;

export type ActionCreators = {
  undo: ActionCreatorWithoutPayload<string>;
  redo: ActionCreatorWithoutPayload<string>;
  jumpToFuture: ActionCreatorWithPayload<number, string>;
  jumpToPast: ActionCreatorWithPayload<number, string>;
  jump: ActionCreatorWithPayload<number, string>;
  clearHistory: ActionCreatorWithoutPayload<string>;
};

export const actionCreators = <Name extends string>(prefix: Name): ActionCreators => ({
  undo: () => ({ type: `${prefix}/undo`, payload: undefined }),
  redo: (): Action => ({ type: `${prefix}/redo` }),
  jumpToFuture: (index: number): Action => ({
    type: `${prefix}/${TYPES.JUMP_TO_FUTURE}`,
    index,
  }),
  jumpToPast: (index: number): Action => ({
    type: `${prefix}/${TYPES.JUMP_TO_PAST}`,
    index,
  }),
  jump: (index: number): Action => ({ type: `${prefix}/${TYPES.JUMP}`, index }),
  clearHistory: (): Action => ({ type: `${prefix}/${TYPES.CLEAR_HISTORY}` }),
});
