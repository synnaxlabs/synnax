// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, createSlice } from '@reduxjs/toolkit'
import { configureStore, reducer as driftReducer } from '@synnaxlabs/drift'
import { ElectronRuntime } from '@synnaxlabs/drift/electron'

const counterSlice = createSlice({
  name: 'counter',
  initialState: {
    value: 0
  },
  reducers: {
    incremented: (state) => {
      state.value += 1
    },
    decremented: (state) => {
      state.value -= 1
    }
  }
})

export const { incremented, decremented } = counterSlice.actions

const rootReducer = combineReducers({
  counter: counterSlice.reducer,
  drift: driftReducer
})

export type StoreState = ReturnType<typeof rootReducer>

export default configureStore<StoreState>({
  runtime: new ElectronRuntime(),
  reducer: rootReducer,
  debug: true,
  enablePrerender: true
})
