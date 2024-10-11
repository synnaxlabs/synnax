import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, SliceState, StoreState } from "@/playback/slice";

export const selectSlice = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectCursor = (state: StoreState) => selectSlice(state).cursor;

export const useSelectCursor = () =>
  useMemoSelect((state: StoreState) => selectCursor(state), []);
