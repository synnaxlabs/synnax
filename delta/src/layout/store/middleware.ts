// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Middleware, Dispatch, AnyAction } from "@reduxjs/toolkit";

export const layoutRemoverEffectMiddleware =

  factory:
): Middleware<
    DispatchExt,
    S,
    D
  > =>
  (store) =>
  (next) =>
  (action) => {
    const state = next(action);
    if (action.type === "layout/removeEffect") {
      const { effect } = action.payload;
      const state = store.getState();
      const { effects } = state.layout;
      const newEffects = effects.filter((e) => e !== effect);
      store.dispatch(layoutSlice.actions.setEffects(newEffects));
    }
    return state;
  };
