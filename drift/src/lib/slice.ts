import { Action, createSlice, PayloadAction } from '@reduxjs/toolkit';

import { DRIFT_NAME } from './type';
import { KeyedWindowProps, Window, WindowProps } from './window';

type WindowStatus = 'creating' | 'created' | 'closing' | 'closed';

type WindowState = {
  status: WindowStatus;
  processNum: number;
  props: KeyedWindowProps;
};

export interface DriftState {
  key: string;
  windows: Record<string, WindowState>;
}

export interface StoreState {
  drift: DriftState;
}

type MaybeKeyPayload = {
  key?: string;
};

type KeyPayload = {
  key: string;
};

export type CreateWindowPayload = WindowProps;
export type CloseWindowPayload = MaybeKeyPayload;
export type SetWindowKeyPayload = KeyPayload;
export type SetWindowPayload = MaybeKeyPayload & { status: WindowStatus };
export type DriftAction = PayloadAction<
  CreateWindowPayload | CloseWindowPayload | SetWindowPayload | MaybeKeyPayload
>;

export const initialState: DriftState = {
  key: 'main',
  windows: {
    main: {
      processNum: 0,
      status: 'created',
      props: {
        key: 'main',
      },
    },
  },
};

const assertKey = <T extends MaybeKeyPayload>(
  pld: MaybeKeyPayload
): T & KeyPayload => {
  if (pld.key === undefined) {
    throw new Error('drift - bug - key is undefined');
  }
  return pld as T & KeyPayload;
};

export const slice = createSlice({
  name: DRIFT_NAME,
  initialState,
  reducers: {
    setWindowKey: (state, action: PayloadAction<SetWindowKeyPayload>) => {
      const { key } = assertKey<SetWindowKeyPayload>(action.payload);
      state.key = key;
    },
    createWindow: (state, { payload }: PayloadAction<CreateWindowPayload>) => {
      const { key } = payload;
      if (!key) return;
      state.windows[key] = {
        status: 'creating',
        processNum: 0,
        props: payload as KeyedWindowProps,
      };
    },
    setWindowStatus: (
      { windows },
      { payload }: PayloadAction<SetWindowPayload>
    ) => {
      const { key, status } = assertKey<SetWindowPayload>(payload);
      windows[key].status = status;
    },
    closeWindow: (
      { windows },
      { payload }: PayloadAction<CloseWindowPayload>
    ) => {
      const { key } = assertKey<CloseWindowPayload>(payload);
      windows[key].status = 'closing';
    },
    registerProcess: (
      { windows },
      { payload }: PayloadAction<MaybeKeyPayload>
    ) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      windows[key].processNum += 1;
    },
    completeProcess: (
      { windows },
      { payload }: PayloadAction<MaybeKeyPayload>
    ) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      const win = windows[key];
      win.processNum -= 1;
      // If the window is closing and there are no more processes, mark it
      // as closed.
      if (shouldCloseWindow(0, win)) win.status = 'closed';
    },
  },
});

export const {
  createWindow,
  closeWindow,
  setWindowKey,
  setWindowStatus,
  completeProcess,
  registerProcess,
} = slice.actions;

export const isDriftAction = (type: string) => type.startsWith(DRIFT_NAME);

const shouldCloseWindow = (threshold: number, state?: WindowState): boolean =>
  state !== undefined &&
  state.processNum <= threshold &&
  state.status === 'closing';

export const maybeSetWindowKey = <S extends StoreState, A extends Action>(
  window: Window<S, A>,
  action: DriftAction,
  getState: () => S
) => {
  const a = { ...action };
  if (!a.payload.key) {
    if (action.type === createWindow.type) {
      a.payload.key = `window-${
        Object.keys(getState().drift.windows).length + 1
      }`;
    } else {
      a.payload.key = window.key();
    }
  }
  return a;
};

export const executeAction = <S extends StoreState>({
  window,
  action,
  getState,
}: {
  window: Window<S, any>;
  action: DriftAction;
  getState: () => StoreState;
}) => {
  if (!action.payload.key) {
    throw new Error("[drift] - bug - action doesn't have a key");
  }
  const {
    drift: { windows },
  } = getState();
  const { key } = action.payload;

  switch (action.type) {
    case closeWindow.type: {
      const win = windows[key];
      // If no processes are running, close the window immediately.
      // Execute a close request even if we can't find the window in state.
      // This is mainly to deal with redux state being out of sync with the
      // window state.
      if (!win || win.processNum <= 0) window.close(key);
      break;
    }
    case createWindow.type: {
      // If we've already created a window with this key, don't create a new one.
      if (!window.exists(key)) {
        window.createWindow(action.payload as KeyedWindowProps);
      } else {
        window.focus(key);
      }
      break;
    }
    case completeProcess.type: {
      // If no processes are running, close the window. Threshold
      // set at 1 because we haven't yet updated the state to include the last
      // closure.
      if (shouldCloseWindow(1 /* threshold */, windows[key])) window.close(key);
      break;
    }
  }
};
