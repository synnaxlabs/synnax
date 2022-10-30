import { createSlice, combineReducers } from '@reduxjs/toolkit';

import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
} from '@synnaxlabs/drift';
import { appWindow } from '@tauri-apps/api/window';

const counterSlice = createSlice({
  name: 'counter',
  initialState: {
    value: 0,
  },
  reducers: {
    incremented: (state) => {
      state.value += 1;
    },
    decremented: (state) => {
      state.value -= 1;
    },
  },
});

export const { incremented, decremented } = counterSlice.actions;

const rootReducer = combineReducers({
  counter: counterSlice.reducer,
  drift: driftReducer,
});

export type StoreState = ReturnType<typeof rootReducer>;

export default configureStore<StoreState>({
  runtime: new TauriRuntime(appWindow),
  reducer: rootReducer,
});
