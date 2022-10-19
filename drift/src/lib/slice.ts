import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { DRIFT_NAME } from './actions';
import { Runtime, KeyedWindowProps } from './runtime';

export interface State {
  windows: { [key: string]: KeyedWindowProps };
}

export type SetWindow = PayloadAction<KeyedWindowProps>;

export const initialState: State = {
  windows: {},
};

export const slice = createSlice({
  name: DRIFT_NAME,
  initialState,
  reducers: {
    setWindow: ({ windows }, { payload }: SetWindow) => {
      const { key } = payload;
      if (!key) {
        console.warn('[drift] - no key provided to setWindow');
        return;
      }
      windows[key] = payload;
    },
  },
});

export const { setWindow } = slice.actions;
export const createWindow = setWindow;

const ACTION_TYPES = [setWindow.type];

export const isDriftAction = (type: string) => ACTION_TYPES.includes(type);

export const executeAction = ({
  runtime,
  action,
  getState,
}: {
  runtime: Runtime;
  action: PayloadAction<unknown>;
  getState: () => { drift: State };
}) => {
  switch (action.type) {
    case setWindow.type:
      const { payload: props } = action as SetWindow;
      const w = getWindow(getState().drift, props.key);
      if (!w) runtime.createWindow(props);
  }
};

const getWindow = (state: State, key: string) => state.windows[key];
