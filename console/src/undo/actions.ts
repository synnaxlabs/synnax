// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ActionCreatorWithPayload } from "@reduxjs/toolkit";

export type ActionCreators<Name extends string> = {
  undo: ActionCreatorWithPayload<undefined, `${Name}/undo`>;
  redo: ActionCreatorWithPayload<undefined, `${Name}/redo`>;
  clearHistory: ActionCreatorWithPayload<undefined, `${Name}/clearHistory`>;
};

const newActionCreator = <P, T extends string>(
  type: T,
): ActionCreatorWithPayload<P, T> => {
  const base = (payload: P) => ({ type, payload });
  base.type = type;
  base.match = (action: unknown): action is { type: T; payload: P } => {
    if (typeof action === "object" && action !== null && "type" in action)
      return action.type === type;
    return false;
  };
  return base;
};

/**
 * @returns a new set of action creators for an undo-enhanced reducer. The actions will
 * be prefixed with the name of the slice.
 */
export const newActionCreators = <Name extends string>(
  prefix: Name,
): ActionCreators<Name> => ({
  undo: newActionCreator<any, `${Name}/undo`>(`${prefix}/undo`),
  redo: newActionCreator<any, `${Name}/redo`>(`${prefix}/redo`),
  clearHistory: newActionCreator<any, `${Name}/clearHistory`>(`${prefix}/clearHistory`),
});
