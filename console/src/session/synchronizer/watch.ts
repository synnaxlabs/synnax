// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type Store } from "@reduxjs/toolkit";
import { type destructor } from "@synnaxlabs/x";

/**
 * Runs onChange whenever the selected value changes (Object.is). For use
 * inside a synchronizer's listen.
 * @returns the store unsubscribe.
 */
export const watch = <S, A extends Action, T>(
  store: Store<S, A>,
  select: (state: S) => T,
  onChange: (value: T, prev: T) => void,
): destructor.Destructor => {
  let prev = select(store.getState());
  return store.subscribe(() => {
    const next = select(store.getState());
    if (Object.is(next, prev)) return;
    const last = prev;
    prev = next;
    onChange(next, last);
  });
};
