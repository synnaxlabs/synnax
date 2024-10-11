import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export const SLICE_NAME = "playback";

export interface SliceState {
  cursor: number;
}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  cursor: 0,
};

export interface SetCursorPayload {
  cursor: number;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setCursor: (state, { payload: { cursor } }: PayloadAction<SetCursorPayload>) => {
      state.cursor = cursor;
    },
  },
});

export const { setCursor } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];
