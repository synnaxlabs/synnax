// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Action, AnyAction, Reducer, enh } from "@reduxjs/toolkit";

export const undoable = <S, A extends Action = AnyAction>(reducer: Reducer<S, A>) => {
  const initialState = {
    past: [],
    present: reducer(undefined, {}),
    future: [],
  };

  return (state = initialState, action: any) => {
    switch(action.)
  };
};
