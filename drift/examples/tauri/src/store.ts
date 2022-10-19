import { createSlice, combineReducers } from '@reduxjs/toolkit';
import { slice, TauriRuntime, configureStore } from '@synnaxlabs/drift';

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

const store = configureStore({
  runtime: new TauriRuntime(),
  reducer: combineReducers({
    counter: counterSlice.reducer,
    drift: slice.reducer,
  }),
});

export default store;
