import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { DRIFT_NAME } from './type';
import { Window, KeyedWindowProps, WindowProps } from './window';

export interface DriftState {
  windows: { [key: string]: KeyedWindowProps };
  numCreated: number;
}

export interface StoreState {
  drift: DriftState;
}

export type CreateWindow = PayloadAction<WindowProps>;
export type CloseWindow = PayloadAction<string | undefined>;

export const initialState: DriftState = {
  numCreated: 1,
  windows: {},
};

export const slice = createSlice({
  name: DRIFT_NAME,
  initialState,
  reducers: {
    createWindow: (state, { payload }: CreateWindow) => {
      const { key } = payload;
      state.numCreated += 1;
      if (key) state.windows[key] = payload as KeyedWindowProps;
    },
    closeWindow: ({ windows }, { payload: key }: CloseWindow) => {
      if (key) delete windows[key];
    },
  },
});

export const { createWindow, closeWindow } = slice.actions;

const isSliceAction = (type: string) => type.startsWith(DRIFT_NAME);

export const executeAction = ({
  window,
  action,
  getState,
}: {
  window: Window;
  action: PayloadAction<unknown>;
  getState: () => StoreState;
}) => {
  if (!isSliceAction(action.type)) return;
  const s = getState();
  switch (action.type) {
    case createWindow.type:
      const { payload: props } = action as CreateWindow;
      const keyedProps = maybeAssignKey(s, props);
      if (!winExists(s, keyedProps.key)) window.createWindow(keyedProps);
      return;
    case closeWindow.type:
      const { payload } = action as CloseWindow;
      const key = payload || window.key();
      if (winExists(s, key)) window.close(key);
      return;
  }
};

const winExists = (state: StoreState, key: string) =>
  state.drift.windows.hasOwnProperty(key);

const maybeAssignKey = (
  state: StoreState,
  props: WindowProps
): KeyedWindowProps => {
  props.key = props.key || `win-${state.drift.numCreated + 1}`;
  return props as KeyedWindowProps;
};
